package core

import (
	"context"
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/pancake-lee/pgo/pkg/putil"
)

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
	// loadWindowFileName 保存正式负载的 UTC 时间窗。
	loadWindowFileName = "12-load-window.json"
)

// loadWindow 记录正式负载区间以关联指标和持续画像。
type loadWindow struct {
	StartUTC time.Time `json:"startUTC"`
	EndUTC   time.Time `json:"endUTC"`
}

// Config 保存所有性能场景共用的单档负载配置。
type Config struct {
	APIURL    string        `json:"apiURL"`
	RPS       int           `json:"rps"`
	Duration  time.Duration `json:"duration"`
	Warmup    time.Duration `json:"warmup,omitempty"`
	OutputDir string        `json:"outputDir"`
}

// Preparer 定义单档负载执行前的场景准备契约。
type Preparer interface {
	Prepare(context.Context, string,
	) (targetPath string, cleanup func() error, err error)
}

// Runner 协调场景准备与 Vegeta 单档负载。
type Runner struct {
	execContext func(context.Context, io.Writer, string, ...string) (string, error)
	checkVegeta func(string) error
	logger      klog.Logger
}

// NewRunner 创建单档负载执行器。
func NewRunner(logger klog.Logger) *Runner {
	return &Runner{
		execContext: putil.ExecContext,
		checkVegeta: checkVegetaVersion,
		logger:      logger,
	}
}

// Run 执行一次场景准备、预热、正式负载、报告生成与清理。
func (runner *Runner) Run(ctx context.Context, config Config, preparer Preparer,
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

	runPath := filepath.Join(config.OutputDir, runFileName)
	err = writeJSON(runPath, config)
	if err != nil {
		return err
	}

	targetPath, cleanup, err := preparer.Prepare(ctx, config.OutputDir)
	if cleanup != nil {
		defer func() {
			runErr = errors.Join(runErr, cleanup())
		}()
	}
	if err != nil {
		return err
	}

	return runner.runLoad(ctx, config, targetPath)
}

// runLoad 执行预热与正式负载并记录 UTC 测量窗口。
func (runner *Runner) runLoad(
	ctx context.Context,
	config Config,
	targetPath string,
) error {
	warmup := config.Warmup
	if warmup == 0 {
		warmup = defaultWarmup
	}
	runner.info(
		"warming up",
		"step",
		"4/7",
		"duration",
		warmup,
		"rps",
		config.RPS,
	)
	warmupErr := runner.runAttack(
		ctx,
		config,
		targetPath,
		warmup,
		io.Discard,
	)
	if warmupErr != nil {
		return fmt.Errorf("Vegeta warmup: %w", warmupErr)
	}

	runner.info(
		"running measured load",
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

	window := loadWindow{StartUTC: time.Now().UTC()}
	runner.info("measurement window started", "startUTC", window.StartUTC)
	attackErr := runner.runAttack(
		ctx,
		config,
		targetPath,
		config.Duration,
		resultFile,
	)
	closeErr := resultFile.Close()
	window.EndUTC = time.Now().UTC()
	windowErr := writeJSON(
		filepath.Join(config.OutputDir, loadWindowFileName), window,
	)
	runner.info("measurement window ended", "endUTC", window.EndUTC)
	err = errors.Join(attackErr, closeErr, windowErr)
	if err != nil {
		return fmt.Errorf("measured load: %w", err)
	}

	runner.info("generating Vegeta report", "step", "6/7")
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

// logArtifacts 输出负载产物及其使用说明。
func (runner *Runner) logArtifacts(outputDir string) {
	runner.info("load artifacts", "outputDir", outputDir)
	artifactList := []struct {
		path        string
		description string
	}{
		{runFileName, "input parameters for reproducing this run"},
		{vegetaResultsFileName, "raw Vegeta request results"},
		{vegetaReportFileName, "human-readable load report"},
		{loadWindowFileName, "UTC measurement window for Grafana and Pyroscope"},
	}
	for _, item := range artifactList {
		runner.info(
			"load artifact",
			"file",
			item.path,
			"description",
			item.description,
		)
	}
}

func (runner *Runner) runAttack(
	ctx context.Context,
	config Config,
	targetPath string,
	duration time.Duration,
	resultWriter io.Writer,
) error {
	argumentList := []string{
		"attack",
		"-format=json",
		"-targets=" + targetPath,
		fmt.Sprintf("-rate=%d/s", config.RPS),
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

// validate 校验单档负载的地址、持续时间及输出参数。
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
	if config.Warmup < 0 {
		return errors.New("warmup must not be negative")
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
