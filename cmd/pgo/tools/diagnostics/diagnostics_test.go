package diagnostics

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadProfile(t *testing.T) {
	requestedPath := ""
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestedPath = request.URL.Path
		_, _ = writer.Write([]byte("profile-data"))
	}))
	defer server.Close()

	output := filepath.Join(t.TempDir(), "heap.pprof")
	if err := downloadProfile(server.URL, "heap", 0, output); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "profile-data" {
		t.Fatalf("profile = %q", content)
	}
	if requestedPath != "/debug/pprof/heap" {
		t.Fatalf("path = %q", requestedPath)
	}
}

func TestDownloadCPUProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/debug/pprof/profile" || request.URL.Query().Get("seconds") != "1" {
			t.Fatalf("request = %s", request.URL.String())
		}
		_, _ = writer.Write([]byte("profile-data"))
	}))
	defer server.Close()

	if err := downloadProfile(server.URL, "cpu", 1, filepath.Join(t.TempDir(), "cpu.pprof")); err != nil {
		t.Fatal(err)
	}
}

func TestDownloadProfileRejectsUnknownType(t *testing.T) {
	if err := downloadProfile("http://localhost", "mutex", 0, "ignored"); err == nil {
		t.Fatal("expected unsupported profile type error")
	}
}
