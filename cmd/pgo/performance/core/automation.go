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

// autoPlanFileName 保存自动升压的固定档位输入。
const autoPlanFileName = "00-auto-run.json"

// stageResult 仅保留判断负载是否成功所需的字段。
type stageResult struct {
	Requests int
	Success  float64
}

// RunAutomatic 顺序执行固定阶梯，报告留在各档目录，不生成跨档汇总。
func RunAutomatic(
	ctx context.Context,
	runner *Runner,
	config Config,
	rpsList []int,
	runStage func(context.Context, Config) error,
) error {
	if len(rpsList) == 0 {
		return errors.New("automatic RPS list is empty")
	}
	for _, rps := range rpsList {
		if rps <= 0 {
			return fmt.Errorf("automatic RPS must be positive, got %d", rps)
		}
	}
	err := os.MkdirAll(config.OutputDir, 0o700)
	if err != nil {
		return err
	}
	plan := struct {
		RPSList []int `json:"rpsList"`
	}{rpsList}
	err = writeJSON(filepath.Join(config.OutputDir, autoPlanFileName), plan)
	if err != nil {
		return err
	}
	for index, rps := range rpsList {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		stageConfig := config
		stageConfig.RPS = rps
		stageConfig.OutputDir = filepath.Join(config.OutputDir, fmt.Sprintf("rps-%03d", rps))
		runner.info("starting automatic load stage", "stage", index+1,
			"stages", len(rpsList), "rps", rps)
		err = runStage(ctx, stageConfig)
		if err != nil {
			return fmt.Errorf("automatic load stopped at %d RPS: %w", rps, err)
		}
	}
	return nil
}

// readStageResult 读取单流负载报告，仅用于检测空负载和非成功响应。
func readStageResult(config Config) (stageResult, error) {
	result := stageResult{}
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
				result.Requests, _ = strconv.Atoi(requests)
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
