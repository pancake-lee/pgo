package login

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
	"github.com/pancake-lee/pgo/cmd/pgo/performance"
	"github.com/pancake-lee/pgo/pkg/pclient"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/pthird"
	"github.com/spf13/cobra"
)

const (
	defaultPortalURL  = "http://127.0.0.1:20080"
	defaultOutputRoot = ".local/performance"
	defaultTimeout    = 5 * time.Second
	loginUserCount    = 100
	usersFileName     = "01-users.json"
	targetsFileName   = "02-login-targets.jsonl"
)

var autoRPSList = []int{10, 25, 50, 100, 200, 500}

func init() {
	performance.RegisterScenario(performance.Scenario{
		Name:             "login",
		InteractiveLabel: "用户登录压测 (User Login)",
		NewCommand:       newLoginCommand,
		RunInteractive:   runLoginInteractive,
	})
}

// newLoginCommand 创建只接收导航页地址和可选 RPS 的登录压测命令。
func newLoginCommand() *cobra.Command {
	rpsInput := "auto"
	command := &cobra.Command{
		Use:   "login <portal-url>",
		Short: "Run the user login load-test workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := klog.NewStdLogger(cmd.OutOrStdout())
			runner := performance.NewRunner(logger)
			config, err := defaultLoadConfig(cmd.Context(), args[0], time.Now())
			if err != nil {
				return err
			}
			return runPlan(cmd.Context(), runner, logger, config, rpsInput)
		},
	}
	command.Flags().StringVar(
		&rpsInput,
		"rps",
		rpsInput,
		"run one positive RPS level instead of the "+
			"built-in automatic ladder",
	)
	return command
}

func runLoginInteractive() {
	cachePath := pconfig.GetDefaultCachePath()
	portalURL := pclient.GetCachedParam(
		cachePath,
		"client.performance.login.portal",
		"portal URL",
		defaultPortalURL,
	)
	logger := plogger.GetDefaultLoggerNoCaller()
	runner := performance.NewRunner(logger)
	ctx := context.Background()
	config, err := defaultLoadConfig(ctx, portalURL, time.Now())
	if err != nil {
		pthird.Interact.Errorf("Invalid performance parameter: %v", err)
		return
	}

	pthird.Interact.Infof(
		"Starting login performance test; artifacts: %s",
		config.OutputDir,
	)
	err = runPlan(ctx, runner, logger, config, "auto")
	if err != nil {
		pthird.Interact.Errorf("Login performance test failed: %v", err)
	}
}

func defaultLoadConfig(ctx context.Context, portalURL string, now time.Time,
) (performance.Config, error) {
	discoveryContext, cancelDiscovery := context.WithTimeout(ctx, defaultTimeout)
	defer cancelDiscovery()

	httpClient := &http.Client{Timeout: defaultTimeout}
	serviceList, err := common.DiscoverPortalServices(
		discoveryContext,
		httpClient,
		portalURL,
	)
	if err != nil {
		return performance.Config{}, err
	}

	timestamp := now.UTC().Format("20060102-150405.000000000Z")
	return performance.Config{
		APIURL:    serviceList.APIURL,
		RPS:       10,
		OutputDir: filepath.Join(defaultOutputRoot, "login", timestamp),
	}, nil
}

func runPlan(
	ctx context.Context,
	runner *performance.Runner,
	logger klog.Logger,
	config performance.Config,
	rpsInput string,
) error {
	rpsList, automatic, err := resolveRPSList(rpsInput, config.RPS)
	if err != nil {
		return err
	}
	runStage := func(ctx context.Context, stageConfig performance.Config) error {
		return runner.Run(ctx, stageConfig, newPreparer(stageConfig, logger))
	}
	if !automatic {
		config.RPS = rpsList[0]
		return runStage(ctx, config)
	}

	options := performance.AutomaticOptions{
		RPSList:      rpsList,
		SummaryTitle: "Login automatic load result",
		SummaryIntroduction: "Built-in RPS ladder: " +
			"`10, 25, 50, 100, 200, 500`.",
	}
	return performance.RunAutomatic(ctx, runner, config, options, runStage)
}

func resolveRPSList(input string, fallback int) ([]int, bool, error) {
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		if fallback <= 0 {
			return nil, false, errors.New("rps must be a positive integer or auto")
		}
		return []int{fallback}, false, nil
	}
	if input == "auto" {
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

type preparer struct {
	config performance.Config
	logger klog.Logger
}

func newPreparer(config performance.Config, logger klog.Logger) *preparer {
	return &preparer{config: config, logger: logger}
}

func (preparer *preparer) Prepare(ctx context.Context, outputDir string,
) (string, func() error, error) {
	baseURL := normalizeHTTPURL(preparer.config.APIURL)
	client, err := common.NewClient(baseURL)
	if err != nil {
		return "", nil, err
	}

	manifestPath := filepath.Join(outputDir, usersFileName)
	targetPath := filepath.Join(outputDir, targetsFileName)

	preparer.info("preparing login users", "step", "1/7")
	manifest, err := Prepare(ctx, client, loginUserCount, manifestPath)
	var cleanup func() error
	if manifest != nil {
		cleanup = func() error {
			preparer.info("cleaning test users", "step", "7/7")
			return Cleanup(context.WithoutCancel(ctx), client, manifest)
		}
	}
	if err != nil {
		return "", cleanup, err
	}

	preparer.info("verifying identities and authenticated access", "step", "2/7")
	err = Verify(ctx, client, manifest)
	if err != nil {
		return "", cleanup, err
	}

	preparer.info("generating Vegeta login targets", "step", "3/7")
	err = manifest.WriteTargets(targetPath)
	if err != nil {
		return "", cleanup, err
	}

	preparer.info(
		"scenario artifact",
		"file",
		usersFileName,
		"description",
		"login user manifest; contains tokens",
	)
	preparer.info(
		"scenario artifact",
		"file",
		targetsFileName,
		"description",
		"Vegeta login request definitions",
	)
	return targetPath, cleanup, nil
}

func (preparer *preparer) info(message string, keyvals ...any) {
	values := append([]any{"msg", message}, keyvals...)
	_ = preparer.logger.Log(klog.LevelInfo, values...)
}

func normalizeHTTPURL(address string) string {
	address = strings.TrimSpace(address)
	if !strings.Contains(address, "://") {
		return "http://" + address
	}
	return address
}
