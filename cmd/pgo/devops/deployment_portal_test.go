package devops

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeploymentPortalConfiguration(t *testing.T) {
	projectRoot := filepath.Join("..", "..", "..")
	compose := readDeploymentTestFile(t,
		filepath.Join(projectRoot, "deploy", "docker", "docker-compose.yaml"))
	portal := readDeploymentTestFile(t,
		filepath.Join(projectRoot, "deploy", "docker", "portal", "index.html"))
	serviceManifest := readDeploymentTestFile(t,
		filepath.Join(projectRoot, "deploy", "docker", "portal", "services.json"))
	var manifest struct {
		Services []struct {
			ID   string `json:"id"`
			Port int    `json:"port"`
		} `json:"services"`
	}
	if err := json.Unmarshal([]byte(serviceManifest), &manifest); err != nil {
		t.Fatalf("parse portal services: %v", err)
	}
	if len(manifest.Services) != 9 {
		t.Fatalf("portal services = %d, want 9", len(manifest.Services))
	}

	for _, expected := range []string{
		"portal:",
		"- \"20080:80\"",
		"./portal:/usr/share/nginx/html:ro",
		"window.location.hostname",
		`fetch("./services.json"`,
	} {
		if !strings.Contains(compose+portal+serviceManifest, expected) {
			t.Errorf("deployment portal configuration missing %q", expected)
		}
	}

	componentMap := map[string]string{
		"后端 API":     "20000",
		"后端 pprof":   "20002",
		"RabbitMQ":   "25674",
		"Swagger UI": "28080",
		"Prometheus": "29090",
		"Grafana":    "23000",
		"Pyroscope":  "24040",
		"Alloy":      "21235",
		"cAdvisor":   "28081",
	}
	for name, port := range componentMap {
		if !strings.Contains(serviceManifest, `"name": "`+name+`"`) {
			t.Errorf("portal missing component %q", name)
		}
		if !strings.Contains(serviceManifest, `"port": `+port) {
			t.Errorf("portal missing port %s for %s", port, name)
		}
		if port != "20000" && port != "20002" && !strings.Contains(compose, port+":") {
			t.Errorf("compose missing published port %s for %s", port, name)
		}
	}
	if !strings.Contains(compose, "20000-20010:20000-20010") {
		t.Error("compose missing published backend port range")
	}
	if !strings.Contains(serviceManifest, `"webPath": "/debug/pprof/heap"`) {
		t.Error("portal missing pprof path")
	}

	deployData := readDeploymentTestFile(t,
		filepath.Join(projectRoot, "deploy", "deploy.json"))
	var deployConfig DeployConfig
	if err := json.Unmarshal([]byte(deployData), &deployConfig); err != nil {
		t.Fatalf("parse deploy config: %v", err)
	}
	if got := deployConfig.Files["deploy/docker/portal/"]; got != "portal/" {
		t.Fatalf("portal deploy mapping = %q, want portal/", got)
	}
}

func TestDeploymentObservabilityConfiguration(t *testing.T) {
	projectRoot := filepath.Join("..", "..", "..")
	compose := readDeploymentTestFile(t, filepath.Join(projectRoot, "deploy", "docker", "docker-compose.yaml"))
	alloy := readDeploymentTestFile(t, filepath.Join(projectRoot, "deploy", "docker", "config", "config.alloy"))
	loki := readDeploymentTestFile(t, filepath.Join(projectRoot, "deploy", "docker", "config", "loki.yaml"))
	datasources := readDeploymentTestFile(
		t,
		filepath.Join(
			projectRoot,
			"deploy/docker/config/grafana/datasources/datasource.yml",
		),
	)
	dashboard := readDeploymentTestFile(
		t,
		filepath.Join(
			projectRoot,
			"deploy/docker/config/grafana/dashboards/pgo-app.json",
		),
	)
	prometheus := readDeploymentTestFile(
		t,
		filepath.Join(projectRoot, "deploy/docker/config/prometheus.yml"),
	)

	for _, expected := range []string{
		"grafana/alloy:v1.20.1",
		"grafana/pyroscope:2.2.0",
		"user: \"0:0\" # bind mount 的数据目录由 Loki 创建子目录",
		"user: \"0:0\" # bind mount 的数据目录由 Pyroscope 创建子目录",
		"pyroscope.scrape \"pgo_app\"",
		"profile.goroutine",
		"delta   = true",
		"scrape_timeout  = \"16s\"",
		"/debug/pprof/heap",
		"loki.source.file",
		"http://pgo-loki:3100/loki/api/v1/push",
		"type: grafana-pyroscope-datasource",
		"/data/loki",
	} {
		if !strings.Contains(compose+alloy+loki+datasources, expected) {
			t.Errorf("observability configuration missing %q", expected)
		}
	}
	if strings.Contains(compose, "promtail:") {
		t.Error("compose still contains Promtail")
	}
	if !strings.Contains(datasources, "timeInterval: 15s") {
		t.Error("Grafana Prometheus interval is not 15s")
	}
	if !strings.Contains(prometheus, "scrape_interval: 15s") {
		t.Error("Prometheus scrape interval is not 15s")
	}
	var dashboardConfig struct {
		Panels []struct {
			Title         string
			MaxDataPoints int
			Targets       []struct {
				Interval string
				Expr     string
			}
		}
	}
	err := json.Unmarshal([]byte(dashboard), &dashboardConfig)
	if err != nil {
		t.Fatalf("parse application dashboard: %v", err)
	}
	requestSelector := `{job="pgo-app",instance=~"${instance:regex}",` +
		`operation=~"${operation:regex}"}`
	errorSelector := strings.TrimSuffix(requestSelector, "}") +
		`,result="error"}`
	requestRate := "sum(irate(pgo_http_requests_total" +
		requestSelector + "[1m]))"
	expectedQueryMap := map[string]string{
		"Request rate": requestRate,
		"Error rate": "sum(irate(pgo_http_requests_total" +
			errorSelector + "[1m])) / clamp_min(" + requestRate + ", 1)",
		"p95 latency": "histogram_quantile(0.95, " +
			"sum(irate(pgo_http_request_duration_seconds_bucket" +
			requestSelector + "[1m])) by (le))",
	}
	for _, panel := range dashboardConfig.Panels {
		expectedQuery, ok := expectedQueryMap[panel.Title]
		if !ok {
			continue
		}
		delete(expectedQueryMap, panel.Title)
		if panel.MaxDataPoints != 2000 || len(panel.Targets) != 1 {
			t.Fatalf("%s must have 2000 points and one query", panel.Title)
		}
		if panel.Targets[0].Interval != "15s" {
			t.Errorf("%s query Min step must be 15s", panel.Title)
		}
		if panel.Targets[0].Expr != expectedQuery {
			t.Errorf("%s query = %q", panel.Title, panel.Targets[0].Expr)
		}
	}
	for title := range expectedQueryMap {
		t.Errorf("%s panel missing", title)
	}
}

func readDeploymentTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
