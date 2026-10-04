package login

import (
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/pancake-lee/pgo/cmd/pgo/common"
	performance "github.com/pancake-lee/pgo/cmd/pgo/performance/core"
)

// TestLoginCommandContract 验证测试和清理的命令入口。
func TestLoginCommandContract(t *testing.T) {
	command := Entrypoint.NewCobraCommand()
	if command.Use != "login <portal-url>" {
		t.Fatalf("login use = %q", command.Use)
	}
	if command.Flags().Lookup("rps") == nil ||
		command.Flags().Lookup("duration") == nil {
		t.Fatalf("login flags = %v", command.Flags())
	}
	entrypointCommand := Entrypoint.NewCobraCommand()
	if entrypointCommand.Name() != "login" {
		t.Fatalf("entrypoint command = %q", entrypointCommand.Name())
	}
}

// TestPreparerReusesTargetsAndCleansUsers 验证默认保留并复用用户且可独立清理。
func TestPreparerReusesTargetsAndCleansUsers(t *testing.T) {
	fakeServer := newFakeUserServer()
	server := httptest.NewServer(fakeServer)
	defer server.Close()
	outputDir := t.TempDir()
	config := performance.Config{
		APIURL:    server.URL,
		RPS:       10,
		OutputDir: outputDir,
	}
	preparer := newPreparer(config, klog.NewStdLogger(io.Discard))
	targetPath, cleanup, err := preparer.Prepare(t.Context(), outputDir)
	if err != nil {
		t.Fatal(err)
	}
	if cleanup != nil {
		t.Fatal("default run must retain users")
	}
	batch := preparer.manifest.BatchID
	second := newPreparer(config, klog.NewStdLogger(io.Discard))
	_, _, err = second.Prepare(t.Context(), outputDir)
	if err != nil || second.manifest.BatchID != batch {
		t.Fatalf("batch was not reused: %v", err)
	}
	pathList := []string{
		filepath.Join(outputDir, usersFileName),
		targetPath,
	}
	for _, path := range pathList {
		_, err = os.Stat(path)
		if err != nil {
			t.Errorf("scenario artifact %s: %v", path, err)
		}
	}
	err = second.CleanupData(t.Context(), outputDir)
	if err != nil {
		t.Fatal(err)
	}
	fakeServer.mu.Lock()
	defer fakeServer.mu.Unlock()
	if len(fakeServer.nameToUser) != 0 {
		t.Fatalf("users remain after cleanup: %d", len(fakeServer.nameToUser))
	}
}

// TestLoginInsufficientDataRebuildAndCleanupRetry 验证规模不足先清理，失败后可重试。
func TestLoginInsufficientDataRebuildAndCleanupRetry(t *testing.T) {
	fakeServer := newFakeUserServer()
	server := httptest.NewServer(fakeServer)
	defer server.Close()
	directory := t.TempDir()
	client, err := common.NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	old, err := Prepare(t.Context(), client, 3,
		filepath.Join(directory, usersFileName))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = client.Login(t.Context(), "existing_user")
	if err != nil {
		t.Fatal(err)
	}
	config := performance.Config{APIURL: server.URL, OutputDir: directory}
	fakeServer.mu.Lock()
	fakeServer.failDelete = true
	fakeServer.mu.Unlock()
	preparer := newPreparer(config, klog.NewStdLogger(io.Discard))
	_, _, err = preparer.Prepare(t.Context(), directory)
	if err == nil {
		t.Fatal("cleanup failure must stop rebuilding")
	}
	retained := &Manifest{}
	err = retained.read(filepath.Join(directory, usersFileName))
	if err != nil || retained.BatchID != old.BatchID || !retained.Cleaning {
		t.Fatalf("old batch lost after cleanup failure: %v", err)
	}
	fakeServer.mu.Lock()
	fakeServer.failDelete = false
	fakeServer.mu.Unlock()
	_, _, err = preparer.Prepare(t.Context(), directory)
	if err != nil || preparer.manifest.BatchID == old.BatchID {
		t.Fatalf("rebuild failed: %v", err)
	}
	fakeServer.mu.Lock()
	if len(fakeServer.nameToUser) != loginUserCount+1 {
		t.Fatalf("old or unrelated users mishandled: %d", len(fakeServer.nameToUser))
	}
	fakeServer.mu.Unlock()
	err = preparer.CleanupData(t.Context(), directory)
	if err != nil {
		t.Fatal(err)
	}
	fakeServer.mu.Lock()
	loginCount := fakeServer.loginCount
	fakeServer.mu.Unlock()
	err = preparer.CleanupData(t.Context(), directory)
	if err != nil {
		t.Fatal(err)
	}
	fakeServer.mu.Lock()
	defer fakeServer.mu.Unlock()
	if len(fakeServer.nameToUser) != 1 || fakeServer.loginCount != loginCount {
		t.Fatal("repeated cleanup created users or deleted unrelated users")
	}
}

// TestLoginCheckFailureRetainsBatch 验证 HTTP 检查错误不会清理或覆盖批次。
func TestLoginCheckFailureRetainsBatch(t *testing.T) {
	fakeServer := newFakeUserServer()
	server := httptest.NewServer(fakeServer)
	defer server.Close()
	directory := t.TempDir()
	config := performance.Config{APIURL: server.URL, OutputDir: directory}
	preparer := newPreparer(config, klog.NewStdLogger(io.Discard))
	_, _, err := preparer.Prepare(t.Context(), directory)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(directory, usersFileName))
	if err != nil {
		t.Fatal(err)
	}
	fakeServer.mu.Lock()
	fakeServer.failGet = true
	fakeServer.mu.Unlock()
	_, _, err = preparer.Prepare(t.Context(), directory)
	if err == nil {
		t.Fatal("expected failed data check")
	}
	after, err := os.ReadFile(filepath.Join(directory, usersFileName))
	if err != nil || string(before) != string(after) {
		t.Fatal("check error overwrote the batch")
	}
	fakeServer.mu.Lock()
	defer fakeServer.mu.Unlock()
	if len(fakeServer.nameToUser) != loginUserCount {
		t.Fatal("check error deleted users")
	}
}
