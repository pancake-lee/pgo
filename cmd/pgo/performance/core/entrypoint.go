package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/pancake-lee/pgo/cmd/pgo/common"
	"github.com/pancake-lee/pgo/pkg/pclient"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/pthird"
	"github.com/spf13/cobra"
)

const (
	defaultPortalURL     = "http://127.0.0.1:20080"
	defaultOutputRoot    = ".local/performance"
	defaultRPSInput      = "auto"
	defaultDurationInput = "60s"
	defaultDuration      = 60 * time.Second
	discoveryTimeout     = 5 * time.Second
)

var autoRPSList = []int{200, 400, 600, 800, 1000}

// Scenario 保存性能场景的准备流程与默认负载参数。
type Scenario struct {
	Name                  string
	Short                 string
	AutomaticSummaryTitle string
	NewPreparer           func(Config, klog.Logger) Preparer
	Warmup                time.Duration
	DefaultRPS            string
}

// Entrypoint exposes one scenario through Cobra and the interactive menu.
type Entrypoint struct {
	scenario Scenario
}

// NewEntrypoint creates a shared performance command entrypoint.
func NewEntrypoint(scenario Scenario) *Entrypoint {
	if strings.TrimSpace(scenario.Name) == "" {
		panic("performance scenario name is required")
	}
	if strings.TrimSpace(scenario.Short) == "" {
		panic("performance scenario description is required")
	}
	if scenario.NewPreparer == nil {
		panic("performance scenario preparer is required")
	}
	return &Entrypoint{scenario: scenario}
}

// NewCobraCommand 使用公共参数与场景默认值创建性能命令。
func (entrypoint *Entrypoint) NewCobraCommand() *cobra.Command {
	rpsInput := defaultRPSInput
	if entrypoint.scenario.DefaultRPS != "" {
		rpsInput = entrypoint.scenario.DefaultRPS
	}
	durationInput := defaultDurationInput
	command := &cobra.Command{
		Use:   entrypoint.scenario.Name + " <portal-url>",
		Short: entrypoint.scenario.Short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := klog.NewStdLogger(cmd.OutOrStdout())
			return entrypoint.run(
				cmd.Context(),
				logger,
				args[0],
				rpsInput,
				durationInput,
				"",
				time.Now(),
			)
		},
	}
	command.Flags().StringVar(
		&rpsInput,
		"rps",
		rpsInput,
		"run one positive RPS level instead of the automatic ladder",
	)
	command.Flags().StringVar(
		&durationInput,
		"duration",
		durationInput,
		"measured load duration for each RPS level",
	)
	return command
}

// RunInteractive 读取公共参数与场景默认值并执行性能测试。
func (entrypoint *Entrypoint) RunInteractive() {
	now := time.Now()
	outputDir := getDefaultOutputDir(entrypoint.scenario.Name, now)
	paramList := getParamList(outputDir)
	if entrypoint.scenario.DefaultRPS != "" {
		paramList[1].Default = entrypoint.scenario.DefaultRPS
	}
	paramMap := pclient.GetCachedParamMap(
		pconfig.GetDefaultCachePath(),
		"client.performance."+entrypoint.scenario.Name+".",
		paramList,
	)
	logger := plogger.GetDefaultLoggerNoCaller()
	err := entrypoint.run(
		context.Background(),
		logger,
		paramMap["portal-url"],
		paramMap["rps"],
		paramMap["duration"],
		paramMap["output-dir"],
		now,
	)
	if err != nil {
		pthird.Interact.Errorf("Performance test failed: %v", err)
	}
}

func getParamList(outputDir string) []pclient.ParamItem {
	return []pclient.ParamItem{
		{Name: "portal-url", Usage: "portal URL", Default: defaultPortalURL},
		{Name: "rps", Usage: "RPS or auto", Default: defaultRPSInput},
		{
			Name:    "duration",
			Usage:   "duration for each RPS level",
			Default: defaultDurationInput,
		},
		{
			Name:    "output-dir",
			Usage:   "output directory",
			Default: outputDir,
		},
	}
}

// run 协调服务发现、单档或自动负载并响应中断。
func (entrypoint *Entrypoint) run(
	ctx context.Context,
	logger klog.Logger,
	portalURL string,
	rpsInput string,
	durationInput string,
	outputDir string,
	now time.Time,
) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	duration, err := parseDuration(durationInput)
	if err != nil {
		return err
	}
	rpsList, automatic, err := resolveRPSList(rpsInput)
	if err != nil {
		return err
	}
	config, err := buildLoadConfig(
		ctx,
		portalURL,
		entrypoint.scenario.Name,
		rpsList[0],
		duration,
		outputDir,
		now,
	)
	if err != nil {
		return err
	}

	config.Warmup = entrypoint.scenario.Warmup
	runner := NewRunner(logger)
	runStage := func(ctx context.Context, stageConfig Config) error {
		preparer := entrypoint.scenario.NewPreparer(stageConfig, logger)
		return runner.Run(ctx, stageConfig, preparer)
	}
	if !automatic {
		return runStage(ctx, config)
	}

	options := AutomaticOptions{
		RPSList:      rpsList,
		SummaryTitle: entrypoint.scenario.AutomaticSummaryTitle,
	}
	return RunAutomatic(ctx, runner, config, options, runStage)
}

func buildLoadConfig(
	ctx context.Context,
	portalURL string,
	scenarioName string,
	rps int,
	duration time.Duration,
	outputDir string,
	now time.Time,
) (Config, error) {
	discoveryContext, cancelDiscovery := context.WithTimeout(
		ctx,
		discoveryTimeout,
	)
	defer cancelDiscovery()

	httpClient := &http.Client{Timeout: discoveryTimeout}
	serviceList, err := common.DiscoverPortalServices(
		discoveryContext,
		httpClient,
		portalURL,
	)
	if err != nil {
		return Config{}, err
	}

	outputDir = strings.TrimSpace(outputDir)
	if outputDir == "" {
		outputDir = getDefaultOutputDir(scenarioName, now)
	}
	return Config{
		APIURL:    serviceList.APIURL,
		RPS:       rps,
		Duration:  duration,
		OutputDir: outputDir,
	}, nil
}

func getDefaultOutputDir(scenarioName string, now time.Time) string {
	timestamp := now.UTC().Format("20060102-150405.000000000Z")
	return filepath.Join(defaultOutputRoot, scenarioName, timestamp)
}

func resolveRPSList(input string) ([]int, bool, error) {
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "auto" {
		if len(autoRPSList) == 0 {
			return nil, false, errors.New("automatic RPS list is empty")
		}
		return append([]int(nil), autoRPSList...), true, nil
	}
	rps, err := strconv.Atoi(input)
	if err != nil || rps <= 0 {
		return nil, false, fmt.Errorf(
			"rps must be a positive integer or auto, got %q",
			input,
		)
	}
	return []int{rps}, false, nil
}

func parseDuration(input string) (time.Duration, error) {
	duration, err := time.ParseDuration(strings.TrimSpace(input))
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf(
			"duration must be a positive Go duration, got %q",
			input,
		)
	}
	return duration, nil
}
