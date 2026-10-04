package core

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	klog "github.com/go-kratos/kratos/v2/log"
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

func TestBuildLoadConfigUsesSharedParameters(t *testing.T) {
	handler := http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		_, _ = io.WriteString(writer, validServiceManifest)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	now := time.Date(2026, 10, 1, 2, 3, 4, 5, time.UTC)
	config, err := buildLoadConfig(
		t.Context(),
		server.URL,
		"sample",
		75,
		45*time.Second,
		"",
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	wantOutput := filepath.Join(
		defaultOutputRoot,
		"sample",
		"20261001-020304.000000005Z",
	)
	if config.OutputDir != wantOutput || config.RPS != 75 ||
		config.Duration != 45*time.Second {
		t.Fatalf("config = %+v, want output %s", config, wantOutput)
	}
}

func TestResolveRPSList(t *testing.T) {
	automaticList := []int{200, 400, 600, 800, 1000}
	rpsList, automatic, err := resolveRPSList("auto")
	if err != nil || !automatic ||
		fmt.Sprint(rpsList) != fmt.Sprint(automaticList) {
		t.Fatalf("auto RPS = %v, %v, %v", rpsList, automatic, err)
	}
	rpsList, automatic, err = resolveRPSList("75")
	if err != nil || automatic || len(rpsList) != 1 || rpsList[0] != 75 {
		t.Fatalf("single RPS = %v, %v, %v", rpsList, automatic, err)
	}
	_, _, err = resolveRPSList("invalid")
	if err == nil {
		t.Fatal("expected invalid RPS error")
	}
	rpsList, _, err = resolveRPSList("auto")
	if err != nil {
		t.Fatal(err)
	}
	rpsList[0] = 999
	if autoRPSList[0] != automaticList[0] {
		t.Fatal("resolved RPS list changed the shared ladder")
	}
}

func TestParseDuration(t *testing.T) {
	duration, err := parseDuration("2m30s")
	if err != nil || duration != 150*time.Second {
		t.Fatalf("duration = %s, %v", duration, err)
	}
	for _, input := range []string{"", "invalid", "0s", "-1s"} {
		_, err = parseDuration(input)
		if err == nil {
			t.Errorf("expected duration error for %q", input)
		}
	}
}

func TestSharedCommandParameters(t *testing.T) {
	entrypoint := NewEntrypoint(Scenario{
		Name:  "sample",
		Short: "Run sample load",
		NewPreparer: func(Config, klog.Logger) Preparer {
			return &fakePreparer{}
		},
	})
	command := entrypoint.NewCobraCommand()
	if command.Use != "sample <portal-url>" {
		t.Fatalf("command use = %q", command.Use)
	}
	for _, name := range []string{"rps", "duration"} {
		if command.Flags().Lookup(name) == nil {
			t.Errorf("missing flag %q", name)
		}
	}
	if command.Flags().Lookup("duration").DefValue != "60s" {
		t.Fatalf(
			"duration default = %q",
			command.Flags().Lookup("duration").DefValue,
		)
	}
	now := time.Date(2026, 10, 1, 2, 3, 4, 5, time.UTC)
	outputDir := getDefaultOutputDir("sample", now)
	paramList := getParamList(outputDir)
	if len(paramList) != 4 || paramList[0].Name != "portal-url" ||
		paramList[1].Name != "rps" || paramList[2].Name != "duration" ||
		paramList[3].Name != "output-dir" ||
		paramList[3].Usage != "output directory" ||
		paramList[3].Default != outputDir {
		t.Fatalf("interactive parameters = %+v", paramList)
	}
}

func TestBuildLoadConfigUsesExactOutputDirectory(t *testing.T) {
	handler := http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		_, _ = io.WriteString(writer, validServiceManifest)
	})
	server := httptest.NewServer(handler)
	defer server.Close()
	now := time.Date(2026, 10, 1, 2, 3, 4, 5, time.UTC)
	defaultDir := filepath.Join(
		defaultOutputRoot, "sample", "20261001-020304.000000005Z",
	)
	absoluteDir := t.TempDir()
	testList := []struct {
		name  string
		input string
		want  string
	}{
		{"relative", "reports/login", "reports/login"},
		{"absolute", absoluteDir, absoluteDir},
		{"trimmed", "  reports/login  ", "reports/login"},
		{"empty", "", defaultDir},
		{"whitespace", "   ", defaultDir},
	}
	for _, test := range testList {
		t.Run(test.name, func(t *testing.T) {
			config, err := buildLoadConfig(
				t.Context(),
				server.URL,
				"sample",
				75,
				45*time.Second,
				test.input,
				now,
			)
			if err != nil {
				t.Fatal(err)
			}
			if config.OutputDir != test.want {
				t.Fatalf("output = %q, want %q", config.OutputDir, test.want)
			}
		})
	}
}
