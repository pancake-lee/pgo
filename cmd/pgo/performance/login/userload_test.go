package login

import (
	"encoding/hex"
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

	"github.com/pancake-lee/pgo/cmd/pgo/common"
)

func TestNewBatchID(t *testing.T) {
	batchID, err := newBatchID()
	if err != nil {
		t.Fatal(err)
	}
	if len(batchID) != 16 {
		t.Fatalf("batch ID length = %d, want 16", len(batchID))
	}
	_, err = strconv.ParseUint(batchID[:12], 10, 64)
	if err != nil {
		t.Fatalf("batch ID timestamp is not numeric: %q", batchID)
	}
	_, err = hex.DecodeString(batchID[12:])
	if err != nil {
		t.Fatalf("batch ID suffix is not hexadecimal: %q", batchID)
	}
}

type fakeUserServer struct {
	mu         sync.Mutex
	nextID     int32
	nameToUser map[string]User
	validToken map[string]bool
	failSuffix string
	rejected   int
}

func TestUserBatchLifecycle(t *testing.T) {
	fakeServer := newFakeUserServer()
	server := httptest.NewServer(fakeServer)
	defer server.Close()
	client, err := common.NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(t.TempDir(), "batch", "users.json")
	var userCount int = 100
	manifest, err := Prepare(t.Context(), client, userCount, manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Users) != userCount {
		t.Fatalf("prepared %d users, want %d", len(manifest.Users), userCount)
	}
	recovered := &Manifest{}
	err = recovered.read(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	err = Verify(t.Context(), client, recovered)
	if err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(t.TempDir(), "targets.json")
	err = recovered.WriteTargets(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	lineList := strings.Split(strings.TrimSpace(string(content)), "\n")
	lineCount := len(lineList)
	if lineCount != userCount {
		t.Fatalf("target line count = %d, want %d", lineCount, userCount)
	}

	err = Cleanup(t.Context(), client, recovered)
	if err != nil {
		t.Fatal(err)
	}
	err = Cleanup(t.Context(), client, recovered)
	if err != nil {
		t.Fatalf("cleanup must be idempotent: %v", err)
	}
	cleaned := &Manifest{}
	err = cleaned.read(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if cleaned.CleanedAt == nil {
		t.Fatal("cleaned manifest does not record cleanup time")
	}
	fakeServer.mu.Lock()
	defer fakeServer.mu.Unlock()
	if fakeServer.rejected != 0 {
		t.Fatalf("unexpected auth rejections: %d", fakeServer.rejected)
	}
	if len(fakeServer.nameToUser) != 0 {
		t.Fatalf("users remain after cleanup: %d", len(fakeServer.nameToUser))
	}
}

func TestPreparePersistsSuccessesWhenSomeRequestsFail(t *testing.T) {
	fakeServer := newFakeUserServer()
	fakeServer.failSuffix = "_000003"
	server := httptest.NewServer(fakeServer)
	defer server.Close()
	client, err := common.NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(t.TempDir(), "users.json")
	var userCount int = 100
	manifest, err := Prepare(t.Context(), client, userCount, manifestPath)
	if err == nil {
		t.Fatal("expected aggregated prepare error")
	}
	if len(manifest.Users) != userCount-1 {
		t.Fatalf("persisted users = %d, want %d", len(manifest.Users), userCount-1)
	}
	recovered := &Manifest{}
	err = recovered.read(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(recovered.Users) != userCount-1 {
		t.Fatalf("recovered users = %d, want %d", len(recovered.Users), userCount-1)
	}
}

func newFakeUserServer() *fakeUserServer {
	return &fakeUserServer{
		nextID:     10,
		nameToUser: make(map[string]User),
		validToken: make(map[string]bool),
	}
}

func (server *fakeUserServer) ServeHTTP(
	writer http.ResponseWriter,
	request *http.Request,
) {
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

func (server *fakeUserServer) login(
	writer http.ResponseWriter,
	request *http.Request,
) {
	writer.Header().Set("Content-Type", "application/json")
	var input struct {
		UserName string `json:"userName"`
	}
	err := json.NewDecoder(request.Body).Decode(&input)
	if err != nil || input.UserName == "" {
		http.Error(writer, "bad request", http.StatusBadRequest)
		return
	}
	shouldFail := server.failSuffix != "" &&
		strings.HasSuffix(input.UserName, server.failSuffix)
	if shouldFail {
		http.Error(writer, "planned failure", http.StatusServiceUnavailable)
		return
	}
	server.mu.Lock()
	user, ok := server.nameToUser[input.UserName]
	if !ok {
		user = User{
			ID:       server.nextID,
			UserName: input.UserName,
			Token:    fmt.Sprintf("token-%d", server.nextID),
		}
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
	if !server.validToken[token] {
		server.rejected++
	}
	return server.validToken[token]
}

func (server *fakeUserServer) getUser(
	writer http.ResponseWriter,
	request *http.Request,
) {
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

func (server *fakeUserServer) delUser(
	writer http.ResponseWriter,
	request *http.Request,
) {
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
