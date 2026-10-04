package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	klog "github.com/go-kratos/kratos/v2/log"
)

type fakePreparer struct {
	prepared     bool
	cleaned      bool
	prepareCount int
	cleanupCount int
}

// fakePreparedLoader 验证场景编排复用准备与结果生命周期。
type fakePreparedLoader struct {
	fakePreparer
	loaded bool
}

// RunLoad 使用通用执行器运行已准备的目标，省略默认预热。
func (loader *fakePreparedLoader) RunLoad(ctx context.Context, runner *Runner,
	config Config, targetPath string,
) error {
	loader.loaded = true
	return runner.RunMeasured(ctx, config, targetPath)
}

// TestRunnerUsesPreparedLoader 验证多流场景不额外执行默认单流负载。
func TestRunnerUsesPreparedLoader(t *testing.T) {
	runner := NewRunner(klog.NewStdLogger(io.Discard))
	attacks := 0
	runner.execContext = func(ctx context.Context, writer io.Writer,
		path string, args ...string,
	) (string, error) {
		if args[0] == "attack" {
			attacks++
		}
		return fakeExecContext(ctx, writer, path, args...)
	}
	runner.checkVegeta = func(string) error { return nil }
	loader := &fakePreparedLoader{}
	config := Config{APIURL: "http://localhost:8080", RPS: 10,
		Duration: time.Second, OutputDir: t.TempDir()}
	err := runner.Run(t.Context(), config, loader)
	if err != nil {
		t.Fatal(err)
	}
	if !loader.loaded || !loader.prepared || !loader.cleaned || attacks != 1 {
		t.Fatalf("custom lifecycle failed; attacks=%d", attacks)
	}
}

func (preparer *fakePreparer) Prepare(_ context.Context, outputDir string,
) (string, func() error, error) {
	preparer.prepared = true
	preparer.prepareCount++
	targetPath := filepath.Join(outputDir, "targets.jsonl")
	err := os.WriteFile(targetPath, []byte("target"), 0o600)
	if err != nil {
		return "", nil, err
	}
	return targetPath, func() error {
		preparer.cleaned = true
		preparer.cleanupCount++
		return nil
	}, nil
}

func fakeExecContext(
	_ context.Context,
	output io.Writer,
	_ string,
	args ...string,
) (string, error) {
	if len(args) > 0 && args[0] == "attack" {
		_, _ = io.WriteString(output, "raw-result")
		return "", nil
	}
	report := "Requests [total, rate, throughput] " +
		"1, 1.00, 1.00\n" +
		"Success [ratio] 100.00%\n"
	_, _ = io.WriteString(output, report)
	return "", nil
}

func TestRunnerExecutesOnePreparedStage(t *testing.T) {
	var output strings.Builder
	runner := NewRunner(klog.NewStdLogger(&output))
	runner.execContext = fakeExecContext
	runner.checkVegeta = func(string) error { return nil }
	preparer := &fakePreparer{}
	config := Config{
		APIURL:    "http://127.0.0.1:20000",
		RPS:       5,
		Duration:  30 * time.Second,
		OutputDir: t.TempDir(),
	}
	err := runner.Run(t.Context(), config, preparer)
	if err != nil {
		t.Fatal(err)
	}
	if !preparer.prepared || !preparer.cleaned {
		t.Fatalf(
			"preparer lifecycle = prepared:%v cleaned:%v",
			preparer.prepared,
			preparer.cleaned,
		)
	}
	artifactList := []string{
		runFileName,
		vegetaResultsFileName,
		vegetaReportFileName,
	}
	for _, name := range artifactList {
		_, err = os.Stat(filepath.Join(config.OutputDir, name))
		if err != nil {
			t.Errorf("artifact %s: %v", name, err)
		}
	}
	if !strings.Contains(output.String(), "file="+filepath.Join(config.OutputDir, vegetaReportFileName)) ||
		strings.Contains(output.String(), vegetaResultsFileName) {
		t.Fatalf("artifact log = %s", output.String())
	}
}

func TestRunAutomaticReusesStageCallbackAndStopsOnFailure(t *testing.T) {
	runner := NewRunner(klog.NewStdLogger(io.Discard))
	config := Config{
		APIURL:    "http://127.0.0.1:20000",
		RPS:       10,
		Duration:  time.Minute,
		OutputDir: t.TempDir(),
	}
	var visitedList []int
	runStage := func(_ context.Context, stageConfig Config) error {
		if stageConfig.Duration != config.Duration {
			return fmt.Errorf("duration = %s", stageConfig.Duration)
		}
		visitedList = append(visitedList, stageConfig.RPS)
		if stageConfig.RPS == 25 {
			return errors.New("injected failure")
		}
		err := os.MkdirAll(stageConfig.OutputDir, 0o700)
		if err != nil {
			return err
		}
		report := "Requests [total, rate, throughput] " +
			"1, 1.00, 1.00\n" +
			"Success [ratio] 100.00%\n"
		reportPath := filepath.Join(stageConfig.OutputDir, vegetaReportFileName)
		return os.WriteFile(reportPath, []byte(report), 0o600)
	}
	rpsList := []int{10, 25, 50}
	err := RunAutomatic(t.Context(), runner, config, rpsList, runStage)
	if err == nil || !strings.Contains(err.Error(), "25 RPS") {
		t.Fatalf("automatic error = %v", err)
	}
	if fmt.Sprint(visitedList) != "[10 25]" {
		t.Fatalf("visited stages = %v", visitedList)
	}
	plan, err := os.ReadFile(filepath.Join(config.OutputDir, autoPlanFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(plan), "25") {
		t.Fatal("missing ladder input")
	}
	for _, name := range []string{"40-auto-results.json", "41-auto-summary.md"} {
		if _, err = os.Stat(filepath.Join(config.OutputDir, name)); !os.IsNotExist(err) {
			t.Fatalf("unexpected cross-stage report %s", name)
		}
	}
}

// TestRunAutomaticStopsAfterUnsuccessfulResponses 验证单档错误停止后续阶梯。
func TestRunAutomaticStopsAfterUnsuccessfulResponses(t *testing.T) {
	runner := NewRunner(klog.NewStdLogger(io.Discard))
	runner.checkVegeta = func(string) error { return nil }
	runner.execContext = func(ctx context.Context, writer io.Writer, path string,
		args ...string,
	) (string, error) {
		if args[0] == "report" && strings.Contains(args[len(args)-1], "rps-025") {
			_, err := io.WriteString(writer, "Requests [total, rate, throughput] 1, 1.00, 1.00\nSuccess [ratio] 90.00%\n")
			return "", err
		}
		return fakeExecContext(ctx, writer, path, args...)
	}
	loader := &fakePreparedLoader{}
	config := Config{APIURL: "http://localhost:8080", RPS: 10,
		Duration: time.Second, OutputDir: t.TempDir()}
	err := runner.RunAutomatic(t.Context(), config, loader, []int{10, 25, 50})
	if err == nil || !strings.Contains(err.Error(), "success ratio 90.00%") {
		t.Fatalf("automatic error = %v", err)
	}
	if loader.prepareCount != 1 || loader.cleanupCount != 1 {
		t.Fatal("automatic data lifecycle did not run once")
	}
	if _, err = os.Stat(filepath.Join(config.OutputDir, "rps-050")); !os.IsNotExist(err) {
		t.Fatal("automatic load continued after unsuccessful responses")
	}
}

// TestAutomaticPreparedLoaderReusesData 验证多窗口场景不需要根目录报告且各档复用目标。
func TestAutomaticPreparedLoaderReusesData(t *testing.T) {
	runner := NewRunner(klog.NewStdLogger(io.Discard))
	runner.checkVegeta = func(string) error { return nil }
	targetMap := make(map[string]bool)
	runner.execContext = func(ctx context.Context, writer io.Writer, path string,
		args ...string,
	) (string, error) {
		if args[0] == "attack" {
			for _, arg := range args {
				if strings.HasPrefix(arg, "-targets=") {
					targetMap[arg] = true
				}
			}
		}
		return fakeExecContext(ctx, writer, path, args...)
	}
	loader := &fakePreparedLoader{}
	config := Config{APIURL: "http://localhost:8080", RPS: 10,
		Duration: time.Second, OutputDir: t.TempDir()}
	err := runner.RunAutomatic(t.Context(), config, loader, []int{10, 25, 50})
	if err != nil {
		t.Fatal(err)
	}
	if loader.prepareCount != 1 || loader.cleanupCount != 1 || len(targetMap) != 1 {
		t.Fatal("automatic stages did not reuse one prepared batch")
	}
	for _, rps := range []int{10, 25, 50} {
		path := filepath.Join(config.OutputDir, fmt.Sprintf("rps-%03d", rps), vegetaReportFileName)
		if _, err = os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = os.Stat(filepath.Join(config.OutputDir, vegetaReportFileName)); !os.IsNotExist(err) {
		t.Fatal("unexpected root load report")
	}
}

func TestFixedDependencyErrorsIncludeInstallGuidance(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing-tool")
	err := checkVegetaVersion(missingPath)
	containsGuidance := err != nil && strings.Contains(
		err.Error(),
		"go install github.com/tsenart/vegeta/v12@v12.13.0",
	)
	if !containsGuidance {
		t.Fatalf("Vegeta error = %v", err)
	}
}
