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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pancake-lee/pgo/pkg/pclient"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/pthird"
	"github.com/spf13/cobra"
)

const (
	expectedVegetaVersion     = "v12.13.0"
	expectedProfileCLIVersion = "2.2.0"
	profileSourcePyroscope    = "pyroscope"
	profileSourcePprof        = "pprof"
)

const (
	runFileName              = "00-run.json"
	usersFileName            = "01-users.json"
	loginTargetsFileName     = "02-login-targets.jsonl"
	vegetaResultsFileName    = "10-vegeta-results.bin"
	vegetaReportFileName     = "11-vegeta-report.txt"
	metricsAfterFileName     = "20-metrics-after.prom"
	metricsBeforeFileName    = "21-metrics-before.prom"
	cpuProfileFileName       = "30-cpu.pprof"
	goroutineProfileFileName = "31-goroutine.pprof"
	heapProfileFileName      = "32-heap.pprof"
	blockProfileFileName     = "33-block.pprof"
	mutexProfileFileName     = "34-mutex.pprof"
	runtimeTraceFileName     = "35-runtime.trace"
	summaryFileName          = "40-summary.md"
)

type loadConfig struct {
	PprofURL       string        `json:"pprofURL"`
	PyroscopeURL   string        `json:"pyroscopeURL"`
	GrafanaURL     string        `json:"grafanaURL"`
	ProfileSource  string        `json:"profileSource"`
	ProfileCLIPath string        `json:"profileCLIPath"`
	ProfileService string        `json:"profileService"`
	RuntimeTrace   bool          `json:"runtimeTrace"`
	Users          int           `json:"users"`
	Concurrency    int           `json:"concurrency"`
	RPS            int           `json:"rps"`
	RPSInput       string        `json:"-"`
	Warmup         time.Duration `json:"warmup"`
	Duration       time.Duration `json:"duration"`
	Timeout        time.Duration `json:"timeout"`
	OutputDir      string        `json:"outputDir"`
	VegetaPath     string        `json:"vegetaPath"`
}

type loginConfig struct {
	loadConfig
	APIURL      string `json:"apiURL"`
	Users       int    `json:"users"`
	Concurrency int    `json:"concurrency"`
}

type commandRunner interface {
	Run(context.Context, string, []string, io.Writer, io.Writer) error
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args []string, stdout, stderr io.Writer) error {
	command := exec.CommandContext(ctx, name, args...)
	command.Stdout = stdout
	command.Stderr = stderr
	return command.Run()
}

type loadRunner struct {
	commandRunner   commandRunner
	httpClient      *http.Client
	checkVegeta     func(string) error
	checkProfileCLI func(string) error
	now             func() time.Time
}

type artifact struct {
	Path        string
	Description string
}

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "performance",
		Short: "Run repeatable performance test scenarios",
	}
	command.AddCommand(newLoginCommand())
	return command
}

func RunInteractive() {
	selector := pthird.Interact.NewSelector("请选择性能测试场景 (Select Performance Scenario)")
	selector.Reg("用户登录压测 (User Login)", runLoginInteractive)
	selector.Loop()
}

func newLoginCommand() *cobra.Command {
	config := defaultLoginConfig()
	command := &cobra.Command{
		Use:   "login",
		Short: "Run the complete user login load-test and diagnostics workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			runner := newLoadRunner(config.Timeout)
			return runLoginPlan(cmd.Context(), runner, config, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	command.Flags().StringVar(&config.APIURL, "api", config.APIURL, "user service HTTP base URL")
	command.Flags().StringVar(&config.PprofURL, "pprof", config.PprofURL, "pprof base URL")
	command.Flags().StringVar(&config.PyroscopeURL, "pyroscope", config.PyroscopeURL, "Pyroscope server URL")
	command.Flags().StringVar(&config.GrafanaURL, "grafana", config.GrafanaURL, "Grafana URL")
	command.Flags().StringVar(&config.ProfileSource, "profile-source", config.ProfileSource, "CPU and heap source: pyroscope or pprof")
	command.Flags().StringVar(&config.ProfileCLIPath, "profilecli", config.ProfileCLIPath, "profilecli executable path")
	command.Flags().StringVar(&config.ProfileService, "profile-service", config.ProfileService, "Pyroscope service_name label")
	command.Flags().BoolVar(&config.RuntimeTrace, "runtime-trace", config.RuntimeTrace, "capture one runtime trace during measured load")
	command.Flags().IntVar(&config.Users, "users", config.Users, "number of users in the test batch")
	command.Flags().IntVar(&config.Concurrency, "concurrency", config.Concurrency, "concurrent user preparation and verification requests")
	command.Flags().StringVar(&config.RPSInput, "rps", config.RPSInput, "login requests per second, or auto for the built-in load ladder")
	command.Flags().DurationVar(&config.Warmup, "warmup", config.Warmup, "warmup duration")
	command.Flags().DurationVar(&config.Duration, "duration", config.Duration, "measured load duration")
	command.Flags().DurationVar(&config.Timeout, "timeout", config.Timeout, "per-request timeout")
	command.Flags().StringVarP(&config.OutputDir, "output", "o", config.OutputDir, "artifact output directory")
	command.Flags().StringVar(&config.VegetaPath, "vegeta", config.VegetaPath, "Vegeta executable path")
	return command
}

func defaultLoginConfig() loginConfig {
	return loginConfig{
		loadConfig: loadConfig{
			PprofURL:       "http://127.0.0.1:20002/debug/pprof/",
			PyroscopeURL:   "http://127.0.0.1:24040",
			GrafanaURL:     "http://127.0.0.1:23000",
			ProfileSource:  profileSourcePyroscope,
			ProfileCLIPath: "profilecli",
			ProfileService: "pgo-app",
			RPS:            10,
			RPSInput:       "10",
			Warmup:         10 * time.Second,
			Duration:       60 * time.Second,
			Timeout:        5 * time.Second,
			OutputDir:      ".local/performance/login",
			VegetaPath:     "vegeta",
		},
		APIURL:      "http://127.0.0.1:20000",
		Users:       100,
		Concurrency: 10,
	}
}

func runLoginInteractive() {
	config, err := getInteractiveLoginConfig()
	if err != nil {
		pthird.Interact.Errorf("Invalid performance parameter: %v", err)
		return
	}
	pthird.Interact.Infof("Starting login performance test; artifacts: %s", config.OutputDir)
	runner := newLoadRunner(config.Timeout)
	if err = runLoginPlan(context.Background(), runner, config, os.Stdout, os.Stderr); err != nil {
		pthird.Interact.Errorf("Login performance test failed: %v", err)
	}
}

func getInteractiveLoginConfig() (loginConfig, error) {
	config := defaultLoginConfig()
	cachePath := pconfig.GetDefaultCachePath()
	const cachePrefix = "client.performance.login."
	config.APIURL = pclient.GetCachedParam(cachePath, cachePrefix+"api", "API address", config.APIURL)
	config.PprofURL = pclient.GetCachedParam(cachePath, cachePrefix+"pprof", "pprof address", config.PprofURL)
	config.PyroscopeURL = pclient.GetCachedParam(cachePath, cachePrefix+"pyroscope", "Pyroscope address", config.PyroscopeURL)
	config.GrafanaURL = pclient.GetCachedParam(cachePath, cachePrefix+"grafana", "Grafana address", config.GrafanaURL)
	config.ProfileSource = pclient.GetCachedParam(cachePath, cachePrefix+"profile-source", "profile source (pyroscope/pprof)", config.ProfileSource)
	config.ProfileCLIPath = pclient.GetCachedParam(cachePath, cachePrefix+"profilecli", "profilecli executable", config.ProfileCLIPath)
	config.ProfileService = pclient.GetCachedParam(cachePath, cachePrefix+"profile-service", "Pyroscope service name", config.ProfileService)
	config.OutputDir = pclient.GetCachedParam(cachePath, cachePrefix+"output", "artifact output directory", config.OutputDir)
	config.VegetaPath = pclient.GetCachedParam(cachePath, cachePrefix+"vegeta", "Vegeta executable", config.VegetaPath)
	var err error
	config.Users, err = getInteractiveInt(cachePath, cachePrefix+"users", "number of users", config.Users)
	if err != nil {
		return loginConfig{}, err
	}
	config.Concurrency, err = getInteractiveInt(cachePath, cachePrefix+"concurrency", "user preparation concurrency", config.Concurrency)
	if err != nil {
		return loginConfig{}, err
	}
	config.RPSInput = pclient.GetCachedParam(cachePath, cachePrefix+"rps", "login requests per second (positive integer or auto)", config.RPSInput)
	config.Warmup, err = getInteractiveDuration(cachePath, cachePrefix+"warmup", "warmup duration", config.Warmup)
	if err != nil {
		return loginConfig{}, err
	}
	config.Duration, err = getInteractiveDuration(cachePath, cachePrefix+"duration", "measured load duration", config.Duration)
	if err != nil {
		return loginConfig{}, err
	}
	config.Timeout, err = getInteractiveDuration(cachePath, cachePrefix+"timeout", "request timeout", config.Timeout)
	if err != nil {
		return loginConfig{}, err
	}
	return config, validateLoginConfig(config)
}

func getInteractiveInt(cachePath, key, prompt string, defaultValue int) (int, error) {
	value := pclient.GetCachedParam(cachePath, key, prompt, strconv.Itoa(defaultValue))
	parsedValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s %q: %w", prompt, value, err)
	}
	return parsedValue, nil
}

func getInteractiveDuration(cachePath, key, prompt string, defaultValue time.Duration) (time.Duration, error) {
	value := pclient.GetCachedParam(cachePath, key, prompt, defaultValue.String())
	parsedValue, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s %q: %w", prompt, value, err)
	}
	return parsedValue, nil
}

func newLoadRunner(timeout time.Duration) *loadRunner {
	clientTimeout := timeout + 75*time.Second
	return &loadRunner{
		commandRunner:   execRunner{},
		httpClient:      &http.Client{Timeout: clientTimeout},
		checkVegeta:     checkVegetaVersion,
		checkProfileCLI: checkProfileCLIVersion,
		now:             time.Now,
	}
}

func (runner *loadRunner) runLoginStage(ctx context.Context, config loginConfig, stdout, stderr io.Writer) (runErr error) {
	if err := validateLoginConfig(config); err != nil {
		return err
	}
	scenario := newLoginScenario(config, runner.now, stdout)
	return runner.runScenarioStage(ctx, config, scenario, stdout, stderr)
}

func (runner *loadRunner) runScenarioStage(
	ctx context.Context,
	config loginConfig,
	scenario performanceScenario,
	stdout io.Writer,
	stderr io.Writer,
) (runErr error) {
	if err := validateLoadConfig(config.loadConfig); err != nil {
		return err
	}
	if err := runner.checkVegeta(config.VegetaPath); err != nil {
		return err
	}
	if config.ProfileSource == profileSourcePyroscope {
		if err := runner.checkProfileCLI(config.ProfileCLIPath); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(config.OutputDir, 0o700); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(config.OutputDir, runFileName), config); err != nil {
		return err
	}

	targetPath, cleanup, err := scenario.Prepare(ctx, config.OutputDir)
	if cleanup != nil {
		defer func() {
			runErr = errors.Join(runErr, cleanup())
		}()
	}
	if err != nil {
		return err
	}
	return runner.runLoadStage(ctx, config.loadConfig, targetPath, stdout, stderr)
}

func (runner *loadRunner) runLoadStage(ctx context.Context, config loadConfig, targetPath string, stdout, stderr io.Writer) error {
	if config.Warmup > 0 {
		fmt.Fprintf(stdout, "[4/7] warming up for %s at %d RPS\n", config.Warmup, config.RPS)
		if err := runner.runAttack(ctx, config, targetPath, config.Warmup, io.Discard, stderr); err != nil {
			return fmt.Errorf("Vegeta warmup: %w", err)
		}
	}

	fmt.Fprintf(stdout, "[5/7] running measured load for %s at %d RPS and collecting diagnostics\n", config.Duration, config.RPS)
	printObservabilityGuidance(stdout, config)
	metricsBeforePath := filepath.Join(config.OutputDir, metricsBeforeFileName)
	metricsAfterPath := filepath.Join(config.OutputDir, metricsAfterFileName)
	if err := runner.download(metricsURL(config.PprofURL), metricsBeforePath); err != nil {
		return fmt.Errorf("collect metrics before load: %w", err)
	}
	resultPath := filepath.Join(config.OutputDir, vegetaResultsFileName)
	resultFile, err := os.OpenFile(resultPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	attackErrChannel := make(chan error, 1)
	go func() {
		attackErrChannel <- runner.runAttack(ctx, config, targetPath, config.Duration, resultFile, stderr)
	}()
	profileStart := runner.now().UTC()
	profileErr := runner.collectProfilesDuringLoad(ctx, config)
	attackErr := <-attackErrChannel
	profileEnd := runner.now().UTC()
	if config.ProfileSource == profileSourcePyroscope {
		profileErr = errors.Join(profileErr, runner.collectPyroscopeProfiles(ctx, config, profileStart, profileEnd, stderr))
	}
	closeErr := resultFile.Close()
	if err = runner.download(metricsURL(config.PprofURL), metricsAfterPath); err != nil {
		profileErr = errors.Join(profileErr, fmt.Errorf("collect metrics after load: %w", err))
	}
	if err = errors.Join(attackErr, closeErr, profileErr); err != nil {
		return fmt.Errorf("measured load: %w", err)
	}

	fmt.Fprintln(stdout, "[6/7] generating Vegeta and metrics reports")
	reportPath := filepath.Join(config.OutputDir, vegetaReportFileName)
	reportFile, err := os.OpenFile(reportPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	// Vegeta remains an external process for now. If installation or process control becomes
	// a recurring problem, consider replacing this boundary with the Vegeta Go package.
	err = runner.commandRunner.Run(ctx, config.VegetaPath, []string{"report", "-type=text", resultPath}, reportFile, stderr)
	closeErr = reportFile.Close()
	if err = errors.Join(err, closeErr); err != nil {
		return fmt.Errorf("Vegeta report: %w", err)
	}
	summaryPath := filepath.Join(config.OutputDir, summaryFileName)
	if err = writeSummary(summaryPath, config, reportPath, metricsBeforePath, metricsAfterPath); err != nil {
		return err
	}
	printArtifacts(stdout, config.OutputDir)
	return nil
}

func (runner *loadRunner) runAttack(
	ctx context.Context,
	config loadConfig,
	targetPath string,
	duration time.Duration,
	stdout io.Writer,
	stderr io.Writer,
) error {
	argumentList := []string{
		"attack",
		"-format=json",
		"-targets=" + targetPath,
		fmt.Sprintf("-rate=%d/s", config.RPS),
		"-duration=" + duration.String(),
		"-timeout=" + config.Timeout.String(),
	}
	// Vegeta remains an external process for now. If installation or process control becomes
	// a recurring problem, consider replacing this boundary with the Vegeta Go package.
	return runner.commandRunner.Run(ctx, config.VegetaPath, argumentList, stdout, stderr)
}

func (runner *loadRunner) collectProfilesDuringLoad(ctx context.Context, config loadConfig) error {
	cpuSeconds := int(config.Duration / time.Second)
	if cpuSeconds > 30 {
		cpuSeconds = 30
	}
	if cpuSeconds < 1 {
		cpuSeconds = 1
	}
	var waitGroup sync.WaitGroup
	errorChannel := make(chan error, 4)
	if config.ProfileSource == profileSourcePprof {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			profileURL := joinPprofURL(config.PprofURL, fmt.Sprintf("profile?seconds=%d", cpuSeconds))
			errorChannel <- runner.download(profileURL, filepath.Join(config.OutputDir, cpuProfileFileName))
		}()
	}
	leaseSeconds := int(config.Duration/time.Second) + 1
	if leaseSeconds > 60 {
		leaseSeconds = 60
	}
	if leaseSeconds < 1 {
		leaseSeconds = 1
	}
	activateURL := joinPprofURL(config.PprofURL, fmt.Sprintf("runtime?seconds=%d&profiles=goroutine,block,mutex", leaseSeconds))
	if err := runner.post(activateURL); err != nil {
		return fmt.Errorf("activate runtime profiles: %w", err)
	}
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		timer := time.NewTimer(config.Duration / 2)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			errorChannel <- ctx.Err()
			return
		case <-timer.C:
		}
		var heapErr error
		if config.ProfileSource == profileSourcePprof {
			heapErr = runner.download(joinPprofURL(config.PprofURL, "heap"), filepath.Join(config.OutputDir, heapProfileFileName))
		}
		goroutineErr := runner.download(joinPprofURL(config.PprofURL, "goroutine"), filepath.Join(config.OutputDir, goroutineProfileFileName))
		blockErr := runner.download(joinPprofURL(config.PprofURL, "block"), filepath.Join(config.OutputDir, blockProfileFileName))
		mutexErr := runner.download(joinPprofURL(config.PprofURL, "mutex"), filepath.Join(config.OutputDir, mutexProfileFileName))
		errorChannel <- errors.Join(heapErr, goroutineErr, blockErr, mutexErr)
	}()
	if config.RuntimeTrace {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			traceSeconds := cpuSeconds
			if traceSeconds > 10 {
				traceSeconds = 10
			}
			traceURL := joinPprofURL(config.PprofURL, fmt.Sprintf("runtime-trace?seconds=%d", traceSeconds))
			errorChannel <- runner.download(traceURL, filepath.Join(config.OutputDir, runtimeTraceFileName))
		}()
	}
	waitGroup.Wait()
	close(errorChannel)
	var errorList []error
	for err := range errorChannel {
		if err != nil {
			errorList = append(errorList, err)
		}
	}
	return errors.Join(errorList...)
}

func (runner *loadRunner) collectPyroscopeProfiles(
	ctx context.Context,
	config loadConfig,
	start time.Time,
	end time.Time,
	stderr io.Writer,
) error {
	profileList := []struct {
		profileType string
		output      string
	}{
		{profileType: "process_cpu:cpu:nanoseconds:cpu:nanoseconds", output: cpuProfileFileName},
		{profileType: "memory:inuse_space:bytes:space:bytes", output: heapProfileFileName},
	}
	var errorList []error
	for _, profile := range profileList {
		argumentList := []string{
			"query", "profile",
			"--url=" + normalizeHTTPURL(config.PyroscopeURL),
			"--profile-type=" + profile.profileType,
			fmt.Sprintf("--query={service_name=%q}", config.ProfileService),
			"--from=" + strconv.FormatInt(start.Unix(), 10),
			"--to=" + strconv.FormatInt(end.Unix(), 10),
			"--output=pprof=" + filepath.Join(config.OutputDir, profile.output),
		}
		if err := runner.commandRunner.Run(ctx, config.ProfileCLIPath, argumentList, io.Discard, stderr); err != nil {
			errorList = append(errorList, fmt.Errorf("export %s from Pyroscope: %w", profile.output, err))
			continue
		}
		profileInfo, err := os.Stat(filepath.Join(config.OutputDir, profile.output))
		if err != nil || profileInfo.Size() == 0 {
			errorList = append(errorList, fmt.Errorf("export %s from Pyroscope produced no profile", profile.output))
		}
	}
	return errors.Join(errorList...)
}

func (runner *loadRunner) post(address string) error {
	request, err := http.NewRequest(http.MethodPost, address, nil)
	if err != nil {
		return err
	}
	response, err := runner.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("POST %s returned %s: %s", address, response.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func (runner *loadRunner) download(address, output string) error {
	request, err := http.NewRequest(http.MethodGet, address, nil)
	if err != nil {
		return err
	}
	response, err := runner.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("GET %s returned %s: %s", address, response.Status, strings.TrimSpace(string(body)))
	}
	file, err := os.OpenFile(output, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(file, response.Body)
	return errors.Join(copyErr, file.Close())
}

func validateLoginConfig(config loginConfig) error {
	if config.Users <= 0 || config.Concurrency <= 0 {
		return errors.New("users and concurrency must be positive")
	}
	if _, err := url.ParseRequestURI(normalizeHTTPURL(config.APIURL)); err != nil {
		return fmt.Errorf("invalid API URL: %w", err)
	}
	return validateLoadConfig(config.loadConfig)
}

func validateLoadConfig(config loadConfig) error {
	if config.RPS <= 0 {
		return errors.New("rps must be positive")
	}
	if config.Duration <= 0 || config.Timeout <= 0 || config.Warmup < 0 {
		return errors.New("duration and timeout must be positive; warmup must not be negative")
	}
	if strings.TrimSpace(config.OutputDir) == "" {
		return errors.New("output directory is required")
	}
	if _, err := url.ParseRequestURI(normalizeHTTPURL(config.PprofURL)); err != nil {
		return fmt.Errorf("invalid pprof URL: %w", err)
	}
	if _, err := url.ParseRequestURI(normalizeHTTPURL(config.GrafanaURL)); err != nil {
		return fmt.Errorf("invalid Grafana URL: %w", err)
	}
	if config.ProfileSource != profileSourcePyroscope && config.ProfileSource != profileSourcePprof {
		return fmt.Errorf("profile source must be %q or %q", profileSourcePyroscope, profileSourcePprof)
	}
	if config.ProfileSource == profileSourcePyroscope {
		if _, err := url.ParseRequestURI(normalizeHTTPURL(config.PyroscopeURL)); err != nil {
			return fmt.Errorf("invalid Pyroscope URL: %w", err)
		}
		if strings.TrimSpace(config.ProfileCLIPath) == "" || strings.TrimSpace(config.ProfileService) == "" {
			return errors.New("profilecli path and profile service are required for Pyroscope")
		}
	}
	return nil
}

func checkProfileCLIVersion(path string) error {
	resolvedPath, err := exec.LookPath(path)
	if err != nil {
		return fmt.Errorf("find profilecli: %w", err)
	}
	command := exec.Command(resolvedPath, "--version")
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("read profilecli version: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if !strings.Contains(string(output), expectedProfileCLIVersion) {
		return fmt.Errorf("profilecli %s is required, found %s", expectedProfileCLIVersion, strings.TrimSpace(string(output)))
	}
	return nil
}

func printObservabilityGuidance(writer io.Writer, config loadConfig) {
	fmt.Fprintf(writer, "Observability: Grafana %s, Pyroscope %s\n", normalizeHTTPURL(config.GrafanaURL), normalizeHTTPURL(config.PyroscopeURL))
	fmt.Fprintf(writer, "Runtime profiles: POST %s\n", joinPprofURL(config.PprofURL, "runtime?seconds=60&profiles=goroutine,block,mutex"))
	fmt.Fprintf(writer, "Runtime trace: GET %s\n", joinPprofURL(config.PprofURL, "runtime-trace?seconds=10"))
}

func checkVegetaVersion(path string) error {
	resolvedPath, err := exec.LookPath(path)
	if err != nil {
		return fmt.Errorf("find Vegeta: %w", err)
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

func normalizeHTTPURL(address string) string {
	address = strings.TrimSpace(address)
	if !strings.Contains(address, "://") {
		return "http://" + address
	}
	return address
}

func joinPprofURL(baseURL, path string) string {
	baseURL = strings.TrimRight(normalizeHTTPURL(baseURL), "/")
	if strings.HasSuffix(baseURL, "/debug/pprof") {
		return baseURL + "/" + path
	}
	return baseURL + "/debug/pprof/" + path
}

func metricsURL(pprofURL string) string {
	parsedURL, err := url.Parse(normalizeHTTPURL(pprofURL))
	if err != nil {
		return strings.TrimRight(pprofURL, "/") + "/metrics"
	}
	parsedURL.Path = "/metrics"
	parsedURL.RawQuery = ""
	return parsedURL.String()
}

func writeJSON(path string, value any) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(content, '\n'), 0o600)
}

func writeSummary(path string, config loadConfig, reportPath, beforePath, afterPath string) error {
	report, err := os.ReadFile(reportPath)
	if err != nil {
		return err
	}
	before, err := readMetricSnapshot(beforePath)
	if err != nil {
		return err
	}
	after, err := readMetricSnapshot(afterPath)
	if err != nil {
		return err
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "# Login performance result\n\n")
	fmt.Fprintf(&builder, "- Users: %d\n- Target rate: %d requests/s\n- Warmup: %s\n- Measured duration: %s\n- Request timeout: %s\n\n",
		config.Users, config.RPS, config.Warmup, config.Duration, config.Timeout)
	fmt.Fprintf(&builder, "- Profile source: %s\n- Profile service: %s\n- Grafana: %s\n- Pyroscope: %s\n\n",
		config.ProfileSource, config.ProfileService, normalizeHTTPURL(config.GrafanaURL), normalizeHTTPURL(config.PyroscopeURL))
	builder.WriteString("## Vegeta report\n\n```text\n")
	builder.Write(report)
	builder.WriteString("```\n\n")
	builder.WriteString("- Throughput shows completed requests per second.\n")
	builder.WriteString("- Success shows the percentage of responses accepted by Vegeta.\n")
	builder.WriteString("- Latencies report request delay distribution; P95 and P99 expose slow-tail behavior.\n\n")
	builder.WriteString("## Service metrics during measured load\n\n")
	builder.WriteString("Counter values below are calculated from the after-minus-before difference. Runtime and connection values show the boundary state.\n\n")
	loginCountKey := `pgo_http_requests_total{operation="/api.User/Login",result="ok"}`
	loginDurationSumKey := `pgo_http_request_duration_seconds_sum{operation="/api.User/Login",result="ok"}`
	loginDurationCountKey := `pgo_http_request_duration_seconds_count{operation="/api.User/Login",result="ok"}`
	querySumKey := `pgo_db_query_duration_seconds_sum{operation="query",result="ok"}`
	queryCountKey := `pgo_db_query_duration_seconds_count{operation="query",result="ok"}`
	fmt.Fprintf(&builder, "- Successful HTTP login requests: %.0f\n", metricDelta(before, after, loginCountKey))
	fmt.Fprintf(&builder, "- Average server-side login duration: %s\n", formatAverageDuration(before, after, loginDurationSumKey, loginDurationCountKey))
	fmt.Fprintf(&builder, "- Successful database queries: %.0f\n", metricDelta(before, after, queryCountKey))
	fmt.Fprintf(&builder, "- Average database query duration: %s\n", formatAverageDuration(before, after, querySumKey, queryCountKey))
	fmt.Fprintf(&builder, "- Database connection waits: %.0f, total wait %s\n",
		metricDelta(before, after, "pgo_db_connection_wait_total"),
		formatSeconds(metricDelta(before, after, "pgo_db_connection_wait_duration_seconds_total")))
	fmt.Fprintf(&builder, "- Database connections after load: open %.0f, in use %.0f, idle %.0f\n",
		after["pgo_db_connections_open"], after["pgo_db_connections_in_use"], after["pgo_db_connections_idle"])
	fmt.Fprintf(&builder, "- Process CPU consumed during load: %s\n", formatSeconds(metricDelta(before, after, "process_cpu_seconds_total")))
	fmt.Fprintf(&builder, "- Resident memory after load: %.2f MiB\n", after["process_resident_memory_bytes"]/(1024*1024))
	fmt.Fprintf(&builder, "- Go allocated heap boundary: %.2f MiB -> %.2f MiB\n",
		before["go_memstats_alloc_bytes"]/(1024*1024), after["go_memstats_alloc_bytes"]/(1024*1024))
	fmt.Fprintf(&builder, "- Goroutines boundary: %.0f -> %.0f\n\n", before["go_goroutines"], after["go_goroutines"])
	builder.WriteString("### Metric meaning\n\n")
	builder.WriteString("- `pgo_http_*`: HTTP request volume, errors, and latency.\n")
	builder.WriteString("- `pgo_db_query_*`: database query volume and latency.\n")
	builder.WriteString("- `pgo_db_connections_*`: open, active, idle, and waiting database connections.\n")
	builder.WriteString("- `go_*`: Go runtime memory, garbage collection, and goroutine state.\n")
	builder.WriteString("- `process_*`: operating-system resource usage for the service process.\n")
	return os.WriteFile(path, []byte(builder.String()), 0o600)
}

func readMetricSnapshot(path string) (map[string]float64, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	metricMap := make(map[string]float64)
	for _, line := range strings.Split(string(content), "\n") {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		fieldList := strings.Fields(line)
		if len(fieldList) != 2 {
			continue
		}
		value, parseErr := strconv.ParseFloat(fieldList[1], 64)
		if parseErr == nil {
			metricMap[fieldList[0]] = value
		}
	}
	return metricMap, nil
}

func metricDelta(before, after map[string]float64, key string) float64 {
	return after[key] - before[key]
}

func formatAverageDuration(before, after map[string]float64, sumKey, countKey string) string {
	count := metricDelta(before, after, countKey)
	if count <= 0 {
		return "not available"
	}
	return formatSeconds(metricDelta(before, after, sumKey) / count)
}

func formatSeconds(seconds float64) string {
	return (time.Duration(seconds * float64(time.Second))).Round(time.Microsecond).String()
}

func printArtifacts(writer io.Writer, outputDir string) {
	artifactList := []artifact{
		{Path: runFileName, Description: "input parameters for reproducing this run"},
		{Path: usersFileName, Description: "batch manifest used for verification and precise cleanup; contains tokens"},
		{Path: loginTargetsFileName, Description: "Vegeta login request definitions"},
		{Path: vegetaResultsFileName, Description: "raw Vegeta request results"},
		{Path: vegetaReportFileName, Description: "human-readable throughput, success, and latency report"},
		{Path: metricsAfterFileName, Description: "Prometheus metrics immediately after measured load"},
		{Path: metricsBeforeFileName, Description: "Prometheus metrics immediately before measured load"},
		{Path: cpuProfileFileName, Description: "CPU samples captured during measured load"},
		{Path: goroutineProfileFileName, Description: "goroutine snapshot captured during measured load"},
		{Path: heapProfileFileName, Description: "heap snapshot captured during measured load"},
		{Path: blockProfileFileName, Description: "blocking profile captured during measured load"},
		{Path: mutexProfileFileName, Description: "mutex contention profile captured during measured load"},
		{Path: summaryFileName, Description: "formatted result summary and metric explanations"},
	}
	if _, err := os.Stat(filepath.Join(outputDir, runtimeTraceFileName)); err == nil {
		artifactList = append(artifactList, artifact{Path: runtimeTraceFileName, Description: "optional Go runtime trace"})
	}
	fmt.Fprintf(writer, "Output directory: %s\n", outputDir)
	fmt.Fprintln(writer, "Artifacts:")
	for _, item := range artifactList {
		fmt.Fprintf(writer, "- %s: %s\n", item.Path, item.Description)
	}
}
