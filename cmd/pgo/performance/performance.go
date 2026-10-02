package performance

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
	"strings"
	"time"

	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/pancake-lee/pgo/cmd/pgo/common"
	"github.com/pancake-lee/pgo/pkg/pclient"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/pthird"
	"github.com/pancake-lee/pgo/pkg/putil"
	"github.com/spf13/cobra"
)

const (
	expectedVegetaVersion = "v12.13.0"
	defaultPortalURL      = "http://127.0.0.1:20080"
	defaultOutputRoot     = ".local/performance"
	defaultVegetaPath     = "vegeta"
	defaultWarmup         = 10 * time.Second
	defaultDuration       = 60 * time.Second
	defaultTimeout        = 5 * time.Second
)

const (
	runFileName           = "00-run.json"
	vegetaResultsFileName = "10-vegeta-results.bin"
	vegetaReportFileName  = "11-vegeta-report.txt"
)

// loadConfig 保存所有性能场景共用的负载配置。
type loadConfig struct {
	APIURL    string `json:"apiURL"`
	RPS       int    `json:"rps"`
	RPSInput  string `json:"-"`
	OutputDir string `json:"outputDir"`
}

// commandRunner 抽象外部压测命令的执行方式。
type commandRunner interface {
	Run(context.Context, string, []string, io.Writer) error
}

// execRunner 使用本地子进程执行外部命令。
type execRunner struct {
	logger klog.Logger
}

// Run 执行外部命令，将标准输出写入结果并在内部处理标准错误。
func (runner execRunner) Run(ctx context.Context, name string, args []string, resultWriter io.Writer) error {
	message, err := putil.ExecContext(ctx, resultWriter, name, args...)
	if err != nil && message != "" {
		return fmt.Errorf("%w: %s", err, message)
	}
	if message != "" {
		_ = runner.logger.Log(klog.LevelWarn, "msg", message, "command", name)
	}
	return err
}

// loadRunner 协调外部负载命令、服务发现与时间来源。
type loadRunner struct {
	commandRunner commandRunner
	httpClient    *http.Client
	checkVegeta   func(string) error // 轻量级注入，测试跳过
	now           func() time.Time
	logger        klog.Logger
}

// artifact 描述一个压测产物的文件名和用途。
type artifact struct {
	Path        string
	Description string
}

// --------------------------------------------------
// NewCommand 创建性能测试的 Cobra 根命令。
func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "performance",
		Short: "Run repeatable performance test scenarios",
	}
	command.AddCommand(newLoginCommand())
	return command
}

// RunInteractive 启动性能测试场景的交互式入口。
func RunInteractive() {
	selector := pthird.Interact.NewSelector("请选择性能测试场景 (Select Performance Scenario)")
	selector.Reg("用户登录压测 (User Login)", runLoginInteractive)
	selector.Loop()
}

// newLoginCommand 创建只接收导航页地址和可选 RPS 的登录压测命令。
func newLoginCommand() *cobra.Command {
	rpsInput := "auto"
	command := &cobra.Command{
		Use:   "login <portal-url>",
		Short: "Run the user login load-test workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runner := newLoadRunner(klog.NewStdLogger(cmd.OutOrStdout()))
			config, err := defaultLoadConfig(cmd.Context(), runner, args[0], rpsInput)
			if err != nil {
				return err
			}
			return runLoginPlan(cmd.Context(), runner, config)
		},
	}
	command.Flags().StringVar(&rpsInput, "rps", rpsInput, "run one positive RPS level instead of the built-in automatic ladder")
	return command
}

// --------------------------------------------------
// runLoginInteractive 读取导航页地址并运行默认自动阶梯压测。
func runLoginInteractive() {
	cachePath := pconfig.GetDefaultCachePath()
	portalURL := pclient.GetCachedParam(cachePath, "client.performance.login.portal", "portal URL", defaultPortalURL)
	runner := newLoadRunner(plogger.GetDefaultLoggerNoCaller())
	config, err := defaultLoadConfig(context.Background(), runner, portalURL, "auto")
	if err != nil {
		pthird.Interact.Errorf("Invalid performance parameter: %v", err)
		return
	}
	pthird.Interact.Infof("Starting login performance test; artifacts: %s", config.OutputDir)
	if err = runLoginPlan(context.Background(), runner, config); err != nil {
		pthird.Interact.Errorf("Login performance test failed: %v", err)
	}
}

// defaultLoadConfig 从导航页发现 API 并生成登录压测的默认配置。
func defaultLoadConfig(
	ctx context.Context,
	runner *loadRunner,
	portalURL string,
	rpsInput string,
) (loadConfig, error) {
	discoveryContext, cancelDiscovery := context.WithTimeout(ctx, defaultTimeout)
	defer cancelDiscovery()
	serviceList, err := common.DiscoverPortalServices(discoveryContext, runner.httpClient, portalURL)
	if err != nil {
		return loadConfig{}, err
	}
	config := loadConfig{
		APIURL:    serviceList.APIURL,
		RPS:       10,
		RPSInput:  rpsInput,
		OutputDir: filepath.Join(defaultOutputRoot, "login", runner.now().UTC().Format("20060102-150405.000000000Z")),
	}
	return config, config.validate()
}

// newLoadRunner 创建负载执行器。
func newLoadRunner(logger klog.Logger) *loadRunner {
	runner := &loadRunner{
		httpClient:  &http.Client{Timeout: defaultTimeout},
		checkVegeta: checkVegetaVersion,
		now:         time.Now,
		logger:      logger,
	}
	runner.commandRunner = execRunner{logger: logger}
	return runner
}

// runLoginStage 使用登录场景执行一个完整压力档位。
func (runner *loadRunner) runLoginStage(ctx context.Context, config loadConfig) error {
	if err := config.validate(); err != nil {
		return err
	}
	preparer := newLoginPreparer(config, runner.logger)
	return runner.runPreparedStage(ctx, config, preparer)
}

// runPreparedStage 完成依赖预检、负载准备、执行与清理。
func (runner *loadRunner) runPreparedStage(
	ctx context.Context,
	config loadConfig,
	preparer preparer,
) (runErr error) {
	if err := config.validate(); err != nil {
		return err
	}
	if err := runner.checkVegeta(defaultVegetaPath); err != nil {
		return err
	}
	if err := os.MkdirAll(config.OutputDir, 0o700); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(config.OutputDir, runFileName), config); err != nil {
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
	return runner.runLoadStage(ctx, config, targetPath)
}

// runLoadStage 执行预热、正式负载和 Vegeta 报告生成。
func (runner *loadRunner) runLoadStage(
	ctx context.Context,
	config loadConfig,
	targetPath string,
) error {
	runner.info("warming up", "step", "4/7", "duration", defaultWarmup, "rps", config.RPS)
	if err := runner.runAttack(ctx, config, targetPath, defaultWarmup, io.Discard); err != nil {
		return fmt.Errorf("Vegeta warmup: %w", err)
	}

	runner.info("running measured load", "step", "5/7", "duration", defaultDuration, "rps", config.RPS)
	resultPath := filepath.Join(config.OutputDir, vegetaResultsFileName)
	resultFile, err := os.OpenFile(resultPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	attackErr := runner.runAttack(ctx, config, targetPath, defaultDuration, resultFile)
	closeErr := resultFile.Close()
	if err = errors.Join(attackErr, closeErr); err != nil {
		return fmt.Errorf("measured load: %w", err)
	}

	runner.info("generating Vegeta report", "step", "6/7")
	reportPath := filepath.Join(config.OutputDir, vegetaReportFileName)
	reportFile, err := os.OpenFile(reportPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	// Vegeta remains an external process for now. If installation or process control becomes
	// a recurring problem, consider replacing this boundary with the Vegeta Go package.
	err = runner.commandRunner.Run(ctx, defaultVegetaPath, []string{"report", "-type=text", resultPath}, reportFile)
	closeErr = reportFile.Close()
	if err = errors.Join(err, closeErr); err != nil {
		return fmt.Errorf("Vegeta report: %w", err)
	}
	runner.logArtifacts(config.OutputDir)
	return nil
}

// runAttack 调用 Vegeta 对目标文件产生指定时长的负载。
func (runner *loadRunner) runAttack(
	ctx context.Context,
	config loadConfig,
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
	// Vegeta remains an external process for now. If installation or process control becomes
	// a recurring problem, consider replacing this boundary with the Vegeta Go package.
	return runner.commandRunner.Run(ctx, defaultVegetaPath, argumentList, resultWriter)
}

// validate 校验负载配置。
func (config loadConfig) validate() error {
	if _, err := url.ParseRequestURI(normalizeHTTPURL(config.APIURL)); err != nil {
		return fmt.Errorf("invalid API URL: %w", err)
	}
	if config.RPS <= 0 {
		return errors.New("rps must be positive")
	}
	if strings.TrimSpace(config.OutputDir) == "" {
		return errors.New("output directory is required")
	}
	return nil
}

// checkVegetaVersion 检查 Vegeta 是否存在且版本精确匹配。
func checkVegetaVersion(path string) error {
	resolvedPath, err := exec.LookPath(path)
	if err != nil {
		return fmt.Errorf("Vegeta %s is required but was not found in PATH; install it with `go install github.com/tsenart/vegeta/v12@%s`: %w",
			expectedVegetaVersion, expectedVegetaVersion, err)
	}
	info, err := buildinfo.ReadFile(resolvedPath)
	if err != nil {
		return fmt.Errorf("read Vegeta build info: %w", err)
	}
	if info.Main.Path != "github.com/tsenart/vegeta/v12" || info.Main.Version != expectedVegetaVersion {
		return fmt.Errorf("Vegeta %s is required, found %s %s", expectedVegetaVersion, info.Main.Path, info.Main.Version)
	}
	return nil
}

// normalizeHTTPURL 为缺少协议的 HTTP 地址补充默认协议。
func normalizeHTTPURL(address string) string {
	address = strings.TrimSpace(address)
	if !strings.Contains(address, "://") {
		return "http://" + address
	}
	return address
}

// writeJSON 将运行数据格式化后写入 JSON 文件。
func writeJSON(path string, value any) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(content, '\n'), 0o600)
}

// logArtifacts 记录本轮生成的文件及其用途。
func (runner *loadRunner) logArtifacts(outputDir string) {
	artifactList := []artifact{
		{Path: runFileName, Description: "input parameters for reproducing this run"},
		{Path: usersFileName, Description: "batch manifest used for verification and precise cleanup; contains tokens"},
		{Path: loginTargetsFileName, Description: "Vegeta login request definitions"},
		{Path: vegetaResultsFileName, Description: "raw Vegeta request results"},
		{Path: vegetaReportFileName, Description: "human-readable request count, throughput, success, and latency report"},
	}
	runner.info("load artifacts", "outputDir", outputDir)
	for _, item := range artifactList {
		runner.info("load artifact", "file", item.Path, "description", item.Description)
	}
}

// info 记录压测流程进度。
func (runner *loadRunner) info(message string, keyvals ...any) {
	values := append([]any{"msg", message}, keyvals...)
	_ = runner.logger.Log(klog.LevelInfo, values...)
}
