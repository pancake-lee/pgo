package permissions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	klog "github.com/go-kratos/kratos/v2/log"
	performance "github.com/pancake-lee/pgo/cmd/pgo/performance/core"
	"github.com/pancake-lee/pgo/internal/pkg/api"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// fixtureAPI 模拟现有 HTTP 契约与关系，验证批次、权限和清理行为。
type fixtureAPI struct {
	mu                  sync.Mutex
	nextID              int32
	recordMap           map[string]map[int32]map[string]any
	updates             int
	samplingRequests    []string
	samplingUpdateCount int
	stop                func()
	failedCreates       bool
	failedDeletes       bool
	failedReads         bool
	createdCount        int
}

// newFixtureAPI 创建不依赖真实服务或数据库的 HTTP 回归环境。
func newFixtureAPI(t *testing.T) (*fixtureAPI, *httptest.Server) {
	t.Helper()
	fixture := &fixtureAPI{recordMap: make(map[string]map[int32]map[string]any)}
	server := httptest.NewServer(http.HandlerFunc(fixture.serveHTTP))
	t.Cleanup(server.Close)
	return fixture, server
}

// addRecord 将一条模拟记录保存并返回主键。
func (fixture *fixtureAPI) addRecord(kind string, record map[string]any) int32 {
	fixture.nextID++
	fixture.createdCount++
	if fixture.recordMap[kind] == nil {
		fixture.recordMap[kind] = make(map[int32]map[string]any)
	}
	record["ID"] = float64(fixture.nextID)
	fixture.recordMap[kind][fixture.nextID] = record
	return fixture.nextID
}

// readID 读取 JSON 数字类型的主键或关联字段。
func readID(record map[string]any, field string) int32 {
	value, _ := record[field].(float64)
	return int32(value)
}

// serveHTTP 使用真实 Proto 校验创建请求，并模拟查询、更新和删除。
func (fixture *fixtureAPI) serveHTTP(writer http.ResponseWriter, request *http.Request) {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	writer.Header().Set("Content-Type", "application/json")
	if request.URL.Path == "/debug/pprof/sampling" {
		fixture.samplingRequests = append(fixture.samplingRequests, request.Method)
		if request.Method == http.MethodPost {
			fixture.samplingUpdateCount = fixture.updates
			_, _ = io.WriteString(writer, `{"id":1,"seconds":1}`)
		} else {
			writer.WriteHeader(http.StatusNoContent)
		}
		return
	}
	kind := strings.TrimPrefix(request.URL.Path, "/")
	if kind == "user/token" {
		var login struct{ UserName string }
		_ = json.NewDecoder(request.Body).Decode(&login)
		var id int32
		for candidate, record := range fixture.recordMap["user"] {
			if record["userName"] == login.UserName {
				id = candidate
			}
		}
		if id == 0 {
			id = fixture.addRecord("user", map[string]any{"userName": login.UserName})
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"user": fixture.recordMap["user"][id], "token": fmt.Sprintf("token-%d", id),
		})
		return
	}
	if request.Header.Get("Authorization") == "" {
		writer.WriteHeader(401)
		return
	}
	if kind == "user/permissions" {
		userID, _ := strconv.Atoi(request.URL.Query().Get("userID"))
		projectID, _ := strconv.Atoi(request.URL.Query().Get("projectID"))
		if request.Header.Get("Authorization") != fmt.Sprintf("Bearer token-%d", userID) {
			writer.WriteHeader(403)
			return
		}
		roleMap := make(map[int32]bool)
		for _, assoc := range fixture.recordMap["user-role-assoc"] {
			roleID := readID(assoc, "roleID")
			if readID(assoc, "userID") == int32(userID) &&
				readID(fixture.recordMap["user-role"][roleID], "projID") == int32(projectID) {
				roleMap[roleID] = true
			}
		}
		idList := make([]int, 0)
		for id := range fixture.recordMap["user-role-permission-assoc"] {
			idList = append(idList, int(id))
		}
		sort.Ints(idList)
		permissionMap := make(map[string]string)
		for _, id := range idList {
			record := fixture.recordMap["user-role-permission-assoc"][int32(id)]
			if roleMap[readID(record, "roleID")] {
				permissionMap[record["action"].(string)] = record["pathPattern"].(string)
			}
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{"actionToPathPattern": permissionMap})
		return
	}
	switch request.Method {
	case http.MethodPost:
		content, _ := io.ReadAll(request.Body)
		var message proto.Message
		switch kind {
		case "project":
			message = new(api.AddProjectRequest)
		case "user-role":
			message = new(api.AddUserRoleRequest)
		case "user-role-assoc":
			message = new(api.AddUserRoleAssocRequest)
		case "user-project-assoc":
			message = new(api.AddUserProjectAssocRequest)
		case "user-role-permission-assoc":
			message = new(api.AddUserRolePermissionAssocRequest)
		}
		if message == nil || protojson.Unmarshal(content, message) != nil {
			writer.WriteHeader(400)
			return
		}
		if fixture.failedCreates && kind == "user-role-permission-assoc" {
			writer.WriteHeader(503)
			return
		}
		var body map[string]map[string]any
		_ = json.Unmarshal(content, &body)
		record := body[envelope(kind)]
		fixture.addRecord(kind, record)
		_ = json.NewEncoder(writer).Encode(map[string]any{envelope(kind): record})
	case http.MethodPatch:
		content, _ := io.ReadAll(request.Body)
		var message api.UpdateUserRolePermissionAssocRequest
		if protojson.Unmarshal(content, &message) != nil {
			writer.WriteHeader(400)
			return
		}
		record := fixture.recordMap[kind][message.UserRolePermissionAssoc.ID]
		record["pathPattern"] = message.UserRolePermissionAssoc.PathPattern
		fixture.updates++
		if fixture.stop != nil && fixture.updates == 1 {
			fixture.stop()
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{envelope(kind): record})
	case http.MethodDelete:
		if fixture.failedDeletes {
			writer.WriteHeader(503)
			return
		}
		for _, text := range request.URL.Query()["IDList"] {
			id, _ := strconv.Atoi(text)
			delete(fixture.recordMap[kind], int32(id))
		}
		_, _ = io.WriteString(writer, "{}")
	case http.MethodGet:
		if fixture.failedReads {
			writer.WriteHeader(503)
			return
		}
		recordList := make([]map[string]any, 0)
		for _, record := range fixture.recordMap[kind] {
			recordList = append(recordList, record)
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{envelope(kind) + "List": recordList})
	default:
		writer.WriteHeader(405)
	}
}

// TestHTTPFixtureWriterAndCleanup 验证重叠权限、停止完整轮次、项目隔离与精确清理。
func TestHTTPFixtureWriterAndCleanup(t *testing.T) {
	fixture, server := newFixtureAPI(t)
	path := filepath.Join(t.TempDir(), "manifest.json")
	manifest, err := prepareFixture(t.Context(), server.URL, path,
		fixtureScale{2, 4, 3, 2, 2}, klog.NewStdLogger(io.Discard))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.RecordIDMap["user-role-permission-assoc"]) != 24 || len(manifest.UserList) != 8 {
		t.Fatal("unexpected fixture size")
	}
	if err = validateFixture(manifest); err != nil {
		t.Fatal(err)
	}
	client := newAPIClient(server.URL, manifest.Admin.Token)
	t.Cleanup(client.httpClient.CloseIdleConnections)
	if err = verifyFixture(t.Context(), client, manifest); err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	fixture.mu.Lock()
	fixture.stop = func() { close(stop) }
	fixture.mu.Unlock()
	resultList, err := runWriter(t.Context(), stop, client, manifest, 0.8)
	if err != nil {
		t.Fatal(err)
	}
	if len(resultList) == 2 && resultList[1].Started.Sub(resultList[0].Started) < 1250*time.Millisecond {
		t.Fatal("fractional writer rate exceeded 0.8 RPS")
	}
	fixture.mu.Lock()
	updates := fixture.updates
	fixture.mu.Unlock()
	if len(resultList) != 2 || updates != 2 || manifest.ProjectList[0].VersionList[0] != 1 {
		t.Fatal("writer did not finish the shared-role round")
	}
	if err = verifyFixture(t.Context(), client, manifest); err != nil {
		t.Fatal(err)
	}
	for _, group := range []string{"hot", "control"} {
		targetPath := filepath.Join(t.TempDir(), group+".jsonl")
		if err = writeTargets(targetPath, manifest, group); err != nil {
			t.Fatal(err)
		}
		content, _ := os.ReadFile(targetPath)
		if strings.Count(string(content), "\n") != 4 {
			t.Fatal("incorrect target count")
		}
	}
	// 模拟响应已成功但最后一批主键尚未持久化，清理须按批次关系找回。
	manifest.RecordIDMap["user-role-permission-assoc"] = manifest.RecordIDMap["user-role-permission-assoc"][:20]
	fixture.mu.Lock()
	unrelated := fixture.addRecord("project", map[string]any{"projName": "existing_project"})
	fixture.mu.Unlock()
	if err = cleanupFixture(t.Context(), client, manifest, path); err != nil {
		t.Fatal(err)
	}
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	for kind, recordMap := range fixture.recordMap {
		if kind == "project" {
			if len(recordMap) != 1 || recordMap[unrelated] == nil {
				t.Fatal("unrelated project deleted")
			}
		} else if len(recordMap) != 0 {
			t.Fatalf("remaining %s records: %d", kind, len(recordMap))
		}
	}
	if _, err = loadManifest(path); err == nil {
		t.Fatal("cleaned manifest accepted")
	}
}

// TestFailedPreparationRetainsCleanableBatch 验证 HTTP 创建失败时可清理成功记录。
func TestFailedPreparationRetainsCleanableBatch(t *testing.T) {
	fixture, server := newFixtureAPI(t)
	fixture.mu.Lock()
	fixture.failedCreates = true
	fixture.mu.Unlock()
	path := filepath.Join(t.TempDir(), "manifest.json")
	manifest, err := prepareFixture(t.Context(), server.URL, path,
		fixtureScale{1, 2, 2, 1, 1}, klog.NewStdLogger(io.Discard))
	if err == nil || manifest == nil || manifest.Ready {
		t.Fatal("expected partial preparation failure")
	}
	loaded, err := loadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	client := newAPIClient(server.URL, loaded.Admin.Token)
	t.Cleanup(client.httpClient.CloseIdleConnections)
	if err = cleanupFixture(t.Context(), client, loaded, path); err != nil {
		t.Fatal(err)
	}
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	for _, recordMap := range fixture.recordMap {
		if len(recordMap) != 0 {
			t.Fatal("partial batch left records behind")
		}
	}
}

// TestWriterReportExcludesWarmupAndDrain 验证写负载统计仅包含正式窗口。
func TestWriterReportExcludesWarmupAndDrain(t *testing.T) {
	begin := time.Now()
	path := filepath.Join(t.TempDir(), "report.txt")
	resultList := []writeResult{
		{begin.Add(-time.Second), time.Second, true},
		{begin.Add(time.Second), time.Millisecond, true},
		{begin.Add(2 * time.Second), 2 * time.Millisecond, false},
		{begin.Add(11 * time.Second), time.Second, true},
	}
	err := writeWriterReport(path, resultList, begin, begin.Add(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(path)
	if !strings.Contains(string(content), "Requests 2\n") || !strings.Contains(string(content), "Success 50.00%") {
		t.Fatal(string(content))
	}
}

// TestPermissionCommandDefaults 验证首次参数与专属清理入口。
func TestPermissionCommandDefaults(t *testing.T) {
	command := Entrypoint.NewCobraCommand()
	if command.Flag("rps").DefValue != "20" || command.Flag("duration").DefValue != "120s" {
		t.Fatal("permission defaults differ from baseline")
	}
	if command.Flag("keep-data").DefValue != "true" || len(command.Commands()) != 1 {
		t.Fatal("missing persistent batch or cleanup command")
	}
}

// useSmallScale 临时缩小固定规模，保持完整 HTTP 业务关系。
func useSmallScale(t *testing.T) {
	t.Helper()
	previous := defaultScale
	defaultScale = fixtureScale{1, 4, 2, 1, 1}
	t.Cleanup(func() { defaultScale = previous })
}

// prepareSmallPermissionBatch 执行真实准备入口并为 HTTP 客户端安排收尾。
func prepareSmallPermissionBatch(t *testing.T, baseURL, directory string,
) *preparer {
	t.Helper()
	loader := &preparer{
		config: performance.Config{APIURL: baseURL, RPS: 20, OutputDir: directory},
		logger: klog.NewStdLogger(io.Discard), opt: defaultOptions(),
	}
	targetPath, closeClient, err := loader.Prepare(t.Context(), directory)
	if err != nil {
		t.Fatal(err)
	}
	if targetPath != filepath.Join(directory, "data", "hot-targets.jsonl") {
		t.Fatalf("unexpected preparation target path: %s", targetPath)
	}
	t.Cleanup(func() {
		if closeClient != nil {
			_ = closeClient()
		}
	})
	return loader
}

// TestPermissionDataReuseAndMissingRowsRebuild 验证复用同批次及缺失背景权限时重建。
func TestPermissionDataReuseAndMissingRowsRebuild(t *testing.T) {
	useSmallScale(t)
	fixture, server := newFixtureAPI(t)
	directory := t.TempDir()
	first := prepareSmallPermissionBatch(t, server.URL, directory)
	batch := first.manifest.Batch
	fixture.mu.Lock()
	created := fixture.createdCount
	fixture.mu.Unlock()
	second := prepareSmallPermissionBatch(t, server.URL, directory)
	second.config.RPS = 15
	if _, _, err := second.Prepare(t.Context(), directory); err != nil {
		t.Fatal(err)
	}
	fixture.mu.Lock()
	if fixture.createdCount != created || second.manifest.Batch != batch {
		t.Fatal("complete batch was not reused")
	}
	// 背景角色不在读组中，必须检查真实记录，不能只抽查返回权限。
	id := second.manifest.ProjectList[0].PermissionIDList[3][1]
	delete(fixture.recordMap["user-role-permission-assoc"], id)
	unrelated := fixture.addRecord("project", map[string]any{"projName": "existing_project"})
	fixture.mu.Unlock()
	third := prepareSmallPermissionBatch(t, server.URL, directory)
	if third.manifest.Batch == batch {
		t.Fatal("missing background row did not trigger rebuilding")
	}
	fixture.mu.Lock()
	if len(fixture.recordMap["project"]) != 2 || fixture.recordMap["project"][unrelated] == nil {
		t.Fatal("old batch leaked or unrelated data deleted")
	}
	fixture.mu.Unlock()
	err := third.CleanupData(t.Context(), directory)
	if err != nil {
		t.Fatal(err)
	}
	fixture.mu.Lock()
	created = fixture.createdCount
	fixture.mu.Unlock()
	err = third.CleanupData(t.Context(), directory)
	if err != nil {
		t.Fatal(err)
	}
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	if fixture.createdCount != created {
		t.Fatal("repeated cleanup recreated an administrator")
	}
	for kind, recordMap := range fixture.recordMap {
		if kind == "project" && len(recordMap) == 1 && recordMap[unrelated] != nil {
			continue
		}
		if len(recordMap) != 0 {
			t.Fatalf("remaining batch records: %s", kind)
		}
	}
}

// TestPermissionScaleUpdateRetriesFailedCleanup 验证规模更新失败不覆盖旧清单。
func TestPermissionScaleUpdateRetriesFailedCleanup(t *testing.T) {
	useSmallScale(t)
	fixture, server := newFixtureAPI(t)
	directory := t.TempDir()
	loader := prepareSmallPermissionBatch(t, server.URL, directory)
	batch := loader.manifest.Batch
	defaultScale.Actions++
	fixture.mu.Lock()
	fixture.failedDeletes = true
	fixture.mu.Unlock()
	_, _, err := loader.Prepare(t.Context(), directory)
	if err == nil {
		t.Fatal("cleanup failure must stop rebuilding")
	}
	retained, err := readManifest(filepath.Join(directory, "data", "01-permissions.json"))
	if err != nil || retained.Batch != batch || retained.Ready {
		t.Fatalf("old cleanup state was not retained: %v", err)
	}
	fixture.mu.Lock()
	fixture.failedDeletes = false
	fixture.mu.Unlock()
	_, _, err = loader.Prepare(t.Context(), directory)
	if err != nil || loader.manifest.Batch == batch || loader.manifest.Scale.Actions != 3 {
		t.Fatalf("failed to rebuild after retry: %v", err)
	}
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	if len(fixture.recordMap["user-role-permission-assoc"]) != 12 {
		t.Fatal("old permission records leaked")
	}
}

// TestPermissionCheckFailureRetainsBatch 验证请求错误不能被误判为数据不足。
func TestPermissionCheckFailureRetainsBatch(t *testing.T) {
	useSmallScale(t)
	fixture, server := newFixtureAPI(t)
	directory := t.TempDir()
	loader := prepareSmallPermissionBatch(t, server.URL, directory)
	batch := loader.manifest.Batch
	fixture.mu.Lock()
	created := fixture.createdCount
	fixture.failedReads = true
	fixture.mu.Unlock()
	_, _, err := loader.Prepare(t.Context(), directory)
	if err == nil {
		t.Fatal("expected failed data check")
	}
	retained, err := readManifest(filepath.Join(directory, "data", "01-permissions.json"))
	if err != nil || retained.Batch != batch || !retained.Ready {
		t.Fatal("read error replaced or cleaned the batch")
	}
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	if fixture.createdCount != created || len(fixture.recordMap["user-role-permission-assoc"]) != 8 {
		t.Fatal("read error changed fixture data")
	}
}

// TestPermissionCleanupRecoversAdministrator 验证初始登录结果漏记仍可恢复完整清理。
func TestPermissionCleanupRecoversAdministrator(t *testing.T) {
	useSmallScale(t)
	fixture, server := newFixtureAPI(t)
	directory := t.TempDir()
	loader := prepareSmallPermissionBatch(t, server.URL, directory)
	loader.manifest.Admin = User{}
	loader.manifest.RecordIDMap["user"] = nil
	loader.manifest.Ready = false
	err := saveManifest(loader.manifestPath, loader.manifest)
	if err != nil {
		t.Fatal(err)
	}
	command := Entrypoint.CleanupEntrypoint().NewCobraCommand()
	command.SetOut(io.Discard)
	command.SetArgs([]string{directory})
	err = command.ExecuteContext(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	for kind, recordMap := range fixture.recordMap {
		if len(recordMap) != 0 {
			t.Fatalf("recovered cleanup left %s records", kind)
		}
	}
}

// TestPrepareClearsStaleReports 验证新测试准备失败时也不会残留旧窗口报告。
func TestPrepareClearsStaleReports(t *testing.T) {
	for _, prepareOnly := range []bool{false, true} {
		t.Run(fmt.Sprintf("prepare-only=%t", prepareOnly), func(t *testing.T) {
			directory := t.TempDir()
			oldReportList := []string{
				"current/hot/11-vegeta-report.txt",
				"current/control/11-vegeta-report.txt",
				"current/writer-report.txt",
				"current/rps-200/writer-report.txt",
				"current/rps-400/hot/11-vegeta-report.txt",
				"current/00-auto-run.json", "current/00-run.json", "current/extra.png",
			}
			preservedList := []string{
				"data/01-permissions.json", "00-run.json", "data/hot-targets.jsonl",
				"data/control-targets.jsonl", "grafana-rps20-1.png",
				"round-01/hot/11-vegeta-report.txt", "round-02/writer-report.txt",
				"round-01/00-run.json", "round-01/grafana.png",
				"notes/readme.md", "rps-notes/readme.md",
			}
			for _, name := range append(oldReportList, preservedList...) {
				path := filepath.Join(directory, name)
				err := os.MkdirAll(filepath.Dir(path), 0o700)
				if err != nil {
					t.Fatal(err)
				}
				err = os.WriteFile(path, []byte("original"), 0o600)
				if err != nil {
					t.Fatal(err)
				}
			}
			opt := defaultOptions()
			opt.PrepareOnly = prepareOnly
			loader := &preparer{
				config: performance.Config{RPS: 20}, opt: opt,
				logger: klog.NewStdLogger(io.Discard),
			}
			_, cleanup, err := loader.Prepare(context.Background(), directory)
			if err == nil {
				t.Fatal("invalid manifest should fail preparation")
			}
			if cleanup != nil {
				t.Cleanup(func() {
					if err := cleanup(); err != nil {
						t.Error(err)
					}
				})
			}
			for _, name := range oldReportList {
				_, err = os.Stat(filepath.Join(directory, name))
				if prepareOnly && err != nil {
					t.Fatalf("preparation removed %s: %v", name, err)
				}
				if !prepareOnly && !os.IsNotExist(err) {
					t.Fatalf("stale report remains: %s (%v)", name, err)
				}
			}
			for _, name := range preservedList {
				content, err := os.ReadFile(filepath.Join(directory, name))
				if err != nil || string(content) != "original" {
					t.Fatalf("preserved file changed: %s (%v)", name, err)
				}
			}
		})
	}
}

// TestRunLoadReadWrite 验证读写并行且报告直接保存在输出目录。
func TestRunLoadReadWrite(t *testing.T) {
	for _, sampling := range []bool{false, true} {
		t.Run(fmt.Sprintf("sampling=%t", sampling), func(t *testing.T) {
			if runtime.GOOS == "windows" {
				t.Skip("fake Vegeta uses a POSIX shell")
			}
			fixture, server := newFixtureAPI(t)
			directory := t.TempDir()
			manifestPath := filepath.Join(directory, "data", "01-permissions.json")
			if err := os.MkdirAll(filepath.Dir(manifestPath), 0o700); err != nil {
				t.Fatal(err)
			}
			manifest, err := prepareFixture(t.Context(), server.URL, manifestPath,
				fixtureScale{2, 4, 3, 2, 2}, klog.NewStdLogger(io.Discard))
			if err != nil {
				t.Fatal(err)
			}
			toolDirectory := t.TempDir()
			toolPath := filepath.Join(toolDirectory, "vegeta")
			script := `#!/bin/sh
case "$1" in
attack)
  printf 'attack %s\n' "$(date +%S)" >> "$PERMISSION_ATTACK_LOG"
  printf '%s\n' "$@" >> "$PERMISSION_RATE_LOG"
  sleep 0.02
  printf 'sample'
  ;;
report)
  printf 'Requests [total, rate, throughput] 10, 10, 10\nSuccess [ratio] 100.00%%\n'
  ;;
*) exit 1 ;;
esac
`
			err = os.WriteFile(toolPath, []byte(script), 0o700)
			if err != nil {
				t.Fatal(err)
			}
			attackLog := filepath.Join(directory, "attacks.txt")
			t.Setenv("PERMISSION_ATTACK_LOG", attackLog)
			rateLog := filepath.Join(directory, "rates.txt")
			t.Setenv("PERMISSION_RATE_LOG", rateLog)
			t.Setenv("PATH", toolDirectory+string(os.PathListSeparator)+os.Getenv("PATH"))
			logger := klog.NewStdLogger(io.Discard)
			opt := defaultOptions()
			opt.WriteRPS = 100
			client := newAPIClient(server.URL, manifest.Admin.Token)
			t.Cleanup(client.httpClient.CloseIdleConnections)
			loader := &preparer{
				manifest: manifest, manifestPath: manifestPath,
				client: client, logger: logger, opt: opt,
			}
			config := performance.Config{
				APIURL: server.URL, DiagnosticsURL: server.URL, Sampling: sampling,
				RPS: 20, Duration: 50 * time.Millisecond,
				OutputDir: filepath.Join(directory, "current"),
			}
			err = loader.RunLoad(t.Context(), performance.NewRunner(logger), config,
				filepath.Join(directory, "data", "hot-targets.jsonl"))
			if err != nil {
				t.Fatal(err)
			}
			content, err := os.ReadFile(attackLog)
			if err != nil || strings.Count(string(content), "attack") != 4 {
				t.Fatalf("expected two read groups warming up and measuring once: %q, %v", content, err)
			}
			rateContent, rateErr := os.ReadFile(rateLog)
			if rateErr != nil || strings.Count(string(rateContent), "-rate=10/1.25s") != 2 ||
				strings.Count(string(rateContent), "-rate=10/1s") != 2 {
				t.Fatalf("unexpected warmup/measured rates: %s, %v", rateContent, rateErr)
			}
			attacks := strings.Split(strings.TrimSpace(string(content)), "\n")
			for _, attack := range attacks[2:] {
				if attack != "attack 00" {
					t.Fatalf("measured read missed minute boundary: %s", attack)
				}
			}
			for _, group := range []string{"hot", "control"} {
				_, err = os.Stat(filepath.Join(directory, "current", group, "11-vegeta-report.txt"))
				if err != nil {
					t.Fatal(err)
				}
			}
			content, err = os.ReadFile(filepath.Join(directory, "current", "writer-report.txt"))
			if err != nil {
				t.Fatal(err)
			}
			var requests int
			_, err = fmt.Sscanf(string(content), "Requests %d", &requests)
			if err != nil || requests == 0 {
				t.Fatalf("administrator did not write during measurement: %s, %v", content, err)
			}
			fixture.mu.Lock()
			updates := fixture.updates
			samplingRequests := strings.Join(fixture.samplingRequests, ",")
			samplingUpdateCount := fixture.samplingUpdateCount
			fixture.mu.Unlock()
			if sampling && samplingUpdateCount == 0 {
				t.Fatal("sampling started before warmup writes")
			}
			wantRequests := ""
			if sampling {
				wantRequests = "POST,DELETE"
			}
			if samplingRequests != wantRequests {
				t.Fatalf("sampling lifecycle = %s", samplingRequests)
			}
			if updates == 0 {
				t.Fatal("read/write load did not modify permissions")
			}
			for _, name := range []string{"pure", "mixed", "recovery", "round-01"} {
				_, err = os.Stat(filepath.Join(directory, name))
				if !os.IsNotExist(err) {
					t.Fatalf("unexpected report directory %s: %v", name, err)
				}
			}
		})
	}
}

// TestSamplingClientCancellation 验证取消后仍关闭本档会话以及开启错误不被忽略。
func TestSamplingClientCancellation(t *testing.T) {
	var deleted atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/debug/pprof/sampling" {
			t.Errorf("path: %s", r.URL.Path)
		}
		if r.Method == http.MethodPost {
			if r.URL.Query().Get("seconds") != "1" {
				t.Error("sampling duration differs from measured duration")
			}
			_, _ = io.WriteString(w, `{"id":42,"seconds":1}`)
			return
		}
		if r.Method != http.MethodDelete || r.URL.Query().Get("id") != "42" {
			t.Error("wrong sampling owner")
		}
		deleted.Store(true)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(t.Context())
	stop, duration, err := performance.StartSampling(ctx, performance.Config{Sampling: true, DiagnosticsURL: server.URL + "/debug/pprof/heap", Duration: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if duration != time.Second {
		t.Fatalf("actual duration = %s", duration)
	}
	cancel()
	if err = stop(); err != nil || !deleted.Load() {
		t.Fatalf("cancel cleanup: %v", err)
	}
	if _, _, err = performance.StartSampling(t.Context(), performance.Config{Sampling: true}); err == nil {
		t.Fatal("missing diagnostics accepted")
	}
	failure := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer failure.Close()
	if _, _, err = performance.StartSampling(t.Context(), performance.Config{Sampling: true, DiagnosticsURL: failure.URL, Duration: time.Second}); err == nil {
		t.Fatal("sampling failure ignored")
	}
}

// TestLoadPreparationDoesNotSample 验证准备阶段失败或仅准备数据时不请求采样。
func TestLoadPreparationDoesNotSample(t *testing.T) {
	var requestList []string
	var requestMu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMu.Lock()
		defer requestMu.Unlock()
		requestList = append(requestList, r.Method)
		if r.Method == http.MethodPost {
			_, _ = io.WriteString(w, `{"id":42,"seconds":1}`)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "not-a-directory")
	err := os.WriteFile(path, []byte("existing"), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	loader := &preparer{opt: defaultOptions(), logger: klog.NewStdLogger(io.Discard)}
	config := performance.Config{DiagnosticsURL: server.URL, OutputDir: path}
	err = loader.RunLoad(t.Context(), nil, config, "")
	requestMu.Lock()
	defer requestMu.Unlock()
	if err == nil || len(requestList) != 0 {
		t.Fatalf("startup failure cleanup: requests=%v, error=%v", requestList, err)
	}
	loader.opt.PrepareOnly = true
	err = loader.RunLoad(t.Context(), nil, config, "")
	if err != nil || len(requestList) != 0 {
		t.Fatal("data preparation touched sampling")
	}
}

// TestSamplingDuration 验证秒级取整及使用后端截断后的实际采样时间。
func TestSamplingDuration(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		measured  time.Duration
		requested string
		actual    int
	}{
		{"fractional", 1500 * time.Millisecond, "2", 2},
		{"capped", 48 * time.Hour, "172800", 86400},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					if r.URL.Query().Get("seconds") != testCase.requested {
						t.Errorf("requested seconds = %s", r.URL.Query().Get("seconds"))
					}
					_, _ = fmt.Fprintf(w, `{"id":1,"seconds":%d}`, testCase.actual)
				} else {
					w.WriteHeader(http.StatusNoContent)
				}
			}))
			defer server.Close()
			stop, duration, err := performance.StartSampling(t.Context(), performance.Config{Sampling: true, DiagnosticsURL: server.URL, Duration: testCase.measured})
			if err != nil {
				t.Fatal(err)
			}
			if duration != time.Duration(testCase.actual)*time.Second {
				t.Fatalf("actual duration = %s", duration)
			}
			if err = stop(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TestRunPairRetainsFailedReports 验证先完成的失败报告不会取消另一组报告进程。
func TestRunPairRetainsFailedReports(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake Vegeta uses a POSIX shell")
	}
	for _, first := range []string{"hot", "control"} {
		t.Run(first, func(t *testing.T) {
			directory := t.TempDir()
			toolDir := t.TempDir()
			script := `#!/bin/sh
case "$1" in
attack) printf 'sample' ;;
report)
  case "$3" in
    */` + first + `/*) ;;
    *) sleep 0.1 ;;
  esac
  printf 'Requests [total, rate, throughput] 10, 10, 9\nSuccess [ratio] 90.00%%\n'
  ;;
*) exit 1 ;;
esac
`
			if err := os.WriteFile(filepath.Join(toolDir, "vegeta"), []byte(script), 0o700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", toolDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			runner := performance.NewRunner(klog.NewStdLogger(io.Discard))
			err := runPair(t.Context(), func(ctx context.Context, group string) error {
				config := performance.Config{APIURL: "http://localhost", RPS: 10,
					Duration: time.Millisecond, OutputDir: filepath.Join(directory, group)}
				return runner.RunMeasured(ctx, config, filepath.Join(directory, group+"-targets.jsonl"))
			})
			if !errors.Is(err, performance.ErrLoadFailed) || strings.Contains(err.Error(), "Vegeta report:") {
				t.Fatalf("unexpected failure: %v", err)
			}
			if strings.Count(err.Error(), "success ratio 90.00%") != 2 {
				t.Fatalf("missing group failure: %v", err)
			}
			for _, group := range []string{"hot", "control"} {
				content, err := os.ReadFile(filepath.Join(directory, group, "11-vegeta-report.txt"))
				if err != nil || !strings.Contains(string(content), "Success [ratio] 90.00%") {
					t.Fatalf("incomplete %s report: %v", group, err)
				}
			}
		})
	}
}

// TestRunPairCancellation 验证执行器故障与用户取消仍能结束另一组负载。
func TestRunPairCancellation(t *testing.T) {
	for _, userCancel := range []bool{false, true} {
		t.Run(fmt.Sprintf("user-cancel=%t", userCancel), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			started := make(chan struct{})
			fault := errors.New("attack failed")
			err := runPair(ctx, func(ctx context.Context, group string) error {
				if group == "control" {
					close(started)
					<-ctx.Done()
					return ctx.Err()
				}
				<-started
				if userCancel {
					cancel()
					return ctx.Err()
				}
				return fault
			})
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("sibling not canceled: %v", err)
			}
			if !userCancel && !errors.Is(err, fault) {
				t.Fatalf("fault lost: %v", err)
			}
		})
	}
}
