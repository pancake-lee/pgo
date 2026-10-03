package papp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	stdhttp "net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/pancake-lee/pgo/pkg/pdb"
	"github.com/pancake-lee/pgo/pkg/pmq"
	"github.com/pancake-lee/pgo/pkg/predis"
	"github.com/pancake-lee/pgo/pkg/putil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// requestIDHeader 指定请求标识的 HTTP 头名称。
const requestIDHeader = "X-Request-ID"

// requestTotal 按接口操作和处理结果累计 HTTP/gRPC 请求数。
var requestTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "pgo_http_requests_total",
	Help: "Total number of HTTP requests handled by pgo.",
}, []string{"operation", "result"})

// requestDuration 按接口操作和处理结果记录请求处理耗时分布。
var requestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "pgo_http_request_duration_seconds",
	Help:    "HTTP request latency handled by pgo.",
	Buckets: prometheus.DefBuckets,
}, []string{"operation", "result"})

// dependencyUp 记录已初始化依赖最近一次健康检查的可达状态。
var dependencyUp = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Name: "pgo_dependency_up",
	Help: "Whether an initialized dependency is reachable (1) or unavailable (0).",
}, []string{"dependency"})

// businessStatus 保存业务调用方主动设置的可用状态。
var businessStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Name: "pgo_business_status",
	Help: "Application-provided business status, where 1 means available.",
}, []string{"name"})

// init 将应用请求与状态指标注册到默认 Prometheus 注册表。
func init() {
	prometheus.MustRegister(requestTotal, requestDuration, dependencyUp, businessStatus)
}

// healthCheck 组合依赖检查的启用条件与检测函数。
type healthCheck struct {
	enabled func() bool
	check   func(context.Context) error
}

// healthCheckRegistry 并发保护内置及业务注册的依赖健康检查。
var healthCheckRegistry = struct {
	sync.RWMutex
	checkMap map[string]healthCheck
}{checkMap: map[string]healthCheck{
	"database": {enabled: pdb.IsInitialized, check: checkDatabase},
	"redis":    {enabled: predis.IsInitialized, check: checkRedis},
	"rabbitmq": {enabled: pmq.IsInitialized, check: checkRabbitMQ},
}}

// RegisterHealthCheck 注册自定义依赖检查并将名称用于健康指标标签。
func RegisterHealthCheck(name string, check func(context.Context) error) error {
	name = strings.TrimSpace(name)
	if name == "" || check == nil {
		return errors.New("health check name and function are required")
	}

	healthCheckRegistry.Lock()
	defer healthCheckRegistry.Unlock()
	if _, loaded := healthCheckRegistry.checkMap[name]; loaded {
		return fmt.Errorf("health check %q already registered", name)
	}
	healthCheckRegistry.checkMap[name] = healthCheck{enabled: func() bool { return true }, check: check}
	return nil
}

// SetBusinessStatus 发布指定业务的可用状态供仪表盘和告警读取。
func SetBusinessStatus(name string, available bool) {
	if strings.TrimSpace(name) == "" {
		return
	}
	value := 0.0
	if available {
		value = 1
	}
	businessStatus.WithLabelValues(name).Set(value)
}

// checkDatabase 在指定上下文内检查默认数据库连接是否可用。
func checkDatabase(ctx context.Context) error {
	return pdb.Ping(ctx)
}

// checkRedis 在指定上下文内检查默认 Redis 连接是否可用。
func checkRedis(ctx context.Context) error {
	return predis.Ping(ctx)
}

// checkRabbitMQ 检查默认 RabbitMQ 连接是否可用。
func checkRabbitMQ(context.Context) error {
	return pmq.Ping()
}

// requestObservabilityMiddleware 传递请求标识并记录正常返回调用的次数与处理耗时。
func requestObservabilityMiddleware() middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			traceID := requestTraceID(ctx)
			ctx = putil.SetTraceIdToCtx(ctx, traceID)
			startTime := time.Now()

			if tr, ok := transport.FromServerContext(ctx); ok {
				tr.ReplyHeader().Set(requestIDHeader, traceID)
			}

			reply, err = next(ctx, req)
			if _, ok := transport.FromServerContext(ctx); ok {
				operation := "unknown"
				if tr, found := transport.FromServerContext(ctx); found && tr.Operation() != "" {
					operation = tr.Operation()
				}
				result := "ok"
				if err != nil {
					result = "error"
				}
				requestTotal.WithLabelValues(operation, result).Inc()
				requestDuration.WithLabelValues(operation, result).Observe(time.Since(startTime).Seconds())
			}
			return reply, err
		}
	}
}

// requestTraceID 依次选用链路标识、合法请求头或新生成的请求标识。
func requestTraceID(ctx context.Context) string {
	if traceID := tracing.TraceID()(ctx); traceID != "" {
		return traceID.(string)
	}

	if tr, ok := transport.FromServerContext(ctx); ok {
		if requestID := tr.RequestHeader().Get(requestIDHeader); isValidRequestID(requestID) {
			return requestID
		}
	}
	return putil.UUID()
}

// isValidRequestID 检查请求标识是否满足长度与可见 ASCII 字符限制。
func isValidRequestID(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for _, char := range value {
		if char < 0x21 || char > 0x7e {
			return false
		}
	}
	return true
}

// diagnosticsServer 管理独立诊断 HTTP 服务及运行时采样控制器的生命周期。
type diagnosticsServer struct {
	address  string
	handler  stdhttp.Handler
	server   *stdhttp.Server
	listener net.Listener
	mu       sync.Mutex
	profiles *runtimeProfileController
}

// newDiagnosticsServer 创建指标与健康路由并按配置接入 pprof 诊断路由。
func newDiagnosticsServer(config diagnosticsConfig) (*diagnosticsServer, error) {
	controller, err := newRuntimeProfileController(config)
	if err != nil {
		return nil, err
	}
	mux := stdhttp.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/readyz", healthHandler)
	if config.Pprof {
		controller.registerRoutes(mux)
	}
	return &diagnosticsServer{address: config.Addr, handler: mux, profiles: controller}, nil
}

// Start 监听诊断地址并启动 HTTP 请求处理。
func (s *diagnosticsServer) Start(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.server != nil {
		return errors.New("diagnostics server already started")
	}
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}
	s.listener = listener
	s.server = &stdhttp.Server{Handler: s.handler, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := s.server.Serve(listener); err != nil && !errors.Is(err, stdhttp.ErrServerClosed) {
			panic(fmt.Errorf("diagnostics server: %w", err))
		}
	}()
	return nil
}

// Stop 关闭运行时采样窗口并在指定上下文内停止诊断服务。
func (s *diagnosticsServer) Stop(ctx context.Context) error {
	s.profiles.stop()
	s.mu.Lock()
	server := s.server
	s.mu.Unlock()
	if server == nil {
		return nil
	}
	return server.Shutdown(ctx)
}

// healthHandler 检查已启用依赖并更新健康指标及 HTTP 响应状态。
func healthHandler(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
	defer cancel()

	healthCheckRegistry.RLock()
	checkMap := make(map[string]healthCheck, len(healthCheckRegistry.checkMap))
	for name, check := range healthCheckRegistry.checkMap {
		checkMap[name] = check
	}
	healthCheckRegistry.RUnlock()

	resultMap := make(map[string]string, len(checkMap))
	healthy := true
	for name, healthCheck := range checkMap {
		if !healthCheck.enabled() {
			continue
		}
		err := healthCheck.check(ctx)
		if err != nil {
			resultMap[name] = err.Error()
			dependencyUp.WithLabelValues(name).Set(0)
			healthy = false
			continue
		}
		resultMap[name] = "ok"
		dependencyUp.WithLabelValues(name).Set(1)
	}

	status := stdhttp.StatusOK
	state := "ok"
	if !healthy {
		status = stdhttp.StatusServiceUnavailable
		state = "unhealthy"
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(map[string]any{"status": state, "dependencies": resultMap})
}

// tracingAndObservabilityMiddleware 组合链路追踪与请求指标中间件。
func tracingAndObservabilityMiddleware() middleware.Middleware {
	return middleware.Chain(tracing.Server(), requestObservabilityMiddleware())
}
