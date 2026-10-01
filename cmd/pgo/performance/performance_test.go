package performance

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
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeCommandRunner struct {
	mu               sync.Mutex
	commands         []string
	failRate         string
	unsuccessfulRate string
}

func (runner *fakeCommandRunner) Run(_ context.Context, name string, args []string, stdout, _ io.Writer) error {
	runner.mu.Lock()
	runner.commands = append(runner.commands, name+" "+strings.Join(args, " "))
	runner.mu.Unlock()
	if len(args) > 0 && args[0] == "attack" {
		if runner.failRate != "" && strings.Contains(strings.Join(args, " "), "-rate="+runner.failRate+"/s") {
			return errors.New("injected attack failure")
		}
		_, _ = io.WriteString(stdout, "raw-result")
		return nil
	}
	if name == "profilecli" {
		for _, argument := range args {
			if strings.HasPrefix(argument, "--output=pprof=") {
				return os.WriteFile(strings.TrimPrefix(argument, "--output=pprof="), []byte("profile-data"), 0o600)
			}
		}
	}
	if runner.unsuccessfulRate != "" && strings.Contains(strings.Join(args, " "), "rps-"+runner.unsuccessfulRate) {
		_, _ = io.WriteString(stdout, "Requests      [total, rate, throughput]  1, 1.00, 1.00\nSuccess       [ratio]                     90.00%\n")
		return nil
	}
	_, _ = io.WriteString(stdout, "Requests      [total, rate, throughput]  1, 1.00, 1.00\nSuccess       [ratio]                     100.00%\n")
	return nil
}

type fakePerformanceServer struct {
	mu         sync.Mutex
	nextID     int32
	nameToUser map[string]fakeUser
}

type fakeUser struct {
	ID       int32
	UserName string
	Token    string
}

func TestLoginRunnerCompletesWorkflow(t *testing.T) {
	fakeServer := &fakePerformanceServer{nextID: 10, nameToUser: make(map[string]fakeUser)}
	server := httptest.NewServer(fakeServer)
	defer server.Close()
	commandRunner := &fakeCommandRunner{}
	runner := &loadRunner{
		commandRunner:   commandRunner,
		httpClient:      server.Client(),
		checkVegeta:     func(string) error { return nil },
		checkProfileCLI: func(string) error { return nil },
		now:             func() time.Time { return time.Date(2026, 9, 30, 1, 2, 3, 0, time.UTC) },
	}
	outputDir := t.TempDir()
	config := defaultLoginConfig()
	config.APIURL = server.URL
	config.PprofURL = server.URL + "/debug/pprof/"
	config.GrafanaURL = server.URL
	config.ProfileSource = profileSourcePprof
	config.Users = 3
	config.Concurrency = 2
	config.RPS = 5
	config.RPSInput = "5"
	config.Warmup = 0
	config.Duration = 20 * time.Millisecond
	config.Timeout = time.Second
	config.OutputDir = outputDir
	config.VegetaPath = "vegeta"
	var stdout strings.Builder
	if err := runner.runLoginStage(t.Context(), config, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		runFileName, usersFileName, loginTargetsFileName, vegetaResultsFileName, vegetaReportFileName,
		metricsAfterFileName, metricsBeforeFileName, cpuProfileFileName, goroutineProfileFileName, heapProfileFileName, summaryFileName,
		blockProfileFileName, mutexProfileFileName,
	} {
		if _, err := os.Stat(filepath.Join(outputDir, name)); err != nil {
			t.Errorf("artifact %s: %v", name, err)
		}
	}
	if !strings.Contains(stdout.String(), summaryFileName) {
		t.Fatalf("stdout does not explain artifacts: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Output directory: "+outputDir) {
		t.Fatalf("stdout does not print output directory: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), filepath.Join(outputDir, summaryFileName)) {
		t.Fatalf("artifact line repeats parent directory: %s", stdout.String())
	}
	runConfig, err := os.ReadFile(filepath.Join(outputDir, runFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(runConfig), `"rps": 5`) || !strings.Contains(string(runConfig), `"apiURL"`) {
		t.Fatalf("run config does not contain common and login fields: %s", runConfig)
	}
	summary, err := os.ReadFile(filepath.Join(outputDir, summaryFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(summary), "Successful HTTP login requests") || strings.Contains(string(summary), "_bucket{") {
		t.Fatalf("summary is not condensed: %s", summary)
	}
	fakeServer.mu.Lock()
	defer fakeServer.mu.Unlock()
	if len(fakeServer.nameToUser) != 0 {
		t.Fatalf("users remain after cleanup: %d", len(fakeServer.nameToUser))
	}
}

func TestRunAutomaticLoginCompletesBuiltInLadder(t *testing.T) {
	fakeServer := &fakePerformanceServer{nextID: 100, nameToUser: make(map[string]fakeUser)}
	server := httptest.NewServer(fakeServer)
	defer server.Close()
	commandRunner := &fakeCommandRunner{}
	runner := &loadRunner{
		commandRunner:   commandRunner,
		httpClient:      server.Client(),
		checkVegeta:     func(string) error { return nil },
		checkProfileCLI: func(string) error { return nil },
		now:             func() time.Time { return time.Date(2026, 10, 1, 1, 2, 3, 0, time.UTC) },
	}
	outputDir := t.TempDir()
	config := defaultLoginConfig()
	config.APIURL = server.URL
	config.PprofURL = server.URL + "/debug/pprof/"
	config.GrafanaURL = server.URL
	config.ProfileSource = profileSourcePprof
	config.Users = 2
	config.Concurrency = 1
	config.RPSInput = "auto"
	config.Warmup = 0
	config.Duration = 10 * time.Millisecond
	config.Timeout = time.Second
	config.OutputDir = outputDir
	config.VegetaPath = "vegeta"
	if err := runLoginPlan(t.Context(), runner, config, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, rps := range autoRPSList {
		stageDir := filepath.Join(outputDir, fmt.Sprintf("rps-%03d", rps))
		if _, err := os.Stat(filepath.Join(stageDir, summaryFileName)); err != nil {
			t.Errorf("stage %d summary: %v", rps, err)
		}
	}
	for _, name := range []string{autoPlanFileName, autoResultFileName, autoSummaryFileName} {
		if _, err := os.Stat(filepath.Join(outputDir, name)); err != nil {
			t.Errorf("automatic artifact %s: %v", name, err)
		}
	}
	summary, err := os.ReadFile(filepath.Join(outputDir, autoSummaryFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(summary), "| 500 |") || !strings.Contains(string(summary), "Service and runtime trend") {
		t.Fatalf("automatic summary = %s", summary)
	}
	fakeServer.mu.Lock()
	defer fakeServer.mu.Unlock()
	if len(fakeServer.nameToUser) != 0 {
		t.Fatalf("users remain after automatic cleanup: %d", len(fakeServer.nameToUser))
	}
}

func TestResolveRPSList(t *testing.T) {
	rpsList, automatic, err := resolveRPSList("auto", 10)
	if err != nil || !automatic || fmt.Sprint(rpsList) != fmt.Sprint(autoRPSList) {
		t.Fatalf("auto RPS = %v, %v, %v", rpsList, automatic, err)
	}
	rpsList, automatic, err = resolveRPSList("75", 10)
	if err != nil || automatic || len(rpsList) != 1 || rpsList[0] != 75 {
		t.Fatalf("single RPS = %v, %v, %v", rpsList, automatic, err)
	}
	if _, _, err = resolveRPSList("invalid", 10); err == nil {
		t.Fatal("expected invalid RPS error")
	}
}

func TestRunAutomaticLoginStopsAfterFailedStage(t *testing.T) {
	fakeServer := &fakePerformanceServer{nextID: 200, nameToUser: make(map[string]fakeUser)}
	server := httptest.NewServer(fakeServer)
	defer server.Close()
	runner := &loadRunner{
		commandRunner:   &fakeCommandRunner{failRate: "25"},
		httpClient:      server.Client(),
		checkVegeta:     func(string) error { return nil },
		checkProfileCLI: func(string) error { return nil },
		now:             time.Now,
	}
	outputDir := t.TempDir()
	config := defaultLoginConfig()
	config.APIURL = server.URL
	config.PprofURL = server.URL + "/debug/pprof/"
	config.GrafanaURL = server.URL
	config.ProfileSource = profileSourcePprof
	config.Users = 1
	config.Concurrency = 1
	config.RPSInput = "auto"
	config.Warmup = 0
	config.Duration = 10 * time.Millisecond
	config.Timeout = time.Second
	config.OutputDir = outputDir
	config.VegetaPath = "vegeta"
	err := runLoginPlan(t.Context(), runner, config, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "25 RPS") {
		t.Fatalf("automatic error = %v", err)
	}
	if _, err = os.Stat(filepath.Join(outputDir, "rps-010", summaryFileName)); err != nil {
		t.Fatalf("completed stage missing: %v", err)
	}
	if _, err = os.Stat(filepath.Join(outputDir, "rps-100")); !os.IsNotExist(err) {
		t.Fatalf("later stage should not run: %v", err)
	}
	fakeServer.mu.Lock()
	defer fakeServer.mu.Unlock()
	if len(fakeServer.nameToUser) != 0 {
		t.Fatalf("users remain after failed automatic run: %d", len(fakeServer.nameToUser))
	}
}

func TestRunAutomaticLoginStopsAfterUnsuccessfulResponses(t *testing.T) {
	fakeServer := &fakePerformanceServer{nextID: 300, nameToUser: make(map[string]fakeUser)}
	server := httptest.NewServer(fakeServer)
	defer server.Close()
	runner := &loadRunner{
		commandRunner:   &fakeCommandRunner{unsuccessfulRate: "025"},
		httpClient:      server.Client(),
		checkVegeta:     func(string) error { return nil },
		checkProfileCLI: func(string) error { return nil },
		now:             time.Now,
	}
	config := defaultLoginConfig()
	config.APIURL = server.URL
	config.PprofURL = server.URL + "/debug/pprof/"
	config.GrafanaURL = server.URL
	config.ProfileSource = profileSourcePprof
	config.Users = 1
	config.Concurrency = 1
	config.RPSInput = "auto"
	config.Warmup = 0
	config.Duration = 10 * time.Millisecond
	config.Timeout = time.Second
	config.OutputDir = t.TempDir()
	config.VegetaPath = "vegeta"
	err := runLoginPlan(t.Context(), runner, config, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "success ratio 90.00%") {
		t.Fatalf("automatic response error = %v", err)
	}
	if _, err = os.Stat(filepath.Join(config.OutputDir, "rps-050")); !os.IsNotExist(err) {
		t.Fatalf("later stage should not run after unsuccessful responses: %v", err)
	}
}

func TestCollectPyroscopeProfiles(t *testing.T) {
	commandRunner := &fakeCommandRunner{}
	runner := &loadRunner{commandRunner: commandRunner}
	config := defaultLoginConfig()
	config.PyroscopeURL = "http://pyroscope:4040"
	config.ProfileCLIPath = "profilecli"
	config.ProfileService = "user-service"
	config.OutputDir = t.TempDir()
	start := time.Unix(100, 0)
	end := time.Unix(200, 0)
	if err := runner.collectPyroscopeProfiles(t.Context(), config.loadConfig, start, end, io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{cpuProfileFileName, heapProfileFileName} {
		if _, err := os.Stat(filepath.Join(config.OutputDir, name)); err != nil {
			t.Fatalf("profile %s: %v", name, err)
		}
	}
	commandRunner.mu.Lock()
	commandList := append([]string(nil), commandRunner.commands...)
	commandRunner.mu.Unlock()
	joined := strings.Join(commandList, "\n")
	if !strings.Contains(joined, "--from=100") || !strings.Contains(joined, "--to=200") || !strings.Contains(joined, `--query={service_name="user-service"}`) {
		t.Fatalf("profilecli commands = %s", joined)
	}
}

func TestValidateLoginConfigRejectsProfileSource(t *testing.T) {
	config := defaultLoginConfig()
	config.ProfileSource = "unknown"
	if err := validateLoginConfig(config); err == nil {
		t.Fatal("expected profile source error")
	}
}

func TestPerformanceCommandContainsLoginOnly(t *testing.T) {
	command := NewCommand()
	if len(command.Commands()) != 1 || command.Commands()[0].Name() != "login" {
		t.Fatalf("performance commands = %v", command.Commands())
	}
}

func TestPerformanceURLs(t *testing.T) {
	pprofURL := "http://127.0.0.1:19090/debug/pprof/"
	if actual := joinPprofURL(pprofURL, "heap"); actual != "http://127.0.0.1:19090/debug/pprof/heap" {
		t.Fatalf("pprof URL = %s", actual)
	}
	if actual := metricsURL(pprofURL); actual != "http://127.0.0.1:19090/metrics" {
		t.Fatalf("metrics URL = %s", actual)
	}
}

func (server *fakePerformanceServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path == "/metrics" {
		_, _ = io.WriteString(writer, "pgo_http_requests_total 3\ngo_goroutines 5\n")
		return
	}
	if strings.HasPrefix(request.URL.Path, "/debug/pprof/") {
		if request.URL.Path == "/debug/pprof/runtime" && request.Method != http.MethodPost {
			http.Error(writer, "method", http.StatusMethodNotAllowed)
			return
		}
		_, _ = io.WriteString(writer, "profile-data")
		return
	}
	if request.URL.Path == "/user/token" && request.Method == http.MethodPost {
		server.login(writer, request)
		return
	}
	if request.URL.Path == "/user" {
		if !server.authorized(request) {
			http.Error(writer, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch request.Method {
		case http.MethodGet:
			server.getUser(writer, request)
		case http.MethodDelete:
			server.delUser(writer, request)
		}
		return
	}
	http.NotFound(writer, request)
}

func (server *fakePerformanceServer) login(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		UserName string `json:"userName"`
	}
	_ = json.NewDecoder(request.Body).Decode(&input)
	server.mu.Lock()
	user, exists := server.nameToUser[input.UserName]
	if !exists {
		user = fakeUser{ID: server.nextID, UserName: input.UserName, Token: fmt.Sprintf("token-%d", server.nextID)}
		server.nextID++
		server.nameToUser[input.UserName] = user
	}
	server.mu.Unlock()
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(map[string]any{
		"user":  map[string]any{"ID": user.ID, "userName": user.UserName},
		"token": user.Token,
	})
}

func (server *fakePerformanceServer) authorized(request *http.Request) bool {
	token := strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer ")
	server.mu.Lock()
	defer server.mu.Unlock()
	for _, user := range server.nameToUser {
		if user.Token == token {
			return true
		}
	}
	return false
}

func (server *fakePerformanceServer) getUser(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseInt(request.URL.Query().Get("IDList"), 10, 32)
	server.mu.Lock()
	defer server.mu.Unlock()
	for _, user := range server.nameToUser {
		if user.ID == int32(id) {
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"userList": []map[string]any{{"ID": user.ID, "userName": user.UserName}},
			})
			return
		}
	}
	_ = json.NewEncoder(writer).Encode(map[string]any{"userList": []any{}})
}

func (server *fakePerformanceServer) delUser(writer http.ResponseWriter, request *http.Request) {
	id, _ := strconv.ParseInt(request.URL.Query().Get("IDList"), 10, 32)
	server.mu.Lock()
	defer server.mu.Unlock()
	for name, user := range server.nameToUser {
		if user.ID == int32(id) {
			delete(server.nameToUser, name)
		}
	}
	writer.WriteHeader(http.StatusOK)
}
