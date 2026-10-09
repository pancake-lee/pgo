package core

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
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
	if !strings.Contains(config.DiagnosticsURL, ":20002/debug/pprof/heap") {
		t.Fatalf("missing discovered diagnostics URL: %s", config.DiagnosticsURL)
	}
	if config.OutputDir != wantOutput || config.RPS != 75 ||
		config.Duration != 45*time.Second {
		t.Fatalf("config = %+v, want output %s", config, wantOutput)
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
	for _, name := range []string{"rps", "auto", "duration", "sampling"} {
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
	if command.Flags().Lookup("sampling").DefValue != "false" {
		t.Fatal("sampling defaults to enabled")
	}
	err := command.ParseFlags([]string{"--sampling"})
	if err != nil {
		t.Fatal(err)
	}
	enabled, err := command.Flags().GetBool("sampling")
	if err != nil || !enabled {
		t.Fatal("sampling flag did not enable")
	}
	err = command.ParseFlags([]string{"--sampling=false"})
	if err != nil {
		t.Fatal(err)
	}
	enabled, err = command.Flags().GetBool("sampling")
	if err != nil || enabled {
		t.Fatal("sampling flag did not disable")
	}
	now := time.Date(2026, 10, 1, 2, 3, 4, 5, time.UTC)
	outputDir := getDefaultOutputDir("sample", now)
	paramList := getParamList(outputDir)
	wantNames := []string{"portal-url", "rps", "auto", "sampling", "duration", "output-dir"}
	if len(paramList) != len(wantNames) {
		t.Fatalf("interactive parameters = %+v", paramList)
	}
	for index, name := range wantNames {
		if paramList[index].Name != name {
			t.Fatalf("parameter %d = %+v", index, paramList[index])
		}
	}
	if paramList[2].Default != "false" || paramList[5].Default != outputDir ||
		command.Flag("auto").DefValue != "false" || command.Flag("rps").DefValue != "20" {
		t.Fatalf("unexpected defaults: %+v", paramList)
	}
	err = command.ParseFlags([]string{"--rps", "200", "--auto=true"})
	if err != nil {
		t.Fatal(err)
	}
	automatic, err := command.Flags().GetBool("auto")
	if err != nil || !automatic {
		t.Fatal("auto flag did not enable independently")
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

func TestParseRPS(t *testing.T) {
	for _, input := range []string{"200", " 205 "} {
		value, err := parseRPS(input)
		if err != nil || value <= 0 {
			t.Fatalf("input=%q value=%d error=%v", input, value, err)
		}
	}
	for _, input := range []string{"auto", "", "0", "-10", "1.5", "true"} {
		_, err := parseRPS(input)
		if err == nil {
			t.Fatalf("accepted invalid RPS %q", input)
		}
	}
}
