package performance

import (
	"context"
	"errors"
	"fmt"
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

// stageResult 保存单个压力档位的负载结果。
type stageResult struct {
	RPS        int     `json:"rps"`
	OutputDir  string  `json:"outputDir"`
	Requests   int     `json:"requests"`
	Throughput float64 `json:"throughput"`
	Success    float64 `json:"success"`
	P50        string  `json:"p50"`
	P95        string  `json:"p95"`
	P99        string  `json:"p99"`
	Error      string  `json:"error,omitempty"`
}

// autoRunRecord 保存自动阶梯计划及各档执行结果。
type autoRunRecord struct {
	Mode      string        `json:"mode"`
	RPSList   []int         `json:"rpsList"`
	StageList []stageResult `json:"stageList"`
}

// runLoginPlan 根据 RPS 模式执行单档或自动阶梯压测。
func runLoginPlan(
	ctx context.Context,
	runner *loadRunner,
	config loadConfig,
) error {
	rpsList, automatic, err := resolveRPSList(config.RPSInput, config.RPS)
	if err != nil {
		return err
	}
	if !automatic {
		config.RPS = rpsList[0]
		return runner.runLoginStage(ctx, config)
	}
	return runAutomaticLogin(ctx, runner, config, rpsList)
}

// resolveRPSList 将用户输入解析为单档或内置压力阶梯。
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

// runAutomaticLogin 按固定档位串行执行登录压测并在失败时停止。
func runAutomaticLogin(
	ctx context.Context,
	runner *loadRunner,
	config loadConfig,
	rpsList []int,
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
		runner.info("starting automatic load stage", "stage", index+1, "stages", len(rpsList), "rps", rps)
		stageErr := runner.runLoginStage(ctx, stageConfig)
		if stageErr != nil {
			result := stageResult{RPS: rps, OutputDir: stageConfig.OutputDir, Error: stageErr.Error()}
			record.StageList = append(record.StageList, result)
			if err := writeAutoResults(rootOutputDir, record); err != nil {
				return errors.Join(stageErr, err)
			}
			runErr = fmt.Errorf("automatic login load stopped at %d RPS: %w", rps, stageErr)
			break
		}
		result, stageErr := readStageResult(stageConfig)
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
	runner.info("automatic load summary", "file", filepath.Join(rootOutputDir, autoSummaryFileName))
	return runErr
}

// readStageResult 从单档产物中提取跨档比较所需的指标。
func readStageResult(config loadConfig) (stageResult, error) {
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
	return result, nil
}

// writeAutoResults 写入机器可读结果和跨档汇总报告。
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
	builder.WriteString("\nEach stage directory contains its load parameters, scenario data, raw Vegeta result, and text report.\n")
	return os.WriteFile(filepath.Join(outputDir, autoSummaryFileName), []byte(builder.String()), 0o600)
}
