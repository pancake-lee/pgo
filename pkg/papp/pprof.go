package papp

import (
	"encoding/json"
	"errors"
	"fmt"
	stdhttp "net/http"
	"net/http/pprof"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// maxRuntimeProfileDuration 限制受控运行时 profile 的最长开放时间。
	maxRuntimeProfileDuration = 60 * time.Second
	// maxRuntimeTraceDuration 限制单次 runtime trace 的最长采集时间。
	maxRuntimeTraceDuration = 10 * time.Second
)

// runtimeProfileController 协调 pprof 采样参数、限时访问窗口及 CPU 和 trace 采集并发。
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
		activeProfileMap:     make(map[string]bool),
		blockProfileRate:     config.BlockProfileRate,
		mutexProfileFraction: config.MutexProfileFraction,
	}, nil
}

// registerRoutes 将持续及受控 pprof 接口注册到诊断路由。
func (controller *runtimeProfileController) registerRoutes(mux *stdhttp.ServeMux) {
	mux.HandleFunc("/debug/pprof/profile", controller.cpuProfileHandler)
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.HandleFunc("/debug/pprof/runtime", controller.runtimeProfileHandler)
	mux.HandleFunc("/debug/pprof/runtime-trace", controller.runtimeTraceHandler)
	for _, profileType := range []string{"goroutine", "block", "mutex"} {
		mux.Handle("/debug/pprof/"+profileType, controller.gatedProfileHandler(profileType))
	}
}

// cpuProfileHandler 串行采集指定时长的 CPU 调用栈样本。
func (controller *runtimeProfileController) cpuProfileHandler(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
	controller.cpuMu.Lock()
	defer controller.cpuMu.Unlock()
	pprof.Profile(writer, request)
}

// runtimeProfileHandler 按请求开启指定 runtime profile 的限时采样与访问窗口。
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

// gatedProfileHandler 仅在对应诊断窗口有效时提供指定 runtime profile。
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
	duration := time.Duration(seconds) * time.Second
	if duration > maximum {
		return 0, fmt.Errorf("seconds must not exceed %d", int(maximum/time.Second))
	}
	return duration, nil
}

// parseRuntimeProfileList 校验并去重请求中的 goroutine、block 和 mutex 类型。
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

// containsString 判断字符串列表是否包含指定值。
func containsString(valueList []string, target string) bool {
	for _, value := range valueList {
		if value == target {
			return true
		}
	}
	return false
}

// expire 仅关闭与定时任务代次一致的运行时诊断窗口。
func (controller *runtimeProfileController) expire(generation uint64) {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	if controller.generation == generation {
		controller.disableLocked()
	}
}

// expireLocked 在持有控制锁时关闭已经到期的诊断窗口。
func (controller *runtimeProfileController) expireLocked(now time.Time) {
	if !controller.expiresAt.IsZero() && !now.Before(controller.expiresAt) {
		controller.disableLocked()
	}
}

// disableLocked 在持有控制锁时停止 block/mutex 采样并清空窗口状态。
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

// stop 并发安全地关闭当前运行时诊断窗口。
func (controller *runtimeProfileController) stop() {
	controller.mu.Lock()
	defer controller.mu.Unlock()
	controller.disableLocked()
}
