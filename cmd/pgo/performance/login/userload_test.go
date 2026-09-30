package login

import (
	"encoding/json"
	"fmt"
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

type fakeUserServer struct {
	mu         sync.Mutex
	nextID     int32
	nameToUser map[string]User
	validToken map[string]bool
	failName   string
}

func TestUserBatchLifecycle(t *testing.T) {
	fakeServer := newFakeUserServer()
	server := httptest.NewServer(fakeServer)
	defer server.Close()
	client, err := NewClient(server.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(t.TempDir(), "batch", "users.json")
	manifest, err := Prepare(t.Context(), client, manifestPath, "batch1", 12, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Users) != 12 {
		t.Fatalf("prepared %d users, want 12", len(manifest.Users))
	}
	recovered, err := ReadManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err = Verify(t.Context(), client, recovered, 3); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(t.TempDir(), "targets.json")
	if err = WriteTargets(targetPath, recovered); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if lineCount := len(strings.Split(strings.TrimSpace(string(content)), "\n")); lineCount != 12 {
		t.Fatalf("target line count = %d, want 12", lineCount)
	}

	if err = Cleanup(t.Context(), client, recovered, 4); err != nil {
		t.Fatal(err)
	}
	if err = Cleanup(t.Context(), client, recovered, 4); err != nil {
		t.Fatalf("cleanup must be idempotent: %v", err)
	}
	fakeServer.mu.Lock()
	defer fakeServer.mu.Unlock()
	if len(fakeServer.nameToUser) != 0 {
		t.Fatalf("users remain after cleanup: %d", len(fakeServer.nameToUser))
	}
}

func TestPreparePersistsSuccessesWhenSomeRequestsFail(t *testing.T) {
	fakeServer := newFakeUserServer()
	fakeServer.failName = "load_partial_000003"
	server := httptest.NewServer(fakeServer)
	defer server.Close()
	client, err := NewClient(server.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(t.TempDir(), "users.json")
	manifest, err := Prepare(t.Context(), client, manifestPath, "partial", 5, 3)
	if err == nil {
		t.Fatal("expected aggregated prepare error")
	}
	if len(manifest.Users) != 4 {
		t.Fatalf("persisted users = %d, want 4", len(manifest.Users))
	}
	recovered, readErr := ReadManifest(manifestPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(recovered.Users) != 4 {
		t.Fatalf("recovered users = %d, want 4", len(recovered.Users))
	}
}

func newFakeUserServer() *fakeUserServer {
	return &fakeUserServer{
		nextID:     10,
		nameToUser: make(map[string]User),
		validToken: make(map[string]bool),
	}
}

func (server *fakeUserServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
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
		default:
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	http.NotFound(writer, request)
}

func (server *fakeUserServer) login(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	var input struct {
		UserName string `json:"userName"`
	}
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil || input.UserName == "" {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}
	if input.UserName == server.failName {
		http.Error(writer, "planned failure", http.StatusServiceUnavailable)
		return
	}
	server.mu.Lock()
	user, ok := server.nameToUser[input.UserName]
	if !ok {
		user = User{ID: server.nextID, UserName: input.UserName, Token: fmt.Sprintf("token-%d", server.nextID)}
		server.nextID++
		server.nameToUser[input.UserName] = user
	}
	server.validToken[user.Token] = true
	server.mu.Unlock()
	_ = json.NewEncoder(writer).Encode(map[string]any{
		"user":  map[string]any{"ID": user.ID, "userName": user.UserName},
		"token": user.Token,
	})
}

func (server *fakeUserServer) authorized(request *http.Request) bool {
	token := strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer ")
	server.mu.Lock()
	defer server.mu.Unlock()
	return server.validToken[token]
}

func (server *fakeUserServer) getUser(writer http.ResponseWriter, request *http.Request) {
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

func (server *fakeUserServer) delUser(writer http.ResponseWriter, request *http.Request) {
	id, _ := strconv.ParseInt(request.URL.Query().Get("IDList"), 10, 32)
	server.mu.Lock()
	defer server.mu.Unlock()
	for name, user := range server.nameToUser {
		if user.ID == int32(id) {
			delete(server.nameToUser, name)
			break
		}
	}
	writer.WriteHeader(http.StatusOK)
}
