package papp

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
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
			server.handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/debug/pprof/"+profileType+"?seconds=1", nil))
			if response.Code != http.StatusOK {
				t.Fatalf("standard route status = %d", response.Code)
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

// TestSamplingSession 验证开关、所有权、手动采集冲突和关闭后的抓取。
func TestSamplingSession(t *testing.T) {
	controller, err := newRuntimeProfileController(diagnosticsConfig{
		BlockProfileRate: 1, MutexProfileFraction: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(controller.stop)
	mux := http.NewServeMux()
	controller.registerRoutes(mux)
	call := func(method, path string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(method, path, nil))
		return response
	}
	for _, path := range []string{
		"/debug/pprof/block?seconds=1", "/debug/pprof/mutex?seconds=1",
		"/debug/pprof/goroutine",
	} {
		response := call(http.MethodGet, path)
		if response.Code != http.StatusOK {
			t.Fatalf("idle %s: %d", path, response.Code)
		}
		assertReadableProfile(t, response.Body.Bytes())
	}
	response := call(http.MethodPost, "/debug/pprof/sampling?seconds=30")
	var session struct {
		ID uint64 `json:"id"`
	}
	err = json.Unmarshal(response.Body.Bytes(), &session)
	if response.Code != http.StatusOK || err != nil || session.ID == 0 {
		t.Fatalf("start sampling: %d %s", response.Code, response.Body.String())
	}
	for _, path := range []string{
		"/debug/pprof/sampling?seconds=30",
		"/debug/pprof/runtime?seconds=1&profile=mutex",
	} {
		if response = call(http.MethodPost, path); response.Code != http.StatusConflict {
			t.Fatalf("conflict: %d %s", response.Code, response.Body.String())
		}
	}
	// 采集区间内制造一次可识别的 channel 等待。
	done := make(chan struct{})
	go func() {
		defer close(done)
		time.Sleep(100 * time.Millisecond)
		blockForProfile()
	}()
	response = call(http.MethodGet, "/debug/pprof/block?seconds=1")
	<-done
	if response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
	output := assertReadableProfile(t, response.Body.Bytes())
	if !strings.Contains(output, "blockForProfile") {
		t.Fatalf("blocking sample missing: %s", output)
	}
	path := "/debug/pprof/sampling?id=" + strconv.FormatUint(session.ID, 10)
	if response = call(http.MethodDelete, path); response.Code != http.StatusNoContent {
		t.Fatalf("stop sampling: %d", response.Code)
	}
	if runtime.SetMutexProfileFraction(-1) != 0 {
		t.Fatal("mutex sampling still active")
	}
	response = call(http.MethodGet, "/debug/pprof/block?seconds=1")
	output = assertReadableProfile(t, response.Body.Bytes())
	if strings.Contains(output, "blockForProfile") {
		t.Fatal("historical blocking sample was replayed")
	}
	response = call(http.MethodPost, "/debug/pprof/sampling?seconds=30")
	if response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
	if response = call(http.MethodDelete, path); response.Code != http.StatusConflict {
		t.Fatalf("old session closed new sampling: %d", response.Code)
	}
}

// blockForProfile 产生一次可在 profile 中识别的等待。
func blockForProfile() {
	ready := make(chan struct{})
	time.AfterFunc(30*time.Millisecond, func() { close(ready) })
	<-ready
}

// assertReadableProfile 使用既有 Go CLI 验证 profile 格式并返回调用栈。
func assertReadableProfile(t *testing.T, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "profile.pprof")
	err := os.WriteFile(path, content, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "go", "tool", "pprof", "-top", "-nodecount=100", path).CombinedOutput()
	if err != nil {
		t.Fatalf("pprof parse: %v %s", err, output)
	}
	return string(output)
}

// TestSamplingExpiryAndLimits 验证会话自动到期、停止和错误输入。
func TestSamplingExpiryAndLimits(t *testing.T) {
	controller, err := newRuntimeProfileController(diagnosticsConfig{
		BlockProfileRate: 1, MutexProfileFraction: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(controller.stop)
	for _, path := range []string{
		"/debug/pprof/sampling", "/debug/pprof/sampling?seconds=0",
		"/debug/pprof/sampling?seconds=-1",
	} {
		response := httptest.NewRecorder()
		controller.samplingHandler(response, httptest.NewRequest(http.MethodPost, path, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("invalid duration: %d", response.Code)
		}
	}
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		response := httptest.NewRecorder()
		controller.samplingHandler(response, httptest.NewRequest(method,
			"/debug/pprof/sampling?seconds=1", nil))
		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("method %s: %d", method, response.Code)
		}
	}
	missingRates, err := newRuntimeProfileController(diagnosticsConfig{})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	missingRates.samplingHandler(response, httptest.NewRequest(http.MethodPost,
		"/debug/pprof/sampling?seconds=1", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing rates: %d", response.Code)
	}
	response = httptest.NewRecorder()
	controller.samplingHandler(response, httptest.NewRequest(http.MethodPost,
		"/debug/pprof/sampling?seconds=1", nil))
	if response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		controller.mu.Lock()
		active := controller.samplingID != 0
		controller.mu.Unlock()
		if !active {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("sampling did not expire")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if runtime.SetMutexProfileFraction(-1) != 0 {
		t.Fatal("expired sampling still active")
	}
	response = httptest.NewRecorder()
	controller.samplingHandler(response, httptest.NewRequest(http.MethodPost,
		"/debug/pprof/sampling?seconds=9223372036854775807", nil))
	var session struct {
		Seconds uint64 `json:"seconds"`
	}
	err = json.Unmarshal(response.Body.Bytes(), &session)
	if response.Code != http.StatusOK || err != nil || session.Seconds != 86400 {
		t.Fatalf("sampling was not capped: %d %s", response.Code, response.Body.String())
	}
	controller.stop()
	response = httptest.NewRecorder()
	controller.samplingHandler(response, httptest.NewRequest(http.MethodPost,
		"/debug/pprof/sampling?seconds=30", nil))
	controller.stop()
	if controller.samplingID != 0 || runtime.SetMutexProfileFraction(-1) != 0 {
		t.Fatal("shutdown did not stop sampling")
	}
}
