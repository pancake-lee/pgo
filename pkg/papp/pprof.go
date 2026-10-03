package papp

import (
	"context"
	"errors"
	"fmt"
	stdhttp "net/http"
	"net/http/pprof"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const (
	// maxRuntimeProfileDuration 限制单次运行时 profile 的最长采集时间。
	maxRuntimeProfileDuration = 60 * time.Second
	// maxRuntimeTraceDuration 限制单次 runtime trace 的最长采集时间。
	maxRuntimeTraceDuration = 10 * time.Second
)

// runtimeProfileController 协调运行时采样参数及 profile 与 trace 采集状态。
type runtimeProfileController struct {
	mu                   sync.Mutex
	activeProfile        string
	cancelProfile        context.CancelFunc
	blockProfileRate     int
	mutexProfileFraction int
	traceActive          bool
}

// newRuntimeProfileController 校验采样参数并初始化运行时 profile 控制状态。
func newRuntimeProfileController(config diagnosticsConfig,
) (*runtimeProfileController, error) {
	if config.MemProfileRate < 0 || config.BlockProfileRate < 0 ||
		config.MutexProfileFraction < 0 {
		return nil, errors.New("diagnostics profile rates must not be negative")
	}
	if config.MemProfileRate > 0 {
		runtime.MemProfileRate = config.MemProfileRate
	}
	return &runtimeProfileController{
		blockProfileRate:     config.BlockProfileRate,
		mutexProfileFraction: config.MutexProfileFraction,
	}, nil
}

// 六、调用栈画像：CPU 与 heap 供 Alloy 持续采集，写入 Pyroscope。

// registerRoutes 将持续及按需采集的 pprof 接口注册到诊断路由。
func (controller *runtimeProfileController) registerRoutes(mux *stdhttp.ServeMux) {
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))

	// 八、手动触发：goroutine、block、mutex 和 trace 仅按需采集。
	mux.HandleFunc("/debug/pprof/runtime", controller.runtimeProfileHandler)
	mux.HandleFunc("/debug/pprof/runtime-trace", controller.runtimeTraceHandler)
}

// 八、手动触发：单次 POST 返回 runtime 增量，trace 单独返回时间线。

// runtimeProfileHandler 开启单个运行时采样并等待后直接返回增量 profile。
func (controller *runtimeProfileController) runtimeProfileHandler(
	writer stdhttp.ResponseWriter,
	request *stdhttp.Request,
) {
	if request.Method != stdhttp.MethodPost {
		writer.Header().Set("Allow", stdhttp.MethodPost)
		stdhttp.Error(writer, "method not allowed", stdhttp.StatusMethodNotAllowed)
		return
	}

	_, err := parseLimitedDuration(request, maxRuntimeProfileDuration)
	if err != nil {
		stdhttp.Error(writer, err.Error(), stdhttp.StatusBadRequest)
		return
	}
	profileType := request.URL.Query().Get("profile")
	switch profileType {
	case "goroutine", "block", "mutex":
	default:
		stdhttp.Error(writer, "profile must be goroutine, block or mutex",
			stdhttp.StatusBadRequest)
		return
	}

	controller.mu.Lock()
	if controller.activeProfile != "" {
		controller.mu.Unlock()
		stdhttp.Error(writer, "runtime profile already active",
			stdhttp.StatusConflict)
		return
	}
	if profileType == "block" && controller.blockProfileRate == 0 {
		controller.mu.Unlock()
		stdhttp.Error(writer, "Diagnostics.BlockProfileRate must be configured",
			stdhttp.StatusServiceUnavailable)
		return
	}
	if profileType == "mutex" && controller.mutexProfileFraction == 0 {
		controller.mu.Unlock()
		stdhttp.Error(writer, "Diagnostics.MutexProfileFraction must be configured",
			stdhttp.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithCancel(request.Context())
	controller.activeProfile = profileType
	controller.cancelProfile = cancel
	if profileType == "block" {
		runtime.SetBlockProfileRate(controller.blockProfileRate)
	}
	if profileType == "mutex" {
		runtime.SetMutexProfileFraction(controller.mutexProfileFraction)
	}
	controller.mu.Unlock()
	defer func() {
		controller.mu.Lock()
		if profileType == "block" {
			runtime.SetBlockProfileRate(0)
		}
		if profileType == "mutex" {
			runtime.SetMutexProfileFraction(0)
		}
		controller.activeProfile = ""
		controller.cancelProfile = nil
		controller.mu.Unlock()
		cancel()
	}()

	profileRequest := request.WithContext(ctx)
	profileRequest.Form = request.URL.Query()
	pprof.Handler(profileType).ServeHTTP(writer, profileRequest)
}

// runtimeTraceHandler 限制采集时长并保证同一时刻只有一个 runtime trace。
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

// parseLimitedDuration 解析秒数参数并检查采集时长是否在指定范围内。
func parseLimitedDuration(request *stdhttp.Request, maximum time.Duration) (time.Duration, error) {
	seconds, err := strconv.Atoi(request.URL.Query().Get("seconds"))
	if err != nil || seconds <= 0 {
		return 0, errors.New("seconds must be a positive integer")
	}
	if seconds > int(maximum/time.Second) {
		return 0, fmt.Errorf("seconds must not exceed %d", int(maximum/time.Second))
	}
	return time.Duration(seconds) * time.Second, nil
}

// stop 取消当前运行时采集以触发请求处理函数关闭采样。
func (controller *runtimeProfileController) stop() {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	if controller.cancelProfile != nil {
		controller.cancelProfile()
	}
}
