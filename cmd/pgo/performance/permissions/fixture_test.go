package permissions

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	klog "github.com/go-kratos/kratos/v2/log"
	performance "github.com/pancake-lee/pgo/cmd/pgo/performance/core"
)

// fakePermissionAPI 提供受保护 CRUD 与项目筛选以验证客户端完整流程。
type fakePermissionAPI struct {
	mu             sync.Mutex
	nextID         int
	rowMap         map[string]map[int]map[string]any
	disableList    bool
	omitMode       bool
	losePermission bool
}

// newFakePermissionAPI 初始化包含非测试数据的模拟服务。
func newFakePermissionAPI() *fakePermissionAPI {
	api := &fakePermissionAPI{
		nextID: 100, rowMap: make(map[string]map[int]map[string]any),
	}
	for _, path := range []string{
		"user", "project", "user-role", "user-role-assoc",
		"user-role-permission-assoc",
	} {
		api.rowMap[path] = map[int]map[string]any{
			1: {"ID": float64(1), "userName": "unrelated", "createUser": float64(1)},
		}
	}
	return api
}

// ServeHTTP 模拟真实 SDK 使用的登录、权限与 CRUD 协议。
func (api *fakePermissionAPI) ServeHTTP(
	w http.ResponseWriter, r *http.Request,
) {
	api.mu.Lock()
	defer api.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Path == "/user/token" {
		var request struct {
			UserName string `json:"userName"`
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		for _, user := range api.rowMap["user"] {
			if user["userName"] == request.UserName {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"user": user, "token": "token",
				})
				return
			}
		}
		api.nextID++
		user := map[string]any{
			"ID": float64(api.nextID), "userName": request.UserName,
		}
		api.rowMap["user"][api.nextID] = user
		_ = json.NewEncoder(w).Encode(map[string]any{
			"user": user, "token": "token",
		})
		return
	}
	if strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ") != "token" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if r.URL.Path == "/user/permissions" {
		mode := r.Header.Get(aggregationHeader)
		if mode == "list" && api.disableList {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if !api.omitMode {
			w.Header().Set(aggregationHeader, mode)
		}
		userID, _ := strconv.Atoi(r.URL.Query().Get("userID"))
		projectID, _ := strconv.Atoi(r.URL.Query().Get("projectID"))
		permissionMap := make(map[string]string)
		for _, assoc := range api.rowMap["user-role-assoc"] {
			if assoc["userID"] != float64(userID) {
				continue
			}
			roleID := int(assoc["roleID"].(float64))
			role := api.rowMap["user-role"][roleID]
			if role["projID"] != float64(projectID) {
				continue
			}
			for _, permission := range api.rowMap["user-role-permission-assoc"] {
				if permission["roleID"] == float64(roleID) {
					permissionMap[permission["action"].(string)] =
						permission["pathPattern"].(string)
				}
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"actionToPathPattern": permissionMap,
		})
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/")
	keyMap := map[string]string{
		"user": "user", "project": "project", "user-role": "userRole",
		"user-role-assoc":            "userRoleAssoc",
		"user-role-permission-assoc": "userRolePermissionAssoc",
	}
	key, ok := keyMap[path]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	switch r.Method {
	case http.MethodPost:
		bodyMap := make(map[string]map[string]any)
		_ = json.NewDecoder(r.Body).Decode(&bodyMap)
		row := bodyMap[key]
		if row == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		api.nextID++
		row["ID"] = float64(api.nextID)
		api.rowMap[path][api.nextID] = row
		if path == "user-role-permission-assoc" && api.losePermission {
			api.losePermission = false
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{key: row})
	case http.MethodGet:
		rowList := make([]map[string]any, 0)
		for _, row := range api.rowMap[path] {
			rowList = append(rowList, row)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{key + "List": rowList})
	case http.MethodDelete:
		idList := r.URL.Query()["IDList"]
		if len(idList) == 0 || len(idList) > 100 || idList[0] == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		for _, text := range idList {
			id, _ := strconv.Atoi(text)
			delete(api.rowMap[path], id)
		}
		_, _ = io.WriteString(w, "{}")
	}
}

// assertOnlyUnrelated 验证清理后所有原始非批次数据仍保留。
func (api *fakePermissionAPI) assertOnlyUnrelated(t *testing.T) {
	t.Helper()
	api.mu.Lock()
	defer api.mu.Unlock()
	for path, rowMap := range api.rowMap {
		if len(rowMap) != 1 || rowMap[1] == nil {
			t.Errorf("%s rows after cleanup: %d", path, len(rowMap))
		}
	}
}

// TestPermissionsBatchLifecycle 验证准备、算法核对、权限目标和精确清理。
func TestPermissionsBatchLifecycle(t *testing.T) {
	api := newFakePermissionAPI()
	server := httptest.NewServer(api)
	defer server.Close()
	directory := t.TempDir()
	preparer := &preparer{
		config:  performance.Config{APIURL: server.URL},
		options: options{Roles: 2, Permissions: 150, Aggregation: "list"},
		logger:  klog.NewStdLogger(io.Discard),
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	targetPath, cleanup, err := preparer.Prepare(ctx, directory)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	var target struct {
		Method string
		URL    string
		Header map[string][]string
	}
	err = json.Unmarshal(content, &target)
	if err != nil || target.Method != "GET" ||
		target.Header[aggregationHeader][0] != "list" ||
		target.Header["Authorization"][0] != "token" {
		t.Fatalf("target = %s, err = %v", content, err)
	}
	manifest, err := readManifest(filepath.Join(directory, "01-permissions.json"))
	if err != nil || len(manifest.PermissionIDList) != 301 {
		t.Fatalf("manifest = %+v, err = %v", manifest, err)
	}
	info, err := os.Stat(manifest.path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatal("manifest permissions must be 0600")
	}
	cancel()
	err = cleanup()
	if err != nil {
		t.Fatal(err)
	}
	api.assertOnlyUnrelated(t)
	err = manifest.cleanup(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	api.assertOnlyUnrelated(t)
}

// TestPermissionsPartialPreparationRecovery 找回服务器写成功但客户端未拿到 ID 的记录。
func TestPermissionsPartialPreparationRecovery(t *testing.T) {
	api := newFakePermissionAPI()
	api.losePermission = true
	server := httptest.NewServer(api)
	defer server.Close()
	directory := t.TempDir()
	preparer := &preparer{
		config:  performance.Config{APIURL: server.URL},
		options: options{Roles: 2, Permissions: 4, Aggregation: "list"},
		logger:  klog.NewStdLogger(io.Discard),
	}
	_, cleanup, err := preparer.Prepare(t.Context(), directory)
	if err == nil || cleanup == nil {
		t.Fatal("expected partial preparation error and cleanup")
	}
	manifest, err := readManifest(filepath.Join(directory, "01-permissions.json"))
	if err != nil || !manifest.Incomplete {
		t.Fatal("partial manifest must be recoverable")
	}
	err = manifest.cleanup(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	api.assertOnlyUnrelated(t)
}

// TestPermissionsPreflightRejectsOldOrDisabledServices 在大批写入前拒绝不匹配服务。
func TestPermissionsPreflightRejectsOldOrDisabledServices(t *testing.T) {
	for _, oldServer := range []bool{false, true} {
		t.Run(fmt.Sprint(oldServer), func(t *testing.T) {
			api := newFakePermissionAPI()
			api.omitMode = oldServer
			api.disableList = !oldServer
			server := httptest.NewServer(api)
			defer server.Close()
			preparer := &preparer{
				config:  performance.Config{APIURL: server.URL},
				options: options{Roles: 50, Permissions: 500, Aggregation: "list"},
				logger:  klog.NewStdLogger(io.Discard),
			}
			_, cleanup, err := preparer.Prepare(t.Context(), t.TempDir())
			if err == nil || cleanup == nil {
				t.Fatal("expected preflight rejection")
			}
			if len(api.rowMap["project"]) != 1 {
				t.Fatal("preflight must precede fixture writes")
			}
			err = cleanup()
			if err != nil {
				t.Fatal(err)
			}
			api.assertOnlyUnrelated(t)
		})
	}
}

// TestPermissionsCommandContract 验证默认低速率、规模参数及恢复清理入口。
func TestPermissionsCommandContract(t *testing.T) {
	command := Entrypoint.NewCobraCommand()
	flagList := []string{
		"roles", "permissions", "aggregation", "rps", "duration",
	}
	for _, name := range flagList {
		if command.Flags().Lookup(name) == nil {
			t.Fatalf("missing flag %s", name)
		}
	}
	if command.Flags().Lookup("rps").DefValue != "10" {
		t.Fatal("permissions must default to low RPS")
	}
	cleanup, _, err := command.Find([]string{"cleanup"})
	if err != nil || cleanup.Name() != "cleanup" {
		t.Fatal("missing cleanup command")
	}
	for _, option := range []options{
		{0, 500, "list"}, {51, 500, "list"},
		{10, 0, "map"}, {10, 2001, "map"}, {10, 500, "unknown"},
	} {
		if option.validate() == nil {
			t.Fatalf("invalid options accepted: %+v", option)
		}
	}
}

// TestPermissionsVegetaSmoke 使用本机 Vegeta 验证真实攻击、报告与清理链路。
func TestPermissionsVegetaSmoke(t *testing.T) {
	_, err := exec.LookPath("vegeta")
	if err != nil {
		t.Skip("Vegeta is not installed")
	}
	api := newFakePermissionAPI()
	server := httptest.NewServer(api)
	defer server.Close()
	config := performance.Config{
		APIURL: server.URL, RPS: 10, Duration: 200 * time.Millisecond,
		Warmup: 100 * time.Millisecond, OutputDir: t.TempDir(),
	}
	logger := klog.NewStdLogger(io.Discard)
	preparer := &preparer{
		config: config, logger: logger,
		options: options{Roles: 2, Permissions: 4, Aggregation: "list"},
	}
	runner := performance.NewRunner(logger)
	err = runner.Run(t.Context(), config, preparer)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(
		filepath.Join(config.OutputDir, "11-vegeta-report.txt"),
	)
	if err != nil || !strings.Contains(string(content), "100.00%") {
		t.Fatalf("Vegeta report = %s, err = %v", content, err)
	}
	manifest, err := readManifest(
		filepath.Join(config.OutputDir, "01-permissions.json"),
	)
	if err != nil || manifest.CleanedAt == nil {
		t.Fatal("Vegeta run must clean the batch")
	}
	api.assertOnlyUnrelated(t)
}
