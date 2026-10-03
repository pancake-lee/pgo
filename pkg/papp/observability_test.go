package papp

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
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
			server, err := newDiagnosticsServer(diagnosticsConfig{Addr: "127.0.0.1:0", Pprof: testCase.enablePprof})
			if err != nil {
				t.Fatal(err)
			}
			pathList := []string{"/debug/pprof/heap"}
			if testCase.enablePprof {
				pathList = append(pathList, "/debug/pprof/profile?seconds=1")
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
	server, err := newDiagnosticsServer(diagnosticsConfig{Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	server.handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	if body := response.Body.String(); body == "" || !containsMetric(body, "pgo_http_requests_total") {
		t.Fatal("metrics response does not expose application request metrics")
	}
}

// TestDiagnosticsRuntimeProfile 验证单次 POST 返回二进制增量 profile 并清理采样状态。
func TestDiagnosticsRuntimeProfile(t *testing.T) {
	server, err := newDiagnosticsServer(diagnosticsConfig{
		Addr: "127.0.0.1:0", Pprof: true,
		BlockProfileRate: 1, MutexProfileFraction: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.profiles.stop()
	for _, profileType := range []string{"goroutine", "block", "mutex"} {
		t.Run(profileType, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost,
				"/debug/pprof/runtime?seconds=1&profile="+profileType, nil)
			started := time.Now()
			server.handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK || time.Since(started) < time.Second {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			reader, err := gzip.NewReader(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			data, err := io.ReadAll(reader)
			if err != nil || len(data) == 0 {
				t.Fatalf("profile data length = %d, error = %v", len(data), err)
			}
			if server.profiles.activeProfile != "" || server.profiles.cancelProfile != nil {
				t.Fatal("runtime profile state was not cleared")
			}
			response = httptest.NewRecorder()
			server.handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/debug/pprof/"+profileType, nil))
			if response.Code != http.StatusNotFound {
				t.Fatalf("removed route status = %d", response.Code)
			}
		})
	}
}

// TestDiagnosticsRuntimeProfileCancellation 验证并发请求被拒绝且取消或停止时关闭采样。
func TestDiagnosticsRuntimeProfileCancellation(t *testing.T) {
	for _, action := range []string{"cancel", "stop"} {
		t.Run(action, func(t *testing.T) {
			controller, err := newRuntimeProfileController(diagnosticsConfig{MutexProfileFraction: 1})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			request := httptest.NewRequest(http.MethodPost, "/debug/pprof/runtime?seconds=60&profile=mutex", nil).WithContext(ctx)
			done := make(chan struct{})
			go func() {
				defer close(done)
				controller.runtimeProfileHandler(httptest.NewRecorder(), request)
			}()
			deadline := time.Now().Add(time.Second)
			for {
				controller.mu.Lock()
				active := controller.activeProfile != ""
				controller.mu.Unlock()
				if active {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("profile did not start")
				}
				time.Sleep(time.Millisecond)
			}
			response := httptest.NewRecorder()
			controller.runtimeProfileHandler(response, httptest.NewRequest(http.MethodPost, "/debug/pprof/runtime?seconds=1&profile=mutex", nil))
			if response.Code != http.StatusConflict {
				t.Errorf("overlapping status = %d", response.Code)
			}
			if action == "stop" {
				controller.stop()
			} else {
				cancel()
			}
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("profile did not stop")
			}
			if controller.activeProfile != "" || controller.cancelProfile != nil || runtime.SetMutexProfileFraction(-1) != 0 {
				t.Fatal("sampling state was not reset")
			}
		})
	}
}

// TestDiagnosticsRuntimeLimits 验证采集参数、请求方法及未配置的采样参数限制。
func TestDiagnosticsRuntimeLimits(t *testing.T) {
	server, err := newDiagnosticsServer(diagnosticsConfig{Addr: "127.0.0.1:0", Pprof: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range []struct {
		method string
		path   string
		status int
	}{
		{method: http.MethodPost, path: "/debug/pprof/runtime?seconds=61&profile=goroutine", status: http.StatusBadRequest},
		{method: http.MethodPost, path: "/debug/pprof/runtime?seconds=1&profile=unknown", status: http.StatusBadRequest},
		{method: http.MethodGet, path: "/debug/pprof/runtime-trace?seconds=11", status: http.StatusBadRequest},
		{method: http.MethodGet, path: "/debug/pprof/runtime?seconds=1&profile=goroutine", status: http.StatusMethodNotAllowed},
		{method: http.MethodPost, path: "/debug/pprof/runtime?seconds=1&profiles=goroutine", status: http.StatusBadRequest},
		{method: http.MethodPost, path: "/debug/pprof/runtime?seconds=1&profile=block", status: http.StatusServiceUnavailable},
		{method: http.MethodPost, path: "/debug/pprof/runtime?seconds=1&profile=mutex", status: http.StatusServiceUnavailable},
		{method: http.MethodPost, path: "/debug/pprof/runtime?seconds=9223372036854775807&profile=goroutine", status: http.StatusBadRequest},
	} {
		response := httptest.NewRecorder()
		server.handler.ServeHTTP(response, httptest.NewRequest(testCase.method, testCase.path, nil))
		if response.Code != testCase.status {
			t.Fatalf("%s status = %d, want %d", testCase.path, response.Code, testCase.status)
		}
	}
}

func TestDiagnosticsRejectsNegativeProfileRate(t *testing.T) {
	if _, err := newDiagnosticsServer(diagnosticsConfig{Addr: "127.0.0.1:0", MemProfileRate: -1}); err == nil {
		t.Fatal("expected negative profile rate error")
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
