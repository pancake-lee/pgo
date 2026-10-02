package common

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

func TestDiscoverPortalServices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/portal/services.json" {
			t.Errorf("manifest path = %s", request.URL.Path)
		}
		_, _ = io.WriteString(writer, validServiceManifest)
	}))
	defer server.Close()
	services, err := DiscoverPortalServices(t.Context(), server.Client(), server.URL+"/portal/")
	if err != nil {
		t.Fatal(err)
	}
	urlList := []string{
		services.APIURL,
		services.DiagnosticsURL,
		services.RabbitMQURL,
		services.SwaggerURL,
		services.PrometheusURL,
		services.GrafanaURL,
		services.PyroscopeURL,
		services.AlloyURL,
		services.CAdvisorURL,
	}
	for _, address := range urlList {
		if !strings.HasPrefix(address, "http://") {
			t.Errorf("service URL = %s", address)
		}
	}
	if !strings.HasSuffix(services.APIURL, ":20000/") {
		t.Errorf("API URL = %s", services.APIURL)
	}
	if !strings.HasSuffix(services.DiagnosticsURL, ":20002/debug/pprof/heap") {
		t.Errorf("diagnostics URL = %s", services.DiagnosticsURL)
	}
}

func TestDiscoverPortalServicesRejectsInvalidInput(t *testing.T) {
	testList := []struct {
		name       string
		manifest   string
		statusCode int
		want       string
	}{
		{name: "missing", manifest: strings.Replace(validServiceManifest, `{"id":"api","name":"API","port":20000,"path":"/","description":"API"},`, "", 1), want: `required service "api" is missing`},
		{name: "duplicate", manifest: strings.Replace(validServiceManifest, `"id":"diagnostics"`, `"id":"api"`, 1), want: `duplicate service id "api"`},
		{name: "port", manifest: strings.Replace(validServiceManifest, `"port":20000`, `"port":0`, 1), want: "invalid port"},
		{name: "path", manifest: strings.Replace(validServiceManifest, `"path":"/debug/pprof/"`, `"path":"debug/pprof/"`, 1), want: "path must start with /"},
		{name: "web path", manifest: strings.Replace(validServiceManifest, `"webPath":"/debug/pprof/heap"`, `"webPath":"heap"`, 1), want: "webPath must start with /"},
		{name: "unknown field", manifest: strings.Replace(validServiceManifest, `"description":"API"`, `"description":"API","unknown":true`, 1), want: "unknown field"},
		{name: "trailing JSON", manifest: validServiceManifest + ` {}`, want: "trailing JSON content"},
		{name: "status", statusCode: http.StatusBadGateway, manifest: "upstream failed", want: "502 Bad Gateway"},
	}
	for _, test := range testList {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				if test.statusCode != 0 {
					writer.WriteHeader(test.statusCode)
				}
				_, _ = io.WriteString(writer, test.manifest)
			}))
			defer server.Close()
			_, err := DiscoverPortalServices(t.Context(), server.Client(), server.URL)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("discover error = %v", err)
			}
		})
	}
}

func TestPortalServiceURLSupportsHTTPSAndIPv6(t *testing.T) {
	portalURL, err := parsePortalURL("https://[2001:db8::1]:20080/portal/")
	if err != nil {
		t.Fatal(err)
	}
	actual := buildServiceURL(portalURL, serviceDefinition{Port: 23000, Path: "/"})
	if actual != "https://[2001:db8::1]:23000/" {
		t.Fatalf("service URL = %s", actual)
	}
}

func TestParsePortalURL(t *testing.T) {
	for _, address := range []string{"", "localhost:20080", "ftp://localhost", "http://user:pass@localhost"} {
		if _, err := parsePortalURL(address); err == nil {
			t.Errorf("expected portal URL error for %q", address)
		}
	}
}
