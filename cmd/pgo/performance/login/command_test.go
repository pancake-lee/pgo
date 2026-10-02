package login

import (
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	klog "github.com/go-kratos/kratos/v2/log"
	performance "github.com/pancake-lee/pgo/cmd/pgo/performance/core"
)

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

func TestPreparerCreatesTargetsAndCleansUsers(t *testing.T) {
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
	if cleanup == nil {
		t.Fatal("cleanup is nil")
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
	err = cleanup()
	if err != nil {
		t.Fatal(err)
	}
	fakeServer.mu.Lock()
	defer fakeServer.mu.Unlock()
	if len(fakeServer.nameToUser) != 0 {
		t.Fatalf("users remain after cleanup: %d", len(fakeServer.nameToUser))
	}
}
