package permissions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	performance "github.com/pancake-lee/pgo/cmd/pgo/performance/core"
)

// validateFixture 确认清单包含完整角色、权限与读组，避免使用部分批次。
func validateFixture(manifest *Manifest) error {
	scale := manifest.Scale
	if !manifest.Ready || scale.Projects < 1 || scale.Actions < 1 ||
		scale.RolesPerGroup < 1 || scale.UsersPerGroup < 1 ||
		scale.Roles < 2*scale.RolesPerGroup ||
		len(manifest.ProjectList) != scale.Projects ||
		len(manifest.UserList) != scale.Projects*2*scale.UsersPerGroup {
		return errors.New("permission fixture is incomplete; use cleanup for partial batches")
	}
	expectedCountMap := map[string]int{
		"project":                    scale.Projects,
		"user-role":                  scale.Projects * scale.Roles,
		"user-role-permission-assoc": scale.Projects * scale.Roles * scale.Actions,
		"user":                       1 + scale.Projects*2*scale.UsersPerGroup,
		"user-project-assoc":         scale.Projects * 2 * scale.UsersPerGroup,
		"user-role-assoc":            scale.Projects * 2 * scale.UsersPerGroup * scale.RolesPerGroup,
	}
	for kind, count := range expectedCountMap {
		idMap := make(map[int32]bool)
		for _, id := range manifest.RecordIDMap[kind] {
			if id <= 0 || idMap[id] {
				return fmt.Errorf("invalid or duplicate %s record ID", kind)
			}
			idMap[id] = true
		}
		if len(idMap) != count {
			return fmt.Errorf("%s manifest record count is incomplete", kind)
		}
	}
	for _, project := range manifest.ProjectList {
		if project.ID <= 0 || len(project.RoleIDList) != scale.Roles ||
			len(project.PermissionIDList) != scale.Roles || len(project.VersionList) != scale.Actions {
			return errors.New("permission fixture project is incomplete")
		}
		for _, idList := range project.PermissionIDList {
			if len(idList) != scale.Actions {
				return errors.New("permission ID list is incomplete")
			}
			for _, id := range idList {
				if id <= 0 {
					return errors.New("invalid permission ID")
				}
			}
		}
	}
	for _, user := range manifest.UserList {
		if user.ID <= 0 || user.Token == "" || user.Project < 0 ||
			user.Project >= scale.Projects || (user.Group != "hot" && user.Group != "control") {
			return errors.New("permission fixture user is incomplete")
		}
	}
	return nil
}

// verifyFixture 抽查每项目两个读组的最终权限与跨项目隔离。
func verifyFixture(ctx context.Context, client *apiClient, manifest *Manifest,
) error {
	seenMap := make(map[string]bool)
	for _, user := range manifest.UserList {
		key := fmt.Sprintf("%d/%s", user.Project, user.Group)
		if seenMap[key] {
			continue
		}
		seenMap[key] = true
		project := manifest.ProjectList[user.Project]
		permissionMap, err := client.getPermissions(ctx, user, project.ID)
		if err != nil {
			return err
		}
		if len(permissionMap) != manifest.Scale.Actions {
			return fmt.Errorf("%w: %s permission count: got %d, want %d", performance.ErrDataMismatch, key, len(permissionMap), manifest.Scale.Actions)
		}
		for action := 0; action < manifest.Scale.Actions; action++ {
			version := 0
			if user.Group == "hot" {
				version = project.VersionList[action]
			}
			want := permissionPath(user.Project, user.Group, action, version)
			if permissionMap[fmt.Sprintf("action_%03d", action)] != want {
				return fmt.Errorf("%w: %s action %d has unexpected path", performance.ErrDataMismatch, key, action)
			}
		}
		if manifest.Scale.Projects > 1 {
			otherProject := manifest.ProjectList[(user.Project+1)%manifest.Scale.Projects]
			otherMap, err := client.getPermissions(ctx, user, otherProject.ID)
			if err != nil {
				return err
			}
			if len(otherMap) != 0 {
				return fmt.Errorf("%w: permissions leaked across projects", performance.ErrDataMismatch)
			}
		}
	}
	return nil
}

// writeTargets 生成携带各用户自身令牌的 Vegeta GET 目标。
func writeTargets(path string, manifest *Manifest, group string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	for _, user := range manifest.UserList {
		if user.Group != group {
			continue
		}
		query := url.Values{}
		query.Set("userID", strconv.Itoa(int(user.ID)))
		query.Set("projectID", strconv.Itoa(int(manifest.ProjectList[user.Project].ID)))
		target := map[string]any{
			"method": http.MethodGet,
			"url":    manifest.BaseURL + "/user/permissions?" + query.Encode(),
			"header": map[string][]string{"Authorization": {"Bearer " + user.Token}},
		}
		err = encoder.Encode(target)
		if err != nil {
			return errors.Join(err, file.Close())
		}
	}
	return file.Close()
}

// runPair 同时运行两个读组，任一执行器失败时取消另一个。
func runPair(ctx context.Context, run func(context.Context, string) error) error {
	pairContext, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	var mu sync.Mutex
	var pairErr error
	for _, group := range []string{"hot", "control"} {
		wg.Add(1)
		go func(group string) {
			defer wg.Done()
			err := run(pairContext, group)
			if err != nil {
				cancel()
				mu.Lock()
				pairErr = errors.Join(pairErr, err)
				mu.Unlock()
			}
		}(group)
	}
	wg.Wait()
	return pairErr
}

// writeResult 保存管理员请求的负载侧完成时刻、耗时与结果。
type writeResult struct {
	Started  time.Time
	Duration time.Duration
	Success  bool
}

// runWriter 按单条请求速率修改共享角色，停止时完成当前动作的角色轮次。
func runWriter(ctx context.Context, stop <-chan struct{}, client *apiClient,
	manifest *Manifest, rps int,
) ([]writeResult, error) {
	var resultList []writeResult
	interval := time.Second / time.Duration(rps)
	for cycle := 0; ; cycle++ {
		select {
		case <-stop:
			return resultList, nil
		case <-ctx.Done():
			return resultList, ctx.Err()
		default:
		}
		projectIndex := cycle % manifest.Scale.Projects
		action := (cycle / manifest.Scale.Projects) % manifest.Scale.Actions
		project := &manifest.ProjectList[projectIndex]
		version := 1 - project.VersionList[action]
		for role := 0; role < manifest.Scale.RolesPerGroup; role++ {
			started := time.Now()
			err := client.updatePermission(ctx, project.PermissionIDList[role][action],
				permissionPath(projectIndex, "hot", action, version))
			resultList = append(resultList, writeResult{started, time.Since(started), err == nil})
			if err != nil {
				return resultList, err
			}
			wait := interval - time.Since(started)
			if wait > 0 {
				timer := time.NewTimer(wait)
				select {
				case <-timer.C:
				case <-ctx.Done():
					timer.Stop()
					return resultList, ctx.Err()
				}
			}
		}
		project.VersionList[action] = version
	}
}

// writeWriterReport 汇总测量窗口中的管理员负载，排除预热与收尾请求。
func writeWriterReport(path string, resultList []writeResult,
	begin, end time.Time,
) error {
	var latencyList []time.Duration
	var success int
	for _, result := range resultList {
		if result.Started.Before(begin) || !result.Started.Before(end) {
			continue
		}
		latencyList = append(latencyList, result.Duration)
		if result.Success {
			success++
		}
	}
	sort.Slice(latencyList, func(i, j int) bool { return latencyList[i] < latencyList[j] })
	quantile := func(fraction float64) time.Duration {
		if len(latencyList) == 0 {
			return 0
		}
		return latencyList[int(float64(len(latencyList)-1)*fraction)]
	}
	count := len(latencyList)
	ratio := float64(0)
	if count > 0 {
		ratio = float64(success) / float64(count)
	}
	report := fmt.Sprintf("Requests %d\nThroughput %.2f/s\nSuccess %.2f%%\nP50 %s\nP95 %s\nP99 %s\n",
		count, float64(success)/end.Sub(begin).Seconds(), ratio*100,
		quantile(.50), quantile(.95), quantile(.99))
	return os.WriteFile(path, []byte(report), 0o600)
}

// RunLoad 在同一批次上同时执行两个读组与管理员修改。
func (preparer *preparer) RunLoad(ctx context.Context, runner *performance.Runner,
	config performance.Config, targetPath string,
) (runErr error) {
	if preparer.opt.PrepareOnly {
		return nil
	}
	err := os.MkdirAll(config.OutputDir, 0o700)
	if err != nil {
		return err
	}
	groupConfig := func(group string) performance.Config {
		stream := config
		stream.RPS = config.RPS / 2
		if group == "hot" {
			stream.RPS += config.RPS % 2
		}
		stream.OutputDir = filepath.Join(config.OutputDir, group)
		return stream
	}
	warmupStarted := time.Now()
	warmupDuration, plannedStart := performance.WarmupWindow(warmupStarted,
		30*time.Second)
	stopWarmupWriter := make(chan struct{})
	warmupWriterDone := make(chan struct{})
	var stopOnce sync.Once
	stopWarmup := func() { stopOnce.Do(func() { close(stopWarmupWriter) }) }
	warmupTimer := time.AfterFunc(warmupDuration, stopWarmup)
	var writerErr error
	go func() {
		defer close(warmupWriterDone)
		_, writerErr = runWriter(ctx, stopWarmupWriter, preparer.client,
			preparer.manifest, preparer.opt.WriteRPS)
	}()
	preparer.info("warming up permission reads and writes",
		"duration", warmupDuration,
		"plannedStartUTC", plannedStart.UTC().Format(time.RFC3339),
		"readRPS", config.RPS, "writeRPS", preparer.opt.WriteRPS)
	err = runPair(ctx, func(pairContext context.Context, group string) error {
		return runner.RunWarmup(pairContext, groupConfig(group),
			filepath.Join(filepath.Dir(targetPath), group+"-targets.jsonl"),
			warmupDuration-time.Since(warmupStarted))
	})
	// 预热失败时立即停止写入；正常时等到预定停止点，完成当前角色轮次。
	if err != nil {
		warmupTimer.Stop()
		stopWarmup()
	}
	<-warmupWriterDone
	warmupTimer.Stop()
	err = errors.Join(err, writerErr)
	var resultList []writeResult
	if err == nil {
		err = runner.WaitForMeasured(ctx, plannedStart)
	}
	var stopSampling func() error
	closeSampling := func() error {
		if stopSampling == nil {
			return nil
		}
		stop := stopSampling
		stopSampling = nil
		err := stop()
		if err == nil {
			preparer.info("runtime sampling disabled")
		}
		return err
	}
	defer func() { runErr = errors.Join(runErr, closeSampling()) }()
	if err == nil && config.Sampling {
		var samplingDuration time.Duration
		stopSampling, samplingDuration, err = performance.StartSampling(ctx, config)
		if err == nil {
			preparer.info("runtime sampling enabled", "duration", samplingDuration)
		}
	}
	begin := time.Now()
	end := begin
	if err == nil {
		preparer.info("permission measured window",
			"startUTC", begin.UTC().Format(time.RFC3339),
			"readRPS", config.RPS, "writeRPS", preparer.opt.WriteRPS)
		stopWriter := make(chan struct{})
		writerDone := make(chan struct{})
		go func() {
			defer close(writerDone)
			resultList, writerErr = runWriter(ctx, stopWriter, preparer.client,
				preparer.manifest, preparer.opt.WriteRPS)
		}()
		err = runPair(ctx, func(pairContext context.Context, group string) error {
			return runner.RunMeasured(pairContext, groupConfig(group),
				filepath.Join(filepath.Dir(targetPath), group+"-targets.jsonl"))
		})
		end = time.Now()
		close(stopWriter)
		<-writerDone
	}
	err = errors.Join(err, closeSampling())
	if writerErr != nil {
		preparer.manifest.Ready = false
	}
	var reportErr error
	if end.After(begin) {
		reportErr = writeWriterReport(
			filepath.Join(config.OutputDir, "writer-report.txt"),
			resultList, begin, end)
	}
	err = errors.Join(err, writerErr, reportErr,
		saveManifest(preparer.manifestPath, preparer.manifest))
	preparer.info("permission window completed",
		"endUTC", end.UTC().Format(time.RFC3339), "outputDir", config.OutputDir)
	if err != nil {
		return err
	}
	return verifyFixture(ctx, preparer.client, preparer.manifest)
}

// clearLoadReports 清除上次负载报告，保留准备数据、目标文件和用户截图。
func clearLoadReports(outputDir string) error {
	entryList, err := os.ReadDir(outputDir)
	if err != nil {
		return err
	}
	for _, entry := range entryList {
		name := entry.Name()
		remove := name == "00-auto-run.json" || name == "writer-report.txt"
		if entry.IsDir() {
			remove = name == "hot" || name == "control"
			if strings.HasPrefix(name, "rps-") {
				value, parseErr := strconv.Atoi(strings.TrimPrefix(name, "rps-"))
				remove = parseErr == nil && value > 0
			}
		}
		if remove {
			err = os.RemoveAll(filepath.Join(outputDir, name))
			if err != nil {
				return err
			}
		}
	}
	return nil
}
