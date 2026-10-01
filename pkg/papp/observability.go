package papp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	stdhttp "net/http"
	"net/http/pprof"
	"runtime"
	"strconv"
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

const requestIDHeader = "X-Request-ID"

const (
	maxRuntimeProfileDuration = 60 * time.Second
	maxRuntimeTraceDuration   = 10 * time.Second
)

var requestTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "pgo_http_requests_total",
	Help: "Total number of HTTP requests handled by pgo.",
}, []string{"operation", "result"})

var requestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "pgo_http_request_duration_seconds",
	Help:    "HTTP request latency handled by pgo.",
	Buckets: prometheus.DefBuckets,
}, []string{"operation", "result"})

var dependencyUp = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Name: "pgo_dependency_up",
	Help: "Whether an initialized dependency is reachable (1) or unavailable (0).",
}, []string{"dependency"})

var businessStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Name: "pgo_business_status",
	Help: "Application-provided business status, where 1 means available.",
}, []string{"name"})

func init() {
	prometheus.MustRegister(requestTotal, requestDuration, dependencyUp, businessStatus)
}

type healthCheck struct {
	enabled func() bool
	check   func(context.Context) error
}

var healthCheckRegistry = struct {
	sync.RWMutex
	checkMap map[string]healthCheck
}{checkMap: map[string]healthCheck{
	"database": {enabled: pdb.IsInitialized, check: checkDatabase},
	"redis":    {enabled: predis.IsInitialized, check: checkRedis},
	"rabbitmq": {enabled: pmq.IsInitialized, check: checkRabbitMQ},
}}

// RegisterHealthCheck adds an application-specific dependency check. The name is
// also exposed as the dependency label in pgo_dependency_up.
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

// SetBusinessStatus publishes a coarse business readiness signal for dashboards
// and alerts without coupling papp to a specific service domain.
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

func checkDatabase(ctx context.Context) error {
	return pdb.Ping(ctx)
}

func checkRedis(ctx context.Context) error {
	return predis.Ping(ctx)
}

func checkRabbitMQ(context.Context) error {
	return pmq.Ping()
}

// requestObservabilityMiddleware makes a sanitized request ID available to all
// downstream code, returns it to HTTP callers, and records RED metrics.
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

type diagnosticsServer struct {
	address  string
	handler  stdhttp.Handler
	server   *stdhttp.Server
	listener net.Listener
	mu       sync.Mutex
	profiles *runtimeProfileController
}

type runtimeProfileController struct {
	mu                   sync.Mutex
	activeProfileMap     map[string]bool
	expiresAt            time.Time
	generation           uint64
	blockProfileRate     int
	mutexProfileFraction int
	cpuMu                sync.Mutex
	traceActive          bool
}

func newDiagnosticsServer(config diagnosticsConfig) (*diagnosticsServer, error) {
	if config.MemProfileRate < 0 || config.BlockProfileRate < 0 || config.MutexProfileFraction < 0 {
		return nil, errors.New("diagnostics profile rates must not be negative")
	}
	if config.MemProfileRate > 0 {
		runtime.MemProfileRate = config.MemProfileRate
	}
	controller := &runtimeProfileController{
		activeProfileMap:     make(map[string]bool),
		blockProfileRate:     config.BlockProfileRate,
		mutexProfileFraction: config.MutexProfileFraction,
	}
	mux := stdhttp.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/readyz", healthHandler)
	if config.Pprof {
		mux.HandleFunc("/debug/pprof/profile", controller.cpuProfileHandler)
		mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		mux.HandleFunc("/debug/pprof/runtime", controller.runtimeProfileHandler)
		mux.HandleFunc("/debug/pprof/runtime-trace", controller.runtimeTraceHandler)
		for _, profileType := range []string{"goroutine", "block", "mutex"} {
			mux.Handle("/debug/pprof/"+profileType, controller.gatedProfileHandler(profileType))
		}
	}
	return &diagnosticsServer{address: config.Addr, handler: mux, profiles: controller}, nil
}

func (controller *runtimeProfileController) cpuProfileHandler(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	controller.cpuMu.Lock()
	defer controller.cpuMu.Unlock()
	pprof.Profile(writer, request)
}

func (controller *runtimeProfileController) runtimeProfileHandler(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	if request.Method != stdhttp.MethodPost {
		writer.Header().Set("Allow", stdhttp.MethodPost)
		stdhttp.Error(writer, "method not allowed", stdhttp.StatusMethodNotAllowed)
		return
	}
	duration, err := parseLimitedDuration(request, maxRuntimeProfileDuration)
	if err != nil {
		stdhttp.Error(writer, err.Error(), stdhttp.StatusBadRequest)
		return
	}
	profileList, err := parseRuntimeProfileList(request.URL.Query().Get("profiles"))
	if err != nil {
		stdhttp.Error(writer, err.Error(), stdhttp.StatusBadRequest)
		return
	}

	controller.mu.Lock()
	defer controller.mu.Unlock()
	controller.expireLocked(time.Now())
	if len(controller.activeProfileMap) > 0 {
		stdhttp.Error(writer, "runtime profiles already active", stdhttp.StatusConflict)
		return
	}
	if containsString(profileList, "block") && controller.blockProfileRate == 0 {
		stdhttp.Error(writer, "Diagnostics.BlockProfileRate must be configured", stdhttp.StatusServiceUnavailable)
		return
	}
	if containsString(profileList, "mutex") && controller.mutexProfileFraction == 0 {
		stdhttp.Error(writer, "Diagnostics.MutexProfileFraction must be configured", stdhttp.StatusServiceUnavailable)
		return
	}
	for _, profileType := range profileList {
		controller.activeProfileMap[profileType] = true
	}
	if controller.activeProfileMap["block"] {
		runtime.SetBlockProfileRate(controller.blockProfileRate)
	}
	if controller.activeProfileMap["mutex"] {
		runtime.SetMutexProfileFraction(controller.mutexProfileFraction)
	}
	controller.expiresAt = time.Now().Add(duration)
	controller.generation++
	generation := controller.generation
	time.AfterFunc(duration, func() { controller.expire(generation) })
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(map[string]any{
		"profiles":  profileList,
		"expiresAt": controller.expiresAt.UTC(),
	})
}

func (controller *runtimeProfileController) gatedProfileHandler(profileType string) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
		controller.mu.Lock()
		controller.expireLocked(time.Now())
		active := controller.activeProfileMap[profileType]
		controller.mu.Unlock()
		if !active {
			stdhttp.Error(writer, "runtime profile is not active", stdhttp.StatusForbidden)
			return
		}
		pprof.Handler(profileType).ServeHTTP(writer, request)
	})
}

func (controller *runtimeProfileController) runtimeTraceHandler(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	if request.Method != stdhttp.MethodGet {
		writer.Header().Set("Allow", stdhttp.MethodGet)
		stdhttp.Error(writer, "method not allowed", stdhttp.StatusMethodNotAllowed)
		return
	}
	if _, err := parseLimitedDuration(request, maxRuntimeTraceDuration); err != nil {
		stdhttp.Error(writer, err.Error(), stdhttp.StatusBadRequest)
		return
	}
	controller.mu.Lock()
	if controller.traceActive {
		controller.mu.Unlock()
		stdhttp.Error(writer, "runtime trace already active", stdhttp.StatusConflict)
		return
	}
	controller.traceActive = true
	controller.mu.Unlock()
	defer func() {
		controller.mu.Lock()
		controller.traceActive = false
		controller.mu.Unlock()
	}()
	pprof.Trace(writer, request)
}

func parseLimitedDuration(request *stdhttp.Request, maximum time.Duration) (time.Duration, error) {
	seconds, err := strconv.Atoi(request.URL.Query().Get("seconds"))
	if err != nil || seconds <= 0 {
		return 0, errors.New("seconds must be a positive integer")
	}
	duration := time.Duration(seconds) * time.Second
	if duration > maximum {
		return 0, fmt.Errorf("seconds must not exceed %d", int(maximum/time.Second))
	}
	return duration, nil
}

func parseRuntimeProfileList(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, errors.New("profiles is required")
	}
	seenMap := make(map[string]bool)
	var profileList []string
	for _, profileType := range strings.Split(value, ",") {
		profileType = strings.TrimSpace(profileType)
		switch profileType {
		case "goroutine", "block", "mutex":
		default:
			return nil, fmt.Errorf("unsupported runtime profile %q", profileType)
		}
		if !seenMap[profileType] {
			seenMap[profileType] = true
			profileList = append(profileList, profileType)
		}
	}
	return profileList, nil
}

func containsString(valueList []string, target string) bool {
	for _, value := range valueList {
		if value == target {
			return true
		}
	}
	return false
}

func (controller *runtimeProfileController) expire(generation uint64) {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	if controller.generation == generation {
		controller.disableLocked()
	}
}

func (controller *runtimeProfileController) expireLocked(now time.Time) {
	if !controller.expiresAt.IsZero() && !now.Before(controller.expiresAt) {
		controller.disableLocked()
	}
}

func (controller *runtimeProfileController) disableLocked() {
	if controller.activeProfileMap["block"] {
		runtime.SetBlockProfileRate(0)
	}
	if controller.activeProfileMap["mutex"] {
		runtime.SetMutexProfileFraction(0)
	}
	clear(controller.activeProfileMap)
	controller.expiresAt = time.Time{}
	controller.generation++
}

func (controller *runtimeProfileController) stop() {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	controller.disableLocked()
}

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

func tracingAndObservabilityMiddleware() middleware.Middleware {
	return middleware.Chain(tracing.Server(), requestObservabilityMiddleware())
}
