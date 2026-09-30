package performance

import (
	"context"
	"encoding/json"
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
	mu       sync.Mutex
	commands []string
}

func (runner *fakeCommandRunner) Run(_ context.Context, name string, args []string, stdout, _ io.Writer) error {
	runner.mu.Lock()
	runner.commands = append(runner.commands, name+" "+strings.Join(args, " "))
	runner.mu.Unlock()
	if len(args) > 0 && args[0] == "attack" {
		_, _ = io.WriteString(stdout, "raw-result")
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
	runner := &loginRunner{
		commandRunner: commandRunner,
		httpClient:    server.Client(),
		checkVegeta:   func(string) error { return nil },
		now:           func() time.Time { return time.Date(2026, 9, 30, 1, 2, 3, 0, time.UTC) },
	}
	outputDir := t.TempDir()
	config := loginConfig{
		APIURL:      server.URL,
		PprofURL:    server.URL + "/debug/pprof/",
		Users:       3,
		Concurrency: 2,
		RPS:         5,
		Duration:    20 * time.Millisecond,
		Timeout:     time.Second,
		OutputDir:   outputDir,
		VegetaPath:  "vegeta",
	}
	var stdout strings.Builder
	if err := runner.run(t.Context(), config, &stdout, io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		runFileName, usersFileName, loginTargetsFileName, vegetaResultsFileName, vegetaReportFileName,
		metricsAfterFileName, metricsBeforeFileName, cpuProfileFileName, goroutineProfileFileName, heapProfileFileName, summaryFileName,
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
