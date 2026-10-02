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

	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/spf13/pflag"
)

type fakeCommandRunner struct {
	mu               sync.Mutex
	commands         []string
	failRate         string
	unsuccessfulRate string
}

func (runner *fakeCommandRunner) Run(_ context.Context, name string, args []string, resultWriter io.Writer) error {
	runner.mu.Lock()
	runner.commands = append(runner.commands, name+" "+strings.Join(args, " "))
	runner.mu.Unlock()
	if len(args) > 0 && args[0] == "attack" {
		if runner.failRate != "" && strings.Contains(strings.Join(args, " "), "-rate="+runner.failRate+"/s") {
			return errors.New("injected attack failure")
		}
		_, _ = io.WriteString(resultWriter, "raw-result")
		return nil
	}
	if runner.unsuccessfulRate != "" && strings.Contains(strings.Join(args, " "), "rps-"+runner.unsuccessfulRate) {
		_, _ = io.WriteString(resultWriter, "Requests      [total, rate, throughput]  1, 1.00, 1.00\nSuccess       [ratio]                     90.00%\n")
		return nil
	}
	_, _ = io.WriteString(resultWriter, "Requests      [total, rate, throughput]  1, 1.00, 1.00\nSuccess       [ratio]                     100.00%\n")
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
	var output strings.Builder
	runner := &loadRunner{
		commandRunner: commandRunner,
		httpClient:    server.Client(),
		checkVegeta:   func(string) error { return nil },
		now:           func() time.Time { return time.Date(2026, 9, 30, 1, 2, 3, 0, time.UTC) },
		logger:        klog.NewStdLogger(&output),
	}
	outputDir := t.TempDir()
	config := loadConfig{RPS: 10, RPSInput: "auto"}
	config.APIURL = server.URL
	config.RPS = 5
	config.RPSInput = "5"
	config.OutputDir = outputDir
	if err := runner.runLoginStage(t.Context(), config); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		runFileName, usersFileName, loginTargetsFileName, vegetaResultsFileName, vegetaReportFileName,
	} {
		if _, err := os.Stat(filepath.Join(outputDir, name)); err != nil {
			t.Errorf("artifact %s: %v", name, err)
		}
	}
	for _, name := range []string{
		"20-metrics-after.prom", "21-metrics-before.prom", "30-cpu.pprof", "31-goroutine.pprof",
		"32-heap.pprof", "33-block.pprof", "34-mutex.pprof", "35-runtime.trace", "40-summary.md",
	} {
		if _, err := os.Stat(filepath.Join(outputDir, name)); !os.IsNotExist(err) {
			t.Errorf("obsolete observability artifact %s exists: %v", name, err)
		}
	}
	if !strings.Contains(output.String(), vegetaReportFileName) {
		t.Fatalf("log does not explain artifacts: %s", output.String())
	}
	if !strings.Contains(output.String(), "outputDir="+outputDir) {
		t.Fatalf("log does not print output directory: %s", output.String())
	}
	if strings.Contains(output.String(), filepath.Join(outputDir, vegetaReportFileName)) {
		t.Fatalf("artifact log repeats parent directory: %s", output.String())
	}
	runConfig, err := os.ReadFile(filepath.Join(outputDir, runFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(runConfig), `"rps": 5`) || !strings.Contains(string(runConfig), `"apiURL"`) {
		t.Fatalf("run config does not contain common and login fields: %s", runConfig)
	}
	if strings.Contains(string(runConfig), `"users"`) || strings.Contains(string(runConfig), `"concurrency"`) {
		t.Fatalf("run config exposes internal user preparation policy: %s", runConfig)
	}
	for _, internalDefault := range []string{`"warmup"`, `"duration"`, `"timeout"`, `"vegetaPath"`} {
		if strings.Contains(string(runConfig), internalDefault) {
			t.Fatalf("run config exposes fixed internal default %q: %s", internalDefault, runConfig)
		}
	}
	for _, obsolete := range []string{"pprofURL", "pyroscopeURL", "grafanaURL", "profileCLIPath", "profileService", "runtimeTrace"} {
		if strings.Contains(string(runConfig), obsolete) {
			t.Fatalf("run config contains obsolete observability field %q: %s", obsolete, runConfig)
		}
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
		commandRunner: commandRunner,
		httpClient:    server.Client(),
		checkVegeta:   func(string) error { return nil },
		now:           func() time.Time { return time.Date(2026, 10, 1, 1, 2, 3, 0, time.UTC) },
		logger:        klog.NewStdLogger(io.Discard),
	}
	outputDir := t.TempDir()
	config := loadConfig{RPS: 10, RPSInput: "auto"}
	config.APIURL = server.URL
	config.RPSInput = "auto"
	config.OutputDir = outputDir
	if err := runLoginPlan(t.Context(), runner, config); err != nil {
		t.Fatal(err)
	}
	for _, rps := range autoRPSList {
		stageDir := filepath.Join(outputDir, fmt.Sprintf("rps-%03d", rps))
		if _, err := os.Stat(filepath.Join(stageDir, vegetaReportFileName)); err != nil {
			t.Errorf("stage %d report: %v", rps, err)
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
	if !strings.Contains(string(summary), "| 500 |") || strings.Contains(string(summary), "Service and runtime trend") {
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
		commandRunner: &fakeCommandRunner{failRate: "25"},
		httpClient:    server.Client(),
		checkVegeta:   func(string) error { return nil },
		now:           time.Now,
		logger:        klog.NewStdLogger(io.Discard),
	}
	outputDir := t.TempDir()
	config := loadConfig{RPS: 10, RPSInput: "auto"}
	config.APIURL = server.URL
	config.RPSInput = "auto"
	config.OutputDir = outputDir
	err := runLoginPlan(t.Context(), runner, config)
	if err == nil || !strings.Contains(err.Error(), "25 RPS") {
		t.Fatalf("automatic error = %v", err)
	}
	if strings.Contains(err.Error(), vegetaReportFileName) {
		t.Fatalf("automatic error contains secondary missing-report failure: %v", err)
	}
	if _, err = os.Stat(filepath.Join(outputDir, "rps-010", vegetaReportFileName)); err != nil {
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
		commandRunner: &fakeCommandRunner{unsuccessfulRate: "025"},
		httpClient:    server.Client(),
		checkVegeta:   func(string) error { return nil },
		now:           time.Now,
		logger:        klog.NewStdLogger(io.Discard),
	}
	config := loadConfig{RPS: 10, RPSInput: "auto"}
	config.APIURL = server.URL
	config.RPSInput = "auto"
	config.OutputDir = t.TempDir()
	err := runLoginPlan(t.Context(), runner, config)
	if err == nil || !strings.Contains(err.Error(), "success ratio 90.00%") {
		t.Fatalf("automatic response error = %v", err)
	}
	if _, err = os.Stat(filepath.Join(config.OutputDir, "rps-050")); !os.IsNotExist(err) {
		t.Fatalf("later stage should not run after unsuccessful responses: %v", err)
	}
}

func TestPerformanceCommandContainsLoginOnly(t *testing.T) {
	command := NewCommand()
	if len(command.Commands()) != 1 || command.Commands()[0].Name() != "login" {
		t.Fatalf("performance commands = %v", command.Commands())
	}
	loginCommand := command.Commands()[0]
	if loginCommand.Use != "login <portal-url>" {
		t.Fatalf("login use = %q", loginCommand.Use)
	}
	if loginCommand.Flags().Lookup("rps") == nil {
		t.Fatalf("login flags = %v", loginCommand.Flags())
	}
	loginCommand.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag.Name != "rps" {
			t.Errorf("unexpected login flag %q", flag.Name)
		}
	})
}

func TestFixedDependencyErrorsIncludeInstallGuidance(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing-tool")
	if err := checkVegetaVersion(missingPath); err == nil || !strings.Contains(err.Error(), "go install github.com/tsenart/vegeta/v12@v12.13.0") {
		t.Fatalf("Vegeta error = %v", err)
	}
}

func TestExecRunnerCapturesStderr(t *testing.T) {
	t.Setenv("PGO_EXEC_RUNNER_HELPER", "1")
	runner := execRunner{logger: klog.NewStdLogger(io.Discard)}
	err := runner.Run(t.Context(), os.Args[0], []string{"-test.run=TestExecRunnerHelper"}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "vegeta failed") {
		t.Fatalf("command error = %v", err)
	}
}

func TestExecRunnerHelper(t *testing.T) {
	if os.Getenv("PGO_EXEC_RUNNER_HELPER") != "1" {
		return
	}
	_, _ = fmt.Fprint(os.Stderr, "vegeta failed")
	os.Exit(2)
}

func (server *fakePerformanceServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
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
