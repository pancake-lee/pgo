package common

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
)

const servicesFileName = "services.json"

var requiredServiceIDList = []string{
	"api",
	"diagnostics",
	"rabbitmq",
	"swagger",
	"prometheus",
	"grafana",
	"pyroscope",
	"alloy",
	"cadvisor",
}

// PortalServices 保存导航页中固定组件的访问地址。
type PortalServices struct {
	APIURL         string
	DiagnosticsURL string
	RabbitMQURL    string
	SwaggerURL     string
	PrometheusURL  string
	GrafanaURL     string
	PyroscopeURL   string
	AlloyURL       string
	CAdvisorURL    string
}

type serviceManifest struct {
	Services []serviceDefinition `json:"services"`
}

type serviceDefinition struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Port        int    `json:"port"`
	Path        string `json:"path"`
	WebPath     string `json:"webPath,omitempty"`
	Description string `json:"description"`
}

// DiscoverPortalServices 读取导航页清单并返回全部固定组件的访问地址。
func DiscoverPortalServices(ctx context.Context, client *http.Client, portalAddress string) (PortalServices, error) {
	portalURL, err := parsePortalURL(portalAddress)
	if err != nil {
		return PortalServices{}, err
	}
	manifestURL := *portalURL
	manifestURL.Path = path.Join(strings.TrimSuffix(portalURL.Path, "/"), servicesFileName)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL.String(), nil)
	if err != nil {
		return PortalServices{}, err
	}
	response, err := client.Do(request)
	if err != nil {
		return PortalServices{}, fmt.Errorf("read portal service manifest %s: %w", manifestURL.String(), err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return PortalServices{}, fmt.Errorf("read portal service manifest %s: %s: %s",
			manifestURL.String(), response.Status, strings.TrimSpace(string(body)))
	}
	var manifest serviceManifest
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1024*1024))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&manifest); err != nil {
		return PortalServices{}, fmt.Errorf("decode portal service manifest %s: %w", manifestURL.String(), err)
	}
	if err = decoder.Decode(&struct{}{}); err != io.EOF {
		return PortalServices{}, fmt.Errorf("decode portal service manifest %s: trailing JSON content", manifestURL.String())
	}
	serviceMap, err := validateServiceManifest(manifest)
	if err != nil {
		return PortalServices{}, fmt.Errorf("validate portal service manifest %s: %w", manifestURL.String(), err)
	}
	return PortalServices{
		APIURL:         buildServiceURL(portalURL, serviceMap["api"]),
		DiagnosticsURL: buildServiceURL(portalURL, serviceMap["diagnostics"]),
		RabbitMQURL:    buildServiceURL(portalURL, serviceMap["rabbitmq"]),
		SwaggerURL:     buildServiceURL(portalURL, serviceMap["swagger"]),
		PrometheusURL:  buildServiceURL(portalURL, serviceMap["prometheus"]),
		GrafanaURL:     buildServiceURL(portalURL, serviceMap["grafana"]),
		PyroscopeURL:   buildServiceURL(portalURL, serviceMap["pyroscope"]),
		AlloyURL:       buildServiceURL(portalURL, serviceMap["alloy"]),
		CAdvisorURL:    buildServiceURL(portalURL, serviceMap["cadvisor"]),
	}, nil
}

func parsePortalURL(address string) (*url.URL, error) {
	address = strings.TrimSpace(address)
	parsedURL, err := url.Parse(address)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Hostname() == "" {
		return nil, fmt.Errorf("portal URL must be an absolute HTTP URL, got %q", address)
	}
	if parsedURL.User != nil || parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return nil, errors.New("portal URL must not contain credentials, query, or fragment")
	}
	return parsedURL, nil
}

func validateServiceManifest(manifest serviceManifest) (map[string]serviceDefinition, error) {
	serviceMap := make(map[string]serviceDefinition, len(manifest.Services))
	for _, service := range manifest.Services {
		service.ID = strings.TrimSpace(service.ID)
		if service.ID == "" {
			return nil, errors.New("service id is required")
		}
		if _, exists := serviceMap[service.ID]; exists {
			return nil, fmt.Errorf("duplicate service id %q", service.ID)
		}
		if service.Port < 1 || service.Port > 65535 {
			return nil, fmt.Errorf("service %q has invalid port %d", service.ID, service.Port)
		}
		if !strings.HasPrefix(service.Path, "/") {
			return nil, fmt.Errorf("service %q path must start with /", service.ID)
		}
		if service.WebPath != "" && !strings.HasPrefix(service.WebPath, "/") {
			return nil, fmt.Errorf("service %q webPath must start with /", service.ID)
		}
		serviceMap[service.ID] = service
	}
	for _, id := range requiredServiceIDList {
		if _, exists := serviceMap[id]; !exists {
			return nil, fmt.Errorf("required service %q is missing", id)
		}
	}
	return serviceMap, nil
}

func buildServiceURL(portalURL *url.URL, service serviceDefinition) string {
	servicePath := service.Path
	if service.WebPath != "" {
		servicePath = service.WebPath
	}
	host := net.JoinHostPort(portalURL.Hostname(), strconv.Itoa(service.Port))
	return (&url.URL{Scheme: portalURL.Scheme, Host: host, Path: servicePath}).String()
}
