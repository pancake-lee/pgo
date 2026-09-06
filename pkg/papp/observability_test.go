package papp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/transport"
	"github.com/pancake-lee/pgo/pkg/putil"
)

type testHeader map[string]string

func (h testHeader) Get(key string) string      { return h[key] }
func (h testHeader) Set(key, value string)      { h[key] = value }
func (h testHeader) Add(key, value string)      { h[key] = value }
func (h testHeader) Keys() []string             { return nil }
func (h testHeader) Values(key string) []string { return []string{h[key]} }

type testTransport struct {
	requestHeader testHeader
	replyHeader   testHeader
}

func (t *testTransport) Kind() transport.Kind            { return transport.KindHTTP }
func (t *testTransport) Endpoint() string                { return "http://test" }
func (t *testTransport) Operation() string               { return "test.operation" }
func (t *testTransport) RequestHeader() transport.Header { return t.requestHeader }
func (t *testTransport) ReplyHeader() transport.Header   { return t.replyHeader }

func TestRequestObservabilityMiddleware(t *testing.T) {
	tr := &testTransport{requestHeader: testHeader{requestIDHeader: "upstream-request"}, replyHeader: testHeader{}}
	ctx := transport.NewServerContext(context.Background(), tr)
	_, err := requestObservabilityMiddleware()(func(ctx context.Context, req any) (any, error) {
		traceID, ok := putil.GetTraceIdFromCtx(ctx)
		if !ok || traceID != "upstream-request" {
			t.Fatalf("trace ID = %q, ok = %t", traceID, ok)
		}
		return nil, nil
	})(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := tr.replyHeader.Get(requestIDHeader); got != "upstream-request" {
		t.Fatalf("response request ID = %q", got)
	}
}

func TestRequestTraceIDRejectsInvalidHeader(t *testing.T) {
	tr := &testTransport{requestHeader: testHeader{requestIDHeader: "unsafe\nvalue"}, replyHeader: testHeader{}}
	ctx := transport.NewServerContext(context.Background(), tr)
	if got := requestTraceID(ctx); got == "unsafe\nvalue" || got == "" {
		t.Fatalf("invalid header trace ID = %q", got)
	}
}

func TestNewAppCtxRequestMetadata(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ctx = putil.SetTraceIdToCtx(ctx, "request-123")
	appCtx := NewAppCtx(ctx)
	if appCtx.TraceID != "request-123" {
		t.Fatalf("TraceID = %q", appCtx.TraceID)
	}
	if appCtx.StartTime.IsZero() || appCtx.Elapsed() < 0 {
		t.Fatalf("invalid start time: %v", appCtx.StartTime)
	}
	if _, ok := appCtx.Deadline(); !ok {
		t.Fatal("embedded context deadline was not preserved")
	}
}

func TestNewAppCtxConcurrentIsolation(t *testing.T) {
	const workers = 32
	traceIDChannel := make(chan string, workers)
	var waitGroup sync.WaitGroup
	for index := 0; index < workers; index++ {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			traceIDChannel <- NewAppCtx(context.Background()).TraceID
		}(index)
	}
	waitGroup.Wait()
	close(traceIDChannel)
	traceIDMap := make(map[string]bool, workers)
	for traceID := range traceIDChannel {
		if traceID == "" || traceIDMap[traceID] {
			t.Fatalf("duplicate or empty trace ID %q", traceID)
		}
		traceIDMap[traceID] = true
	}
}

func TestDiagnosticsPprofSwitch(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		enablePprof bool
		status      int
	}{
		{name: "disabled", status: http.StatusNotFound},
		{name: "enabled", enablePprof: true, status: http.StatusOK},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			server := newDiagnosticsServer("127.0.0.1:0", testCase.enablePprof)
			pathList := []string{"/debug/pprof/goroutine"}
			if testCase.enablePprof {
				pathList = append(pathList, "/debug/pprof/heap", "/debug/pprof/profile?seconds=1")
			}
			for _, path := range pathList {
				response := httptest.NewRecorder()
				server.handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
				if response.Code != testCase.status {
					t.Fatalf("path %s: status = %d, want %d", path, response.Code, testCase.status)
				}
			}
		})
	}
}

func TestDiagnosticsMetrics(t *testing.T) {
	server := newDiagnosticsServer("127.0.0.1:0", false)
	response := httptest.NewRecorder()
	server.handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	if body := response.Body.String(); body == "" || !containsMetric(body, "pgo_http_requests_total") {
		t.Fatal("metrics response does not expose application request metrics")
	}
}

func containsMetric(body, metric string) bool {
	for index := 0; index+len(metric) <= len(body); index++ {
		if body[index:index+len(metric)] == metric {
			return true
		}
	}
	return false
}
