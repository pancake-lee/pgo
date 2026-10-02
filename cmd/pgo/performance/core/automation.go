package core

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

// AutomaticOptions 保存本次自动升压档位和汇总标题。
type AutomaticOptions struct {
	RPSList      []int
	SummaryTitle string
}

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

type autoRunRecord struct {
	Mode      string        `json:"mode"`
	RPSList   []int         `json:"rpsList"`
	StageList []stageResult `json:"stageList"`
}

// RunAutomatic 按指定档位串行复用单档压测，失败或出现非成功响应时停止。
func RunAutomatic(
	ctx context.Context,
	runner *Runner,
	config Config,
	options AutomaticOptions,
	runStage func(context.Context, Config) error,
) error {
	if len(options.RPSList) == 0 {
		return errors.New("automatic RPS list is empty")
	}
	if strings.TrimSpace(options.SummaryTitle) == "" {
		return errors.New("automatic summary title is required")
	}

	rootOutputDir := config.OutputDir
	err := os.MkdirAll(rootOutputDir, 0o700)
	if err != nil {
		return err
	}

	record := autoRunRecord{
		Mode:    "auto",
		RPSList: append([]int(nil), options.RPSList...),
	}
	planPath := filepath.Join(rootOutputDir, autoPlanFileName)
	err = writeJSON(planPath, record)
	if err != nil {
		return err
	}

	var runErr error
	for index, rps := range options.RPSList {
		if rps <= 0 {
			runErr = fmt.Errorf(
				"automatic RPS must be positive, got %d",
				rps,
			)
			break
		}

		contextErr := ctx.Err()
		if contextErr != nil {
			runErr = contextErr
			break
		}

		stageConfig := config
		stageConfig.RPS = rps
		stageName := fmt.Sprintf("rps-%03d", rps)
		stageConfig.OutputDir = filepath.Join(rootOutputDir, stageName)
		runner.info(
			"starting automatic load stage",
			"stage",
			index+1,
			"stages",
			len(options.RPSList),
			"rps",
			rps,
		)

		stageErr := runStage(ctx, stageConfig)
		if stageErr != nil {
			result := stageResult{
				RPS:       rps,
				OutputDir: stageConfig.OutputDir,
				Error:     stageErr.Error(),
			}
			record.StageList = append(record.StageList, result)

			writeErr := writeAutoResults(rootOutputDir, record, options)
			if writeErr != nil {
				return errors.Join(stageErr, writeErr)
			}

			runErr = fmt.Errorf(
				"automatic load stopped at %d RPS: %w",
				rps,
				stageErr,
			)
			break
		}

		result, stageErr := readStageResult(stageConfig)
		if stageErr != nil {
			result.Error = stageErr.Error()
		}
		record.StageList = append(record.StageList, result)

		writeErr := writeAutoResults(rootOutputDir, record, options)
		if writeErr != nil {
			return errors.Join(stageErr, writeErr)
		}
		if stageErr != nil {
			runErr = fmt.Errorf(
				"automatic load stopped at %d RPS: %w",
				rps,
				stageErr,
			)
			break
		}
		if result.Success < 1 {
			runErr = fmt.Errorf(
				"automatic load stopped at %d RPS: "+
					"success ratio %.2f%%",
				rps,
				result.Success*100,
			)
			record.StageList[len(record.StageList)-1].Error =
				runErr.Error()

			writeErr = writeAutoResults(rootOutputDir, record, options)
			if writeErr != nil {
				return errors.Join(runErr, writeErr)
			}
			break
		}
	}

	writeErr := writeAutoResults(rootOutputDir, record, options)
	if writeErr != nil {
		return errors.Join(runErr, writeErr)
	}

	summaryPath := filepath.Join(rootOutputDir, autoSummaryFileName)
	runner.info("automatic load summary", "file", summaryPath)
	return runErr
}

func readStageResult(config Config) (stageResult, error) {
	result := stageResult{
		RPS:       config.RPS,
		OutputDir: config.OutputDir,
	}
	reportPath := filepath.Join(config.OutputDir, vegetaReportFileName)
	content, err := os.ReadFile(reportPath)
	if err != nil {
		return result, err
	}

	for _, line := range strings.Split(string(content), "\n") {
		value := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(value, "Requests"):
			fieldList := getReportFieldList(value)
			if len(fieldList) == 3 {
				requests := strings.TrimSpace(fieldList[0])
				throughput := strings.TrimSpace(fieldList[2])
				result.Requests, _ = strconv.Atoi(requests)
				result.Throughput, _ = strconv.ParseFloat(throughput, 64)
			}
		case strings.HasPrefix(value, "Latencies"):
			fieldList := getReportFieldList(value)
			if len(fieldList) == 5 {
				result.P50 = strings.TrimSpace(fieldList[1])
				result.P95 = strings.TrimSpace(fieldList[2])
				result.P99 = strings.TrimSpace(fieldList[3])
			}
		case strings.HasPrefix(value, "Success"):
			percent := getReportValue(value)
			percent = strings.TrimSuffix(percent, "%")
			parsed, parseErr := strconv.ParseFloat(percent, 64)
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

func getReportFieldList(value string) []string {
	return strings.Split(getReportValue(value), ",")
}

func getReportValue(value string) string {
	closingBracket := strings.LastIndex(value, "]")
	return strings.TrimSpace(value[closingBracket+1:])
}

func writeAutoResults(
	outputDir string,
	record autoRunRecord,
	options AutomaticOptions,
) error {
	resultPath := filepath.Join(outputDir, autoResultFileName)
	err := writeJSON(resultPath, record)
	if err != nil {
		return err
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "# %s\n\n", options.SummaryTitle)
	rpsTextList := make([]string, len(record.RPSList))
	for index, rps := range record.RPSList {
		rpsTextList[index] = strconv.Itoa(rps)
	}
	fmt.Fprintf(
		&builder,
		"Built-in RPS ladder: `%s`.\n\n",
		strings.Join(rpsTextList, ", "),
	)
	builder.WriteString(
		"| RPS | Requests | Throughput | Success | " +
			"P50 | P95 | P99 | Status |\n",
	)
	builder.WriteString(
		"| ---: | ---: | ---: | ---: | " +
			"---: | ---: | ---: | --- |\n",
	)
	for _, result := range record.StageList {
		status := "passed"
		if result.Error != "" {
			status = "failed: " + strings.ReplaceAll(
				result.Error,
				"|",
				"\\|",
			)
		}
		fmt.Fprintf(
			&builder,
			"| %d | %d | %.2f | %.2f%% | "+
				"%s | %s | %s | %s |\n",
			result.RPS,
			result.Requests,
			result.Throughput,
			result.Success*100,
			result.P50,
			result.P95,
			result.P99,
			status,
		)
	}
	builder.WriteString(
		"\nEach stage directory contains its load parameters, " +
			"scenario data, raw Vegeta result, and text report.\n",
	)

	summaryPath := filepath.Join(outputDir, autoSummaryFileName)
	return os.WriteFile(summaryPath, []byte(builder.String()), 0o600)
}
