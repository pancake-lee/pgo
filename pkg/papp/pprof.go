package papp

import (
	"context"
	"encoding/json"
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
	// maxSamplingDuration 限制客户端丢失后采样会话的最长存活时间。
	maxSamplingDuration = 24 * time.Hour
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
	samplingID           uint64
	samplingGeneration   uint64
	samplingTimer        *time.Timer
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
		samplingGeneration:   uint64(time.Now().UnixNano()),
		blockProfileRate:     config.BlockProfileRate,
		mutexProfileFraction: config.MutexProfileFraction,
	}, nil
}

// 六、调用栈画像：标准 pprof 抓取与测试期间采样控制分开。

// registerRoutes 将持续及按需采集的 pprof 接口注册到诊断路由。
func (controller *runtimeProfileController) registerRoutes(mux *stdhttp.ServeMux) {
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))

	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.HandleFunc("/debug/pprof/block", deltaProfileHandler("block"))
	mux.HandleFunc("/debug/pprof/mutex", deltaProfileHandler("mutex"))
	mux.HandleFunc("/debug/pprof/sampling", controller.samplingHandler)

	// 八、手动触发：保留限时采集，与测试采样会话互斥。
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
	if controller.activeProfile != "" || controller.samplingID != 0 {
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
	controller.closeSamplingLocked(controller.samplingID)
}

// deltaProfileHandler 使用标准 pprof 区间差值，采样关闭时也正常返回。
func deltaProfileHandler(profileType string) stdhttp.HandlerFunc {
	return func(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
		if request.Method != stdhttp.MethodGet {
			writer.Header().Set("Allow", stdhttp.MethodGet)
			stdhttp.Error(writer, "method not allowed", stdhttp.StatusMethodNotAllowed)
			return
		}
		if request.URL.Query().Get("seconds") == "" {
			query := request.URL.Query()
			query.Set("seconds", "14")
			request.URL.RawQuery = query.Encode()
		}
		_, err := parseLimitedDuration(request, maxRuntimeProfileDuration)
		if err != nil {
			stdhttp.Error(writer, err.Error(), stdhttp.StatusBadRequest)
			return
		}
		pprof.Handler(profileType).ServeHTTP(writer, request)
	}
}

// samplingHandler 开启限时采样会话，或仅关闭调用者持有的会话。
func (controller *runtimeProfileController) samplingHandler(
	writer stdhttp.ResponseWriter, request *stdhttp.Request,
) {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	switch request.Method {
	case stdhttp.MethodPost:
		duration, err := parseLimitedDuration(request, maxSamplingDuration)
		if err != nil {
			stdhttp.Error(writer, err.Error(), stdhttp.StatusBadRequest)
			return
		}
		if controller.samplingID != 0 || controller.activeProfile != "" {
			stdhttp.Error(writer, "sampling already active", stdhttp.StatusConflict)
			return
		}
		if controller.blockProfileRate == 0 || controller.mutexProfileFraction == 0 {
			stdhttp.Error(writer,
				"configure positive Diagnostics.BlockProfileRate and MutexProfileFraction",
				stdhttp.StatusServiceUnavailable)
			return
		}
		controller.samplingGeneration++
		id := controller.samplingGeneration
		controller.samplingID = id
		runtime.SetBlockProfileRate(controller.blockProfileRate)
		runtime.SetMutexProfileFraction(controller.mutexProfileFraction)
		controller.samplingTimer = time.AfterFunc(duration, func() {
			controller.mu.Lock()
			defer controller.mu.Unlock()
			controller.closeSamplingLocked(id)
		})
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]uint64{"id": id})
	case stdhttp.MethodDelete:
		id, err := strconv.ParseUint(request.URL.Query().Get("id"), 10, 64)
		if err != nil || id == 0 || id > controller.samplingGeneration {
			stdhttp.Error(writer, "invalid sampling id", stdhttp.StatusBadRequest)
			return
		}
		if controller.samplingID != 0 && controller.samplingID != id {
			stdhttp.Error(writer, "sampling owned by another session", stdhttp.StatusConflict)
			return
		}
		controller.closeSamplingLocked(id)
		writer.WriteHeader(stdhttp.StatusNoContent)
	default:
		writer.Header().Set("Allow", "POST, DELETE")
		stdhttp.Error(writer, "method not allowed", stdhttp.StatusMethodNotAllowed)
	}
}

// closeSamplingLocked 在持锁状态下关闭指定会话并停止到期计时器。
func (controller *runtimeProfileController) closeSamplingLocked(id uint64) {
	if id == 0 || controller.samplingID != id {
		return
	}
	controller.samplingTimer.Stop()
	controller.samplingTimer = nil
	controller.samplingID = 0
	runtime.SetBlockProfileRate(0)
	runtime.SetMutexProfileFraction(0)
}
