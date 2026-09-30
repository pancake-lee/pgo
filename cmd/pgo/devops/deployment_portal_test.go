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

	for _, expected := range []string{
		"portal:",
		"- \"20080:80\"",
		"./portal:/usr/share/nginx/html:ro",
		"window.location.hostname",
	} {
		if !strings.Contains(compose+portal, expected) {
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
		"cAdvisor":   "28081",
	}
	for name, port := range componentMap {
		if !strings.Contains(portal, `name: "`+name+`"`) {
			t.Errorf("portal missing component %q", name)
		}
		if !strings.Contains(portal, "port: "+port) {
			t.Errorf("portal missing port %s for %s", port, name)
		}
		if port != "20000" && port != "20002" && !strings.Contains(compose, port+":") {
			t.Errorf("compose missing published port %s for %s", port, name)
		}
	}
	if !strings.Contains(compose, "20000-20010:20000-20010") {
		t.Error("compose missing published backend port range")
	}
	if !strings.Contains(portal, `path: "/debug/pprof/"`) {
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

func readDeploymentTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
