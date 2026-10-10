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

// 一、服务健康：依赖状态与健康检查，对应文档的服务健康分组。

// dependencyUp 记录已初始化依赖最近一次健康检查的可达状态。
var dependencyUp = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Name: "pgo_dependency_up",
	Help: "Whether an initialized dependency is reachable (1) or unavailable (0).",
}, []string{"dependency"})

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

// checkDependencies 检查已启用依赖并写入共用健康指标。
func checkDependencies(ctx context.Context) (map[string]string, bool) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
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
	return resultMap, healthy
}

// healthHandler 执行实时依赖检查并更新健康指标及 HTTP 响应状态。
func healthHandler(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	resultMap, healthy := checkDependencies(request.Context())

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

// 二、请求 RED：请求量、错误率及耗时，对应文档的请求 RED 分组。

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

// tracingAndObservabilityMiddleware 组合链路追踪与请求指标中间件。
func tracingAndObservabilityMiddleware() middleware.Middleware {
	return middleware.Chain(tracing.Server(), requestObservabilityMiddleware())
}

// 三、依赖瓶颈：数据库操作、连接池与等待指标由 pkg/pdb/observability.go 提供。
// 此处导入 pdb 接入该包注册的指标，实际业务由 GORM 回调记录。

// 四、进程资源：process_* 由 client_golang 的默认 ProcessCollector 提供。
// 按 dashboard 顺序关注 CPU、RSS、文件描述符数量和使用比例。

// 五、Go runtime：go_* 由 client_golang 的默认 GoCollector 提供。
// 按 dashboard 顺序关注 heap、分配速率、对象数、goroutine、GC 暂停、频率与时间。
// 两类默认采集器随 prometheus 包初始化注册，由下面 /metrics 路由统一暴露。

// 共用接线：注册应用指标并管理诊断 HTTP 服务，画像路由见 pprof.go。

// init 按服务健康与请求 RED 顺序注册应用指标到默认 Prometheus 注册表。
func init() {
	prometheus.MustRegister(dependencyUp, requestTotal, requestDuration)
}

// diagnosticsServer 管理独立诊断 HTTP 服务及运行时采样控制器的生命周期。
type diagnosticsServer struct {
	address      string
	handler      stdhttp.Handler
	server       *stdhttp.Server
	listener     net.Listener
	mu           sync.Mutex
	profiles     *runtimeProfileController
	healthCancel context.CancelFunc
	healthDone   chan struct{}
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
	healthCtx, cancel := context.WithCancel(context.Background())
	s.healthCancel = cancel
	s.healthDone = make(chan struct{})
	go s.refreshDependencies(healthCtx, 15*time.Second)
	go func() {
		if err := s.server.Serve(listener); err != nil && !errors.Is(err, stdhttp.ErrServerClosed) {
			panic(fmt.Errorf("diagnostics server: %w", err))
		}
	}()
	return nil
}

// refreshDependencies 在独立任务中立即检查并定时刷新健康指标。
func (s *diagnosticsServer) refreshDependencies(
	ctx context.Context, interval time.Duration,
) {
	defer close(s.healthDone)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		checkDependencies(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Stop 取消定时检查和运行时采集并在指定上下文内停止诊断服务。
func (s *diagnosticsServer) Stop(ctx context.Context) error {
	s.profiles.stop()
	s.mu.Lock()
	server := s.server
	cancel := s.healthCancel
	done := s.healthDone
	s.mu.Unlock()
	if server == nil {
		return nil
	}
	cancel()
	err := server.Shutdown(ctx)
	select {
	case <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
