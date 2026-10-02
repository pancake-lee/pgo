package login

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/pancake-lee/pgo/cmd/pgo/performance"
	"github.com/spf13/pflag"
)

const validServiceManifest = `{
  "services": [
    {
      "id":"api","name":"API","port":20000,
      "path":"/","description":"API"
    },
    {
      "id":"diagnostics","name":"pprof","port":20002,
      "path":"/debug/pprof/","webPath":"/debug/pprof/heap",
      "description":"pprof"
    },
    {
      "id":"rabbitmq","name":"RabbitMQ","port":25674,
      "path":"/","description":"RabbitMQ"
    },
    {
      "id":"swagger","name":"Swagger","port":28080,
      "path":"/","description":"Swagger"
    },
    {
      "id":"prometheus","name":"Prometheus","port":29090,
      "path":"/","description":"Prometheus"
    },
    {
      "id":"grafana","name":"Grafana","port":23000,
      "path":"/","description":"Grafana"
    },
    {
      "id":"pyroscope","name":"Pyroscope","port":24040,
      "path":"/","description":"Pyroscope"
    },
    {
      "id":"alloy","name":"Alloy","port":21235,
      "path":"/","description":"Alloy"
    },
    {
      "id":"cadvisor","name":"cAdvisor","port":28081,
      "path":"/","description":"cAdvisor"
    }
  ]
}`

func TestDefaultLoadConfigUsesLoginOutputDirectory(t *testing.T) {
	handler := http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		_, _ = io.WriteString(writer, validServiceManifest)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	now := time.Date(2026, 10, 1, 2, 3, 4, 5, time.UTC)
	config, err := defaultLoadConfig(t.Context(), server.URL, now)
	if err != nil {
		t.Fatal(err)
	}
	wantOutput := filepath.Join(
		defaultOutputRoot,
		"login",
		"20261001-020304.000000005Z",
	)
	if config.OutputDir != wantOutput || config.RPS != 10 {
		t.Fatalf("config = %+v, want output %s and 10 RPS", config, wantOutput)
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
	_, _, err = resolveRPSList("invalid", 10)
	if err == nil {
		t.Fatal("expected invalid RPS error")
	}
}

func TestLoginCommandContract(t *testing.T) {
	command := newLoginCommand()
	if command.Use != "login <portal-url>" {
		t.Fatalf("login use = %q", command.Use)
	}
	if command.Flags().Lookup("rps") == nil {
		t.Fatalf("login flags = %v", command.Flags())
	}
	command.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag.Name != "rps" {
			t.Errorf("unexpected login flag %q", flag.Name)
		}
	})
	rootCommand := performance.NewCommand()
	commandList := rootCommand.Commands()
	invalidCommandList := len(commandList) != 1 ||
		commandList[0].Name() != "login"
	if invalidCommandList {
		t.Fatalf("registered performance commands = %v", rootCommand.Commands())
	}
}

func TestPreparerCreatesTargetsAndCleansUsers(t *testing.T) {
	fakeServer := newFakeUserServer()
	server := httptest.NewServer(fakeServer)
	defer server.Close()
	outputDir := t.TempDir()
	config := performance.Config{APIURL: server.URL, RPS: 10, OutputDir: outputDir}
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
