package performance

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

const validServiceManifest = `{
  "services": [
    {"id":"api","name":"API","port":20000,"path":"/","description":"API"},
    {"id":"diagnostics","name":"pprof","port":20002,"path":"/debug/pprof/","webPath":"/debug/pprof/heap","description":"pprof"},
    {"id":"rabbitmq","name":"RabbitMQ","port":25674,"path":"/","description":"RabbitMQ"},
    {"id":"swagger","name":"Swagger","port":28080,"path":"/","description":"Swagger"},
    {"id":"prometheus","name":"Prometheus","port":29090,"path":"/","description":"Prometheus"},
    {"id":"grafana","name":"Grafana","port":23000,"path":"/","description":"Grafana"},
    {"id":"pyroscope","name":"Pyroscope","port":24040,"path":"/","description":"Pyroscope"},
    {"id":"alloy","name":"Alloy","port":21235,"path":"/","description":"Alloy"},
    {"id":"cadvisor","name":"cAdvisor","port":28081,"path":"/","description":"cAdvisor"}
  ]
}`

func TestDefaultLoadConfigUsesInternalDefaults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, validServiceManifest)
	}))
	defer server.Close()
	runner := &loadRunner{
		httpClient: server.Client(),
		now: func() time.Time {
			return time.Date(2026, 10, 1, 2, 3, 4, 5, time.UTC)
		},
	}
	config, err := defaultLoadConfig(t.Context(), runner, server.URL, "auto")
	if err != nil {
		t.Fatal(err)
	}
	if config.RPSInput != "auto" {
		t.Fatalf("default rps mode = %s, want auto", config.RPSInput)
	}
	wantOutput := filepath.Join(defaultOutputRoot, "login", "20261001-020304.000000005Z")
	if config.OutputDir != wantOutput {
		t.Fatalf("output = %s, want %s", config.OutputDir, wantOutput)
	}
}
