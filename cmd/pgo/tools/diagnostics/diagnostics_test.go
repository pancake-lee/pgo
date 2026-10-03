package diagnostics

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadProfile(t *testing.T) {
	var requestedPathList []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestedPathList = append(requestedPathList, request.Method+" "+request.URL.Path)
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
	if len(requestedPathList) != 1 || requestedPathList[0] != "GET /debug/pprof/heap" {
		t.Fatalf("paths = %q", requestedPathList)
	}
}

// TestDownloadRuntimeProfileUsesSingleRequest 验证运行时 profile 仅通过一次 POST 下载。
func TestDownloadRuntimeProfileUsesSingleRequest(t *testing.T) {
	var requestedPathList []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestedPathList = append(requestedPathList, request.Method+" "+request.URL.String())
		_, _ = writer.Write([]byte("profile-data"))
	}))
	defer server.Close()

	if err := downloadProfile(server.URL, "mutex", 5, filepath.Join(t.TempDir(), "mutex.pprof")); err != nil {
		t.Fatal(err)
	}
	if len(requestedPathList) != 1 || !strings.Contains(requestedPathList[0], "POST /debug/pprof/runtime?") || !strings.Contains(requestedPathList[0], "profile=mutex") || !strings.Contains(requestedPathList[0], "seconds=5") {
		t.Fatalf("paths = %q", requestedPathList)
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
	if err := downloadProfile("http://localhost", "unknown", 0, "ignored"); err == nil {
		t.Fatal("expected unsupported profile type error")
	}
}
