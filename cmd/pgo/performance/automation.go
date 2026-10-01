package performance

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	autoPlanFileName    = "00-auto-run.json"
	autoResultFileName  = "40-auto-results.json"
	autoSummaryFileName = "41-auto-summary.md"
)

var autoRPSList = []int{10, 25, 50, 100, 200, 500}

type stageResult struct {
	RPS                 int     `json:"rps"`
	OutputDir           string  `json:"outputDir"`
	Requests            int     `json:"requests"`
	Throughput          float64 `json:"throughput"`
	Success             float64 `json:"success"`
	P50                 string  `json:"p50"`
	P95                 string  `json:"p95"`
	P99                 string  `json:"p99"`
	ServerLoginDuration string  `json:"serverLoginDuration"`
	DBQueryDuration     string  `json:"dbQueryDuration"`
	DBConnectionWaits   float64 `json:"dbConnectionWaits"`
	CPUTime             string  `json:"cpuTime"`
	ResidentMemoryMiB   float64 `json:"residentMemoryMiB"`
	GoroutinesBefore    float64 `json:"goroutinesBefore"`
	GoroutinesAfter     float64 `json:"goroutinesAfter"`
	Error               string  `json:"error,omitempty"`
}

type autoRunRecord struct {
	Mode      string        `json:"mode"`
	RPSList   []int         `json:"rpsList"`
	StageList []stageResult `json:"stageList"`
}

func runLoginPlan(
	ctx context.Context,
	runner *loadRunner,
	config loginConfig,
	stdout io.Writer,
	stderr io.Writer,
) error {
	rpsList, automatic, err := resolveRPSList(config.RPSInput, config.RPS)
	if err != nil {
		return err
	}
	if !automatic {
		config.RPS = rpsList[0]
		return runner.runLoginStage(ctx, config, stdout, stderr)
	}
	return runAutomaticLogin(ctx, runner, config, rpsList, stdout, stderr)
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
		return nil, false, fmt.Errorf("rps must be a positive integer or auto, got %q", input)
	}
	return []int{rps}, false, nil
}

func runAutomaticLogin(
	ctx context.Context,
	runner *loadRunner,
	config loginConfig,
	rpsList []int,
	stdout io.Writer,
	stderr io.Writer,
) error {
	rootOutputDir := config.OutputDir
	if err := os.MkdirAll(rootOutputDir, 0o700); err != nil {
		return err
	}
	record := autoRunRecord{Mode: "auto", RPSList: append([]int(nil), rpsList...)}
	if err := writeJSON(filepath.Join(rootOutputDir, autoPlanFileName), record); err != nil {
		return err
	}

	var runErr error
	for index, rps := range rpsList {
		if err := ctx.Err(); err != nil {
			runErr = err
			break
		}
		stageConfig := config
		stageConfig.RPS = rps
		stageConfig.RPSInput = strconv.Itoa(rps)
		stageConfig.OutputDir = filepath.Join(rootOutputDir, fmt.Sprintf("rps-%03d", rps))
		fmt.Fprintf(stdout, "\n[auto %d/%d] starting %d RPS stage\n", index+1, len(rpsList), rps)
		stageErr := runner.runLoginStage(ctx, stageConfig, stdout, stderr)
		result, resultErr := readStageResult(stageConfig)
		stageErr = errors.Join(stageErr, resultErr)
		if stageErr != nil {
			result.Error = stageErr.Error()
		}
		record.StageList = append(record.StageList, result)
		if err := writeAutoResults(rootOutputDir, record); err != nil {
			return errors.Join(stageErr, err)
		}
		if stageErr != nil {
			runErr = fmt.Errorf("automatic login load stopped at %d RPS: %w", rps, stageErr)
			break
		}
		if result.Success < 1 {
			runErr = fmt.Errorf("automatic login load stopped at %d RPS: success ratio %.2f%%", rps, result.Success*100)
			record.StageList[len(record.StageList)-1].Error = runErr.Error()
			if err := writeAutoResults(rootOutputDir, record); err != nil {
				return errors.Join(runErr, err)
			}
			break
		}
	}
	if err := writeAutoResults(rootOutputDir, record); err != nil {
		return errors.Join(runErr, err)
	}
	fmt.Fprintf(stdout, "\nAutomatic load summary: %s\n", filepath.Join(rootOutputDir, autoSummaryFileName))
	return runErr
}

func readStageResult(config loginConfig) (stageResult, error) {
	result := stageResult{RPS: config.RPS, OutputDir: config.OutputDir}
	content, err := os.ReadFile(filepath.Join(config.OutputDir, vegetaReportFileName))
	if err != nil {
		return result, err
	}
	for _, line := range strings.Split(string(content), "\n") {
		value := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(value, "Requests"):
			fieldList := strings.Split(value[strings.LastIndex(value, "]")+1:], ",")
			if len(fieldList) == 3 {
				result.Requests, _ = strconv.Atoi(strings.TrimSpace(fieldList[0]))
				result.Throughput, _ = strconv.ParseFloat(strings.TrimSpace(fieldList[2]), 64)
			}
		case strings.HasPrefix(value, "Latencies"):
			fieldList := strings.Split(value[strings.LastIndex(value, "]")+1:], ",")
			if len(fieldList) == 5 {
				result.P50 = strings.TrimSpace(fieldList[1])
				result.P95 = strings.TrimSpace(fieldList[2])
				result.P99 = strings.TrimSpace(fieldList[3])
			}
		case strings.HasPrefix(value, "Success"):
			percent := strings.TrimSpace(value[strings.LastIndex(value, "]")+1:])
			percent = strings.TrimSuffix(percent, "%")
			parsed, parseErr := strconv.ParseFloat(strings.TrimSpace(percent), 64)
			if parseErr == nil {
				result.Success = parsed / 100
			}
		}
	}
	if result.Requests == 0 {
		return result, errors.New("Vegeta report contains no requests")
	}
	before, err := readMetricSnapshot(filepath.Join(config.OutputDir, metricsBeforeFileName))
	if err != nil {
		return result, err
	}
	after, err := readMetricSnapshot(filepath.Join(config.OutputDir, metricsAfterFileName))
	if err != nil {
		return result, err
	}
	result.ServerLoginDuration = formatAverageDuration(before, after,
		`pgo_http_request_duration_seconds_sum{operation="/api.User/Login",result="ok"}`,
		`pgo_http_request_duration_seconds_count{operation="/api.User/Login",result="ok"}`)
	result.DBQueryDuration = formatAverageDuration(before, after,
		`pgo_db_query_duration_seconds_sum{operation="query",result="ok"}`,
		`pgo_db_query_duration_seconds_count{operation="query",result="ok"}`)
	result.DBConnectionWaits = metricDelta(before, after, "pgo_db_connection_wait_total")
	result.CPUTime = formatSeconds(metricDelta(before, after, "process_cpu_seconds_total"))
	result.ResidentMemoryMiB = after["process_resident_memory_bytes"] / (1024 * 1024)
	result.GoroutinesBefore = before["go_goroutines"]
	result.GoroutinesAfter = after["go_goroutines"]
	return result, nil
}

func writeAutoResults(outputDir string, record autoRunRecord) error {
	if err := writeJSON(filepath.Join(outputDir, autoResultFileName), record); err != nil {
		return err
	}
	var builder strings.Builder
	builder.WriteString("# Login automatic load result\n\n")
	builder.WriteString("Built-in RPS ladder: `10, 25, 50, 100, 200, 500`.\n\n")
	builder.WriteString("| RPS | Requests | Throughput | Success | P50 | P95 | P99 | Status |\n")
	builder.WriteString("| ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |\n")
	for _, result := range record.StageList {
		status := "passed"
		if result.Error != "" {
			status = "failed: " + strings.ReplaceAll(result.Error, "|", "\\|")
		}
		fmt.Fprintf(&builder, "| %d | %d | %.2f | %.2f%% | %s | %s | %s | %s |\n",
			result.RPS, result.Requests, result.Throughput, result.Success*100,
			result.P50, result.P95, result.P99, status)
	}
	builder.WriteString("\n## Service and runtime trend\n\n")
	builder.WriteString("| RPS | Server login | DB query | DB waits | CPU time | RSS MiB | Goroutines |\n")
	builder.WriteString("| ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	for _, result := range record.StageList {
		fmt.Fprintf(&builder, "| %d | %s | %s | %.0f | %s | %.2f | %.0f → %.0f |\n",
			result.RPS, result.ServerLoginDuration, result.DBQueryDuration, result.DBConnectionWaits,
			result.CPUTime, result.ResidentMemoryMiB, result.GoroutinesBefore, result.GoroutinesAfter)
	}
	builder.WriteString("\nEach stage directory contains its metrics, profiles, raw load result, and detailed summary.\n")
	return os.WriteFile(filepath.Join(outputDir, autoSummaryFileName), []byte(builder.String()), 0o600)
}
