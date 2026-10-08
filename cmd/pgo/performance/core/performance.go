package core

import (
	"context"
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/pancake-lee/pgo/pkg/putil"
)

// WarmupLoadRatio 控制预热速率为正式负载的 80%。
const WarmupLoadRatio = 0.8

// warmupDrain 留出整分钟前五秒供预热请求收尾。
const warmupDrain = 5 * time.Second

const (
	expectedVegetaVersion = "v12.13.0"
	defaultVegetaPath     = "vegeta"
	defaultWarmup         = 10 * time.Second
	defaultTimeout        = 5 * time.Second
)

const (
	runFileName           = "00-run.json"
	vegetaResultsFileName = "10-vegeta-results.bin"
	vegetaReportFileName  = "11-vegeta-report.txt"
)

// ErrLoadFailed 表示报告已完成但存在非成功响应，区别于执行器故障。
var ErrLoadFailed = errors.New("load stopped")

// Config 保存所有性能场景共用的单档负载配置。
type Config struct {
	APIURL         string        `json:"apiURL"`
	DiagnosticsURL string        `json:"diagnosticsURL,omitempty"`
	Sampling       bool          `json:"sampling"`
	RPS            int           `json:"rps"`
	Duration       time.Duration `json:"duration"`
	OutputDir      string        `json:"outputDir"`
}

// Preparer 定义单档负载执行前的场景准备契约。
type Preparer interface {
	Prepare(context.Context, string,
	) (targetPath string, cleanup func() error, err error)
}

// ResultDirectory 允许场景将当前结果与准备数据分开保存。
type ResultDirectory interface {
	ResultDirectory(string) string
}

// resultConfig 定位本轮报告目录，准备数据仍使用用户输入的输出目录。
func resultConfig(config Config, preparer Preparer) Config {
	if layout, ok := preparer.(ResultDirectory); ok {
		config.OutputDir = layout.ResultDirectory(config.OutputDir)
	}
	return config
}

// PreparedLoader 允许场景使用已准备的数据编排多个独立负载流。
type PreparedLoader interface {
	RunLoad(context.Context, *Runner, Config, string) error
}

// Runner 协调场景准备与 Vegeta 单档负载。
type Runner struct {
	execContext func(context.Context, io.Writer, string, ...string) (string, error)
	checkVegeta func(string) error
	waitUntil   func(context.Context, time.Time) error
	logger      klog.Logger
}

// NewRunner 创建单档负载执行器。
func NewRunner(logger klog.Logger) *Runner {
	return &Runner{
		execContext: putil.ExecContext,
		checkVegeta: checkVegetaVersion,
		waitUntil:   waitUntil,
		logger:      logger,
	}
}

// Run 执行一次场景准备、预热、正式负载、报告生成与清理。
func (runner *Runner) Run(ctx context.Context, config Config, preparer Preparer,
) error {
	return runner.runPrepared(ctx, config, preparer, func(targetPath string) error {
		return runner.loadPrepared(ctx, resultConfig(config, preparer), preparer, targetPath)
	})
}

// RunAutomatic 准备一次数据，所有档位复用目标，整轮结束后统一清理。
func (runner *Runner) RunAutomatic(ctx context.Context, config Config,
	preparer Preparer, rpsList []int,
) error {
	return runner.runPrepared(ctx, config, preparer, func(targetPath string) error {
		return RunAutomatic(ctx, runner, resultConfig(config, preparer), rpsList,
			func(ctx context.Context, stageConfig Config) error {
				return runner.loadPrepared(ctx, stageConfig, preparer, targetPath)
			})
	})
}

// runPrepared 管理数据准备与收尾，实际负载由调用方编排。
func (runner *Runner) runPrepared(ctx context.Context, config Config,
	preparer Preparer, load func(string) error,
) (runErr error) {
	err := config.validate()
	if err != nil {
		return err
	}

	err = runner.checkVegeta(defaultVegetaPath)
	if err != nil {
		return err
	}

	err = os.MkdirAll(config.OutputDir, 0o700)
	if err != nil {
		return err
	}

	targetPath, cleanup, err := preparer.Prepare(ctx, config.OutputDir)
	if cleanup != nil {
		defer func() {
			runErr = errors.Join(runErr, cleanup())
		}()
	}
	// 准备阶段清理旧结果后再记录本轮参数，失败时也保留输入。
	reportConfig := resultConfig(config, preparer)
	if reportErr := os.MkdirAll(reportConfig.OutputDir, 0o700); reportErr != nil {
		return errors.Join(err, reportErr)
	}
	if reportErr := writeJSON(filepath.Join(reportConfig.OutputDir, runFileName), config); reportErr != nil {
		return errors.Join(err, reportErr)
	}
	if err != nil {
		return err
	}
	return load(targetPath)
}

// loadPrepared 将本档参数和报告保存在独立目录，复用场景数据。
func (runner *Runner) loadPrepared(ctx context.Context, config Config,
	preparer Preparer, targetPath string,
) error {
	err := os.MkdirAll(config.OutputDir, 0o700)
	if err != nil {
		return err
	}
	err = writeJSON(filepath.Join(config.OutputDir, runFileName), config)
	if err != nil {
		return err
	}
	loader, custom := preparer.(PreparedLoader)
	if custom {
		return loader.RunLoad(ctx, runner, config, targetPath)
	}
	return runner.runLoad(ctx, config, targetPath)
}

// RunWarmup 对已准备的目标执行预热，不保存预热结果。
func (runner *Runner) RunWarmup(ctx context.Context, config Config,
	targetPath string, duration time.Duration,
) error {
	return runner.runAttack(ctx, config, targetPath, duration,
		time.Duration(float64(time.Second)/WarmupLoadRatio), io.Discard)
}

// RunMeasured 保存一个已准备负载流的结果，保持通用报告格式。
func (runner *Runner) RunMeasured(ctx context.Context, config Config,
	targetPath string,
) error {
	err := config.validate()
	if err != nil {
		return err
	}
	err = os.MkdirAll(config.OutputDir, 0o700)
	if err != nil {
		return err
	}
	err = writeJSON(filepath.Join(config.OutputDir, runFileName), config)
	if err != nil {
		return err
	}
	err = runner.runMeasured(ctx, config, targetPath)
	if err != nil {
		return err
	}
	result, err := readStageResult(config)
	if err != nil {
		return err
	}
	if result.Success < 1 {
		return fmt.Errorf("%w: success ratio %.2f%%; see %s",
			ErrLoadFailed, result.Success*100, config.OutputDir)
	}
	return nil
}

// WarmupWindow 将最低预热时长延长至整分钟前五秒，返回发请求时长与正式开始时刻。
func WarmupWindow(now time.Time, minimum time.Duration) (time.Duration, time.Time) {
	earliest := now.Add(minimum + warmupDrain)
	start := earliest.Truncate(time.Minute).Add(time.Minute)
	return start.Add(-warmupDrain).Sub(now), start
}

// waitUntil 等待目标时刻，允许调用方取消等待。
func waitUntil(ctx context.Context, start time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	timer := time.NewTimer(time.Until(start))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}

// WaitForMeasured 等待预定正式窗口，并在收尾超过预定时刻时记录实际延迟。
func (runner *Runner) WaitForMeasured(ctx context.Context, start time.Time) error {
	runner.info("waiting for measured window",
		"plannedStartUTC", start.UTC().Format(time.RFC3339))
	if time.Now().After(start) {
		runner.info("warmup drained after planned start", "delay", time.Since(start))
	}
	return runner.waitUntil(ctx, start)
}

// runLoad 延长预热并预留五秒收尾，然后在整分钟启动正式负载。
func (runner *Runner) runLoad(
	ctx context.Context,
	config Config,
	targetPath string,
) (runErr error) {
	warmupDuration, start := WarmupWindow(time.Now(), defaultWarmup)
	runner.info(
		"warming up",
		"step",
		"4/7",
		"duration",
		warmupDuration,
		"rps",
		float64(config.RPS)*WarmupLoadRatio,
	)
	warmupErr := runner.RunWarmup(
		ctx,
		config,
		targetPath,
		warmupDuration,
	)
	if warmupErr != nil {
		return fmt.Errorf("Vegeta warmup: %w", warmupErr)
	}
	if err := runner.WaitForMeasured(ctx, start); err != nil {
		return err
	}
	stopSampling, duration, err := StartSampling(ctx, config)
	if err != nil {
		return err
	}
	if config.Sampling {
		runner.info("runtime sampling enabled", "duration", duration)
	}
	defer func() {
		err := stopSampling()
		runErr = errors.Join(runErr, err)
		if config.Sampling && err == nil {
			runner.info("runtime sampling disabled")
		}
	}()
	return runner.RunMeasured(ctx, config, targetPath)
}

// runMeasured 执行正式负载并保存报告。
func (runner *Runner) runMeasured(ctx context.Context, config Config,
	targetPath string,
) error {
	runner.info(
		"running measured load",
		"startUTC",
		time.Now().UTC().Format(time.RFC3339),
		"step",
		"5/7",
		"duration",
		config.Duration,
		"rps",
		config.RPS,
	)
	resultPath := filepath.Join(config.OutputDir, vegetaResultsFileName)
	resultFile, err := os.OpenFile(
		resultPath,
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
		0o600,
	)
	if err != nil {
		return err
	}

	attackErr := runner.runAttack(
		ctx,
		config,
		targetPath,
		config.Duration,
		time.Second,
		resultFile,
	)
	runner.info("measured load completed",
		"endUTC", time.Now().UTC().Format(time.RFC3339))
	closeErr := resultFile.Close()
	err = errors.Join(attackErr, closeErr)
	if err != nil {
		return fmt.Errorf("measured load: %w", err)
	}

	reportPath := filepath.Join(config.OutputDir, vegetaReportFileName)
	reportFile, err := os.OpenFile(
		reportPath,
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
		0o600,
	)
	if err != nil {
		return err
	}

	message, err := runner.execContext(
		ctx,
		reportFile,
		defaultVegetaPath,
		"report",
		"-type=text",
		resultPath,
	)
	if err != nil && message != "" {
		err = fmt.Errorf("%w: %s", err, message)
	} else if message != "" {
		_ = runner.logger.Log(
			klog.LevelWarn,
			"msg",
			message,
			"command",
			defaultVegetaPath,
		)
	}

	closeErr = reportFile.Close()
	err = errors.Join(err, closeErr)
	if err != nil {
		return fmt.Errorf("Vegeta report: %w", err)
	}

	runner.logArtifacts(config.OutputDir)
	return nil
}

func (runner *Runner) logArtifacts(outputDir string) {
	runner.info("load report", "file", filepath.Join(outputDir, vegetaReportFileName))
}

func (runner *Runner) runAttack(
	ctx context.Context,
	config Config,
	targetPath string,
	duration time.Duration,
	ratePeriod time.Duration,
	resultWriter io.Writer,
) error {
	argumentList := []string{
		"attack",
		"-format=json",
		"-targets=" + targetPath,
		fmt.Sprintf("-rate=%d/%s", config.RPS, ratePeriod),
		"-duration=" + duration.String(),
		"-timeout=" + defaultTimeout.String(),
	}
	message, err := runner.execContext(
		ctx,
		resultWriter,
		defaultVegetaPath,
		argumentList...,
	)
	if err != nil && message != "" {
		return fmt.Errorf("%w: %s", err, message)
	}
	if message != "" {
		_ = runner.logger.Log(
			klog.LevelWarn,
			"msg",
			message,
			"command",
			defaultVegetaPath,
		)
	}

	return err
}

func (runner *Runner) info(message string, keyvals ...any) {
	values := append([]any{"msg", message}, keyvals...)
	_ = runner.logger.Log(klog.LevelInfo, values...)
}

func (config Config) validate() error {
	_, err := url.ParseRequestURI(normalizeHTTPURL(config.APIURL))
	if err != nil {
		return fmt.Errorf("invalid API URL: %w", err)
	}
	if config.RPS <= 0 {
		return errors.New("rps must be positive")
	}
	if config.Duration <= 0 {
		return errors.New("duration must be positive")
	}
	if strings.TrimSpace(config.OutputDir) == "" {
		return errors.New("output directory is required")
	}
	return nil
}

func checkVegetaVersion(path string) error {
	resolvedPath, err := exec.LookPath(path)
	if err != nil {
		return fmt.Errorf(
			"Vegeta %s is required but was not found in PATH; "+
				"install it with `go install "+
				"github.com/tsenart/vegeta/v12@%s`: %w",
			expectedVegetaVersion,
			expectedVegetaVersion,
			err,
		)
	}

	info, err := buildinfo.ReadFile(resolvedPath)
	if err != nil {
		return fmt.Errorf("read Vegeta build info: %w", err)
	}
	pathMismatch := info.Main.Path != "github.com/tsenart/vegeta/v12"
	versionMismatch := info.Main.Version != expectedVegetaVersion
	if pathMismatch || versionMismatch {
		return fmt.Errorf(
			"Vegeta %s is required, found %s %s",
			expectedVegetaVersion,
			info.Main.Path,
			info.Main.Version,
		)
	}
	return nil
}

func normalizeHTTPURL(address string) string {
	address = strings.TrimSpace(address)
	if !strings.Contains(address, "://") {
		return "http://" + address
	}
	return address
}

func writeJSON(path string, value any) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(content, '\n'), 0o600)
}

// StartSampling 根据公共开关控制正式负载采样，取消后仍尝试关闭会话。
func StartSampling(ctx context.Context, config Config,
) (func() error, time.Duration, error) {
	if !config.Sampling {
		return func() error { return nil }, 0, nil
	}
	address, err := url.Parse(config.DiagnosticsURL)
	if err != nil || address.Host == "" ||
		(address.Scheme != "http" && address.Scheme != "https") {
		return nil, 0, errors.New("valid diagnostics URL is required for runtime sampling")
	}
	if config.Duration <= 0 {
		return nil, 0, errors.New("measured duration must be positive")
	}
	address.Path = "/debug/pprof/sampling"
	address.RawQuery = ""
	address.Fragment = ""
	// 请求正式负载时长，后端截断到采样上限；秒级有效期向上取整。
	seconds := int64((config.Duration-1)/time.Second) + 1
	query := url.Values{"seconds": {strconv.FormatInt(seconds, 10)}}
	address.RawQuery = query.Encode()
	client := &http.Client{Timeout: 5 * time.Second}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, address.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, 0, fmt.Errorf("enable runtime sampling: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return nil, 0, fmt.Errorf("enable runtime sampling: %s: %s",
			response.Status, strings.TrimSpace(string(body)))
	}
	var session struct {
		ID      uint64 `json:"id"`
		Seconds uint64 `json:"seconds"`
	}
	err = json.NewDecoder(response.Body).Decode(&session)
	if err != nil || session.ID == 0 || session.Seconds == 0 {
		return nil, 0, fmt.Errorf("invalid runtime sampling session: id=%d, error=%v", session.ID, err)
	}
	return func() error {
		cleanupContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		address.RawQuery = url.Values{"id": {strconv.FormatUint(session.ID, 10)}}.Encode()
		request, err := http.NewRequestWithContext(cleanupContext,
			http.MethodDelete, address.String(), nil)
		if err != nil {
			return err
		}
		response, err := client.Do(request)
		if err != nil {
			return fmt.Errorf("disable runtime sampling: %w", err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusNoContent {
			return fmt.Errorf("disable runtime sampling: %s", response.Status)
		}
		return nil
	}, time.Duration(session.Seconds) * time.Second, nil
}
