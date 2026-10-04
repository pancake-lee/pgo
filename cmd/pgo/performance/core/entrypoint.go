package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
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

// Scenario defines the behavior supplied by one performance scenario.
type Scenario struct {
	Name            string
	Short           string
	NewPreparer     func(Config, klog.Logger) Preparer
	DefaultRPS      string
	DefaultDuration string
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

// NewCobraCommand 创建带公共负载参数、数据目录和清理子命令的场景入口。
func (entrypoint *Entrypoint) NewCobraCommand() *cobra.Command {
	outputDir := ""
	rpsInput := defaultRPSInput
	durationInput := defaultDurationInput
	if entrypoint.scenario.DefaultRPS != "" {
		rpsInput = entrypoint.scenario.DefaultRPS
	}
	if entrypoint.scenario.DefaultDuration != "" {
		durationInput = entrypoint.scenario.DefaultDuration
	}
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
				outputDir,
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
	command.Flags().StringVar(&outputDir, "output-dir", "",
		"report directory and reusable test data directory")
	command.AddCommand(entrypoint.CleanupEntrypoint().newCommand("cleanup"))
	return command
}

// RunInteractive 读取公共参数并记住数据目录，供测试和清理共同使用。
func (entrypoint *Entrypoint) RunInteractive() {
	now := time.Now()
	outputDir := getDefaultOutputDir(entrypoint.scenario.Name, now)
	paramList := getParamList(outputDir)
	for index := range paramList {
		if paramList[index].Name == "rps" && entrypoint.scenario.DefaultRPS != "" {
			paramList[index].Default = entrypoint.scenario.DefaultRPS
		}
		if paramList[index].Name == "duration" && entrypoint.scenario.DefaultDuration != "" {
			paramList[index].Default = entrypoint.scenario.DefaultDuration
		}
	}
	paramMap := pclient.GetCachedParamMap(
		pconfig.GetDefaultCachePath(),
		"client.performance."+entrypoint.scenario.Name+".",
		paramList,
	)
	// 首次直接接受默认目录时也保存，清理入口才能定位同一批次。
	err := pconfig.SetCacheValue(pconfig.GetDefaultCachePath(),
		"client.performance."+entrypoint.scenario.Name+".output-dir",
		paramMap["output-dir"])
	if err != nil {
		pthird.Interact.Errorf("Cannot remember test data directory: %v", err)
		return
	}
	logger := plogger.GetDefaultLoggerNoCaller()
	err = entrypoint.run(
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

func (entrypoint *Entrypoint) run(
	ctx context.Context,
	logger klog.Logger,
	portalURL string,
	rpsInput string,
	durationInput string,
	outputDir string,
	now time.Time,
) error {
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

	runner := NewRunner(logger)
	preparer := entrypoint.scenario.NewPreparer(config, logger)
	if !automatic {
		return runner.Run(ctx, config, preparer)
	}
	return runner.RunAutomatic(ctx, config, preparer, rpsList)
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
