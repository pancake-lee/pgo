package core

import (
	"context"
	"encoding/json"
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
	prepared bool
	cleaned  bool
}

func (preparer *fakePreparer) Prepare(_ context.Context, outputDir string,
) (string, func() error, error) {
	preparer.prepared = true
	targetPath := filepath.Join(outputDir, "targets.jsonl")
	err := os.WriteFile(targetPath, []byte("target"), 0o600)
	if err != nil {
		return "", nil, err
	}
	return targetPath, func() error {
		preparer.cleaned = true
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

// TestRunnerExecutesOnePreparedStage 验证准备、负载产物与清理生命周期。
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
		loadWindowFileName,
	}
	for _, name := range artifactList {
		_, err = os.Stat(filepath.Join(config.OutputDir, name))
		if err != nil {
			t.Errorf("artifact %s: %v", name, err)
		}
	}
	if !strings.Contains(output.String(), "outputDir="+config.OutputDir) {
		t.Fatalf("artifact log = %s", output.String())
	}
}

// TestRunnerWarmupAndWindow 验证场景预热及正式负载时间窗不包含准备清理。
func TestRunnerWarmupAndWindow(t *testing.T) {
	runner := NewRunner(klog.NewStdLogger(io.Discard))
	runner.checkVegeta = func(string) error { return nil }
	var durationList []string
	runner.execContext = func(
		ctx context.Context, output io.Writer, name string, args ...string,
	) (string, error) {
		if args[0] == "attack" {
			for _, arg := range args {
				if strings.HasPrefix(arg, "-duration=") {
					durationList = append(durationList, arg)
				}
			}
		}
		return fakeExecContext(ctx, output, name, args...)
	}
	config := Config{
		APIURL: "http://test", RPS: 10, Duration: 120 * time.Second,
		Warmup: 30 * time.Second, OutputDir: t.TempDir(),
	}
	before := time.Now().UTC()
	err := runner.Run(t.Context(), config, &fakePreparer{})
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(durationList) != "[-duration=30s -duration=2m0s]" {
		t.Fatalf("attack durations = %v", durationList)
	}
	content, err := os.ReadFile(filepath.Join(config.OutputDir, loadWindowFileName))
	if err != nil {
		t.Fatal(err)
	}
	var window loadWindow
	err = json.Unmarshal(content, &window)
	if err != nil || window.StartUTC.Before(before) ||
		window.EndUTC.Before(window.StartUTC) {
		t.Fatalf("window = %s, err = %v", content, err)
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
	options := AutomaticOptions{
		RPSList:      []int{10, 25, 50},
		SummaryTitle: "Test automatic load result",
	}
	err := RunAutomatic(t.Context(), runner, config, options, runStage)
	if err == nil || !strings.Contains(err.Error(), "25 RPS") {
		t.Fatalf("automatic error = %v", err)
	}
	if fmt.Sprint(visitedList) != "[10 25]" {
		t.Fatalf("visited stages = %v", visitedList)
	}
	artifactList := []string{
		autoPlanFileName,
		autoResultFileName,
		autoSummaryFileName,
	}
	for _, name := range artifactList {
		_, err = os.Stat(filepath.Join(config.OutputDir, name))
		if err != nil {
			t.Errorf("automatic artifact %s: %v", name, err)
		}
	}
	summaryPath := filepath.Join(config.OutputDir, autoSummaryFileName)
	summary, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(summary), "`10, 25, 50`") {
		t.Fatalf("summary does not describe the actual ladder: %s", summary)
	}
}

func TestRunAutomaticStopsAfterUnsuccessfulResponses(t *testing.T) {
	runner := NewRunner(klog.NewStdLogger(io.Discard))
	config := Config{
		APIURL:    "http://127.0.0.1:20000",
		RPS:       10,
		Duration:  time.Minute,
		OutputDir: t.TempDir(),
	}
	var visitedList []int
	runStage := func(_ context.Context, stageConfig Config) error {
		visitedList = append(visitedList, stageConfig.RPS)
		err := os.MkdirAll(stageConfig.OutputDir, 0o700)
		if err != nil {
			return err
		}
		success := "100.00"
		if stageConfig.RPS == 25 {
			success = "90.00"
		}
		report := fmt.Sprintf(
			"Requests [total, rate, throughput] "+
				"1, 1.00, 1.00\nSuccess [ratio] %s%%\n",
			success,
		)
		reportPath := filepath.Join(stageConfig.OutputDir, vegetaReportFileName)
		return os.WriteFile(reportPath, []byte(report), 0o600)
	}
	options := AutomaticOptions{
		RPSList:      []int{10, 25, 50},
		SummaryTitle: "Test automatic load result",
	}
	err := RunAutomatic(t.Context(), runner, config, options, runStage)
	if err == nil || !strings.Contains(err.Error(), "success ratio 90.00%") {
		t.Fatalf("automatic error = %v", err)
	}
	if fmt.Sprint(visitedList) != "[10 25]" {
		t.Fatalf("visited stages = %v", visitedList)
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
