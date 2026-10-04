package permissions

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/pancake-lee/pgo/cmd/pgo/common"
	performance "github.com/pancake-lee/pgo/cmd/pgo/performance/core"
)

// fixtureScale 保存固定业务规模，测试可使用小规模验证完整流程。
type fixtureScale struct{ Projects, Roles, Actions, UsersPerGroup, RolesPerGroup int }

// defaultScale 使用本轮确认的项目、角色、权限点与用户数量。
var defaultScale = fixtureScale{20, 20, 100, 25, 5}

// User 保存用户身份与所属测试组，令牌只写入私有清单。
type User struct {
	ID      int32
	Name    string
	Token   string
	Project int
	Group   string
}

// Project 保存角色、权限主键以及管理员已完成的路径版本。
type Project struct {
	ID               int32
	RoleIDList       []int32
	PermissionIDList [][]int32
	VersionList      []int
}

// Manifest 保存 HTTP 创建成功的批次主键，支持保留数据与中断后清理。
type Manifest struct {
	Batch       string
	BaseURL     string
	Scale       fixtureScale
	Admin       User
	UserList    []User
	ProjectList []Project
	RecordIDMap map[string][]int32
	Ready       bool
	Cleaned     bool
}

// saveManifest 原子替换私有清单，保存数据而不保存监控信息。
func saveManifest(path string, manifest *Manifest) error {
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(path+".tmp", content, 0o600)
	if err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}

// readManifest 读取批次身份，允许已清理清单供幂等操作识别。
func readManifest(path string) (*Manifest, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	err = json.Unmarshal(content, &manifest)
	if err != nil {
		return nil, err
	}
	if manifest.Batch == "" || manifest.BaseURL == "" {
		return nil, errors.New("invalid permission manifest")
	}
	if manifest.RecordIDMap == nil {
		manifest.RecordIDMap = make(map[string][]int32)
	}
	return &manifest, nil
}

// loadManifest 读取未清理的批次，供已有准备与清理流程使用。
func loadManifest(path string) (*Manifest, error) {
	manifest, err := readManifest(path)
	if err != nil {
		return nil, err
	}
	if manifest.Cleaned {
		return nil, errors.New("permission manifest is already cleaned")
	}
	return manifest, nil
}

// prepareFixture 使用现有 HTTP 接口准备数据并逐阶段保存成功记录。
func prepareFixture(ctx context.Context, baseURL, path string,
	scale fixtureScale, logger klog.Logger,
) (*Manifest, error) {
	batchBytes := make([]byte, 6)
	_, err := rand.Read(batchBytes)
	if err != nil {
		return nil, err
	}
	manifest := &Manifest{
		Batch: hex.EncodeToString(batchBytes), BaseURL: baseURL,
		Scale: scale, RecordIDMap: make(map[string][]int32),
	}
	err = os.MkdirAll(filepath.Dir(path), 0o700)
	if err != nil {
		return nil, err
	}
	err = saveManifest(path, manifest)
	if err != nil {
		return manifest, err
	}
	started := time.Now()
	var count int
	defer func() {
		average := time.Duration(0)
		if count > 0 {
			average = time.Since(started) / time.Duration(count)
		}
		_ = logger.Log(klog.LevelInfo, "msg", "HTTP data preparation summary",
			"created", count, "elapsed", time.Since(started), "ready", manifest.Ready,
			"wallTimePerRecord", average,
			"manifest", path)
	}()
	loginClient, err := common.NewClient(baseURL)
	if err != nil {
		return manifest, err
	}
	adminName := "perm_" + manifest.Batch + "_admin"
	admin, token, err := loginClient.Login(ctx, adminName)
	if err != nil {
		return manifest, err
	}
	manifest.Admin = User{ID: admin.ID, Name: adminName, Token: token}
	manifest.RecordIDMap["user"] = append(manifest.RecordIDMap["user"], admin.ID)
	count++
	err = saveManifest(path, manifest)
	if err != nil {
		return manifest, err
	}
	client := newAPIClient(baseURL, token)
	defer client.httpClient.CloseIdleConnections()
	add := func(kind string, fields map[string]any) (int32, error) {
		fields["createTime"] = time.Now().Unix()
		fields["createUser"] = admin.ID
		id, addErr := client.add(ctx, kind, fields)
		if addErr == nil {
			manifest.RecordIDMap[kind] = append(manifest.RecordIDMap[kind], id)
			count++
		}
		return id, addErr
	}
	for projectIndex := 0; projectIndex < scale.Projects; projectIndex++ {
		projectID, addErr := add("project", map[string]any{
			"projName":     fmt.Sprintf("perm_%s_%02d", manifest.Batch, projectIndex),
			"lastEditFrom": "load", "updateTime": time.Now().Unix(), "updateUser": admin.ID,
		})
		if addErr != nil {
			return manifest, errors.Join(addErr, saveManifest(path, manifest))
		}
		project := Project{ID: projectID, VersionList: make([]int, scale.Actions)}
		for roleIndex := 0; roleIndex < scale.Roles; roleIndex++ {
			roleID, addErr := add("user-role", map[string]any{
				"projID": projectID, "roleName": fmt.Sprintf("role_%02d", roleIndex),
				"updateTime": time.Now().Unix(), "updateUser": admin.ID,
			})
			if addErr != nil {
				return manifest, errors.Join(addErr, saveManifest(path, manifest))
			}
			project.RoleIDList = append(project.RoleIDList, roleID)
			project.PermissionIDList = append(project.PermissionIDList, make([]int32, scale.Actions))
		}
		manifest.ProjectList = append(manifest.ProjectList, project)
		err = saveManifest(path, manifest)
		if err != nil {
			return manifest, err
		}
	}
	// 权限创建按小批次并发，清单持久化发生在每批请求结束后。
	for projectIndex := range manifest.ProjectList {
		project := &manifest.ProjectList[projectIndex]
		for roleIndex, roleID := range project.RoleIDList {
			group := "other"
			if roleIndex < scale.RolesPerGroup {
				group = "hot"
			}
			if roleIndex >= scale.RolesPerGroup && roleIndex < 2*scale.RolesPerGroup {
				group = "control"
			}
			for start := 0; start < scale.Actions; start += 8 {
				var wg sync.WaitGroup
				var mu sync.Mutex
				var batchErr error
				for action := start; action < min(start+8, scale.Actions); action++ {
					wg.Add(1)
					go func(action int) {
						defer wg.Done()
						id, addErr := client.add(ctx, "user-role-permission-assoc", map[string]any{
							"createTime": time.Now().Unix(), "createUser": admin.ID,
							"roleID": roleID, "action": fmt.Sprintf("action_%03d", action),
							"pathPattern": permissionPath(projectIndex, group, action, 0),
						})
						mu.Lock()
						defer mu.Unlock()
						if addErr != nil {
							batchErr = errors.Join(batchErr, addErr)
							return
						}
						project.PermissionIDList[roleIndex][action] = id
						manifest.RecordIDMap["user-role-permission-assoc"] = append(
							manifest.RecordIDMap["user-role-permission-assoc"], id)
						count++
					}(action)
				}
				wg.Wait()
				err = errors.Join(batchErr, saveManifest(path, manifest))
				if err != nil {
					return manifest, err
				}
			}
		}
		for _, group := range []string{"hot", "control"} {
			for index := 0; index < scale.UsersPerGroup; index++ {
				name := fmt.Sprintf("p_%s_%02d_%s_%02d", manifest.Batch, projectIndex, group[:1], index)
				user, userToken, loginErr := loginClient.Login(ctx, name)
				if loginErr != nil {
					return manifest, loginErr
				}
				manifest.RecordIDMap["user"] = append(manifest.RecordIDMap["user"], user.ID)
				manifest.UserList = append(manifest.UserList, User{
					ID: user.ID, Name: name, Token: userToken, Project: projectIndex, Group: group,
				})
				count++
				_, err = add("user-project-assoc", map[string]any{"userID": user.ID, "projID": project.ID})
				if err != nil {
					return manifest, errors.Join(err, saveManifest(path, manifest))
				}
				roleStart := 0
				if group == "control" {
					roleStart = scale.RolesPerGroup
				}
				for _, roleID := range project.RoleIDList[roleStart : roleStart+scale.RolesPerGroup] {
					_, err = add("user-role-assoc", map[string]any{"userID": user.ID, "roleID": roleID})
					if err != nil {
						return manifest, errors.Join(err, saveManifest(path, manifest))
					}
				}
				err = saveManifest(path, manifest)
				if err != nil {
					return manifest, err
				}
			}
		}
		_ = logger.Log(klog.LevelInfo, "msg", "permission project data ready",
			"projectsCompleted", projectIndex+1, "projectsTotal", scale.Projects,
			"permissionsCreated", len(manifest.RecordIDMap["user-role-permission-assoc"]),
			"usersCreated", len(manifest.UserList))
	}
	manifest.Ready = true
	return manifest, saveManifest(path, manifest)
}

// cleanupFixture 按依赖顺序分批删除清单记录，成功批次立即移出清单。
func cleanupFixture(ctx context.Context, client *apiClient,
	manifest *Manifest, path string,
) error {
	err := recoverRecordIDs(ctx, client, manifest)
	if err != nil {
		return err
	}
	err = saveManifest(path, manifest)
	if err != nil {
		return err
	}
	for _, kind := range []string{"user-role-permission-assoc", "user-role-assoc",
		"user-project-assoc", "user-role", "project", "user"} {
		for len(manifest.RecordIDMap[kind]) > 0 {
			idList := manifest.RecordIDMap[kind]
			// 管理员最后删除，保证中断清理仍能重新登录。
			if kind == "user" && idList[0] == manifest.Admin.ID && len(idList) > 1 {
				idList = append(append([]int32{}, idList[1:]...), idList[0])
				manifest.RecordIDMap[kind] = idList
			}
			size := min(100, len(idList))
			if kind == "user" && idList[len(idList)-1] == manifest.Admin.ID && len(idList) > 1 {
				size = min(size, len(idList)-1)
			}
			err := client.deleteIDs(ctx, kind, idList[:size])
			if err != nil {
				return err
			}
			manifest.RecordIDMap[kind] = idList[size:]
			err = saveManifest(path, manifest)
			if err != nil {
				return err
			}
		}
	}
	manifest.Cleaned = true
	return saveManifest(path, manifest)
}

// recoverRecordIDs 找回中断时已入库但未写入清单的本批次记录。
func recoverRecordIDs(ctx context.Context, client *apiClient,
	manifest *Manifest,
) error {
	knownMap := make(map[string]map[int32]bool)
	for kind, idList := range manifest.RecordIDMap {
		knownMap[kind] = make(map[int32]bool)
		for _, id := range idList {
			knownMap[kind][id] = true
		}
	}
	for _, kind := range []string{"project", "user", "user-role",
		"user-role-permission-assoc", "user-role-assoc", "user-project-assoc"} {
		var response map[string][]struct {
			ID       int32
			ProjID   int32
			RoleID   int32
			UserID   int32
			ProjName string
			UserName string
		}
		err := client.request(ctx, "GET", "/"+kind, nil, &response)
		if err != nil {
			return err
		}
		if knownMap[kind] == nil {
			knownMap[kind] = make(map[int32]bool)
		}
		for _, record := range response[envelope(kind)+"List"] {
			owned := false
			switch kind {
			case "project":
				owned = strings.HasPrefix(record.ProjName, "perm_"+manifest.Batch+"_")
			case "user":
				owned = strings.HasPrefix(record.UserName, "p_"+manifest.Batch+"_") ||
					record.UserName == "perm_"+manifest.Batch+"_admin"
			case "user-role", "user-project-assoc":
				owned = knownMap["project"][record.ProjID]
			case "user-role-permission-assoc", "user-role-assoc":
				owned = knownMap["user-role"][record.RoleID]
			}
			if owned && record.ID > 0 && !knownMap[kind][record.ID] {
				knownMap[kind][record.ID] = true
				manifest.RecordIDMap[kind] = append(manifest.RecordIDMap[kind], record.ID)
			}
		}
	}
	return nil
}

// verifyRecordIDs 检查清单记录仍然存在，避免缺失背景行被权限抽查掩盖。
func verifyRecordIDs(ctx context.Context, client *apiClient,
	manifest *Manifest,
) error {
	for kind, idList := range manifest.RecordIDMap {
		var response map[string][]struct{ ID int32 }
		err := client.request(ctx, "GET", "/"+kind, nil, &response)
		if err != nil {
			return err
		}
		idMap := make(map[int32]bool)
		for _, record := range response[envelope(kind)+"List"] {
			idMap[record.ID] = true
		}
		for _, id := range idList {
			if !idMap[id] {
				return performance.ErrDataMismatch
			}
		}
	}
	return nil
}
