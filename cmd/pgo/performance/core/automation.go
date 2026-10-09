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

// autoPlanFileName 保存搜索起点、实际档位与已测边界。
const autoPlanFileName = "00-auto-run.json"

const autoRPSPrecision = 10

// stageResult 仅保留判断负载是否成功所需的字段。
type stageResult struct {
	Requests  int
	Success   float64
	HasErrors bool
}

// RunAutomatic 翻倍扩展负载，再在成功与失败之间折半搜索。
func RunAutomatic(
	ctx context.Context,
	runner *Runner,
	config Config,
	runStage func(context.Context, Config) error,
) error {
	if config.RPS <= 0 {
		return fmt.Errorf("automatic RPS must be positive, got %d", config.RPS)
	}
	err := os.MkdirAll(config.OutputDir, 0o700)
	if err != nil {
		return err
	}
	plan := struct {
		StartRPS   int   `json:"startRPS"`
		Precision  int   `json:"precision"`
		RPSList    []int `json:"rpsList"`
		SuccessRPS int   `json:"successRPS,omitempty"`
		FailureRPS int   `json:"failureRPS,omitempty"`
	}{StartRPS: config.RPS, Precision: autoRPSPrecision}
	rps, step := config.RPS, autoRPSPrecision
	var failureErr error
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		plan.RPSList = append(plan.RPSList, rps)
		err = writeJSON(filepath.Join(config.OutputDir, autoPlanFileName), plan)
		if err != nil {
			return err
		}
		stageConfig := config
		stageConfig.RPS = rps
		stageConfig.OutputDir = filepath.Join(config.OutputDir,
			fmt.Sprintf("rps-%03d", rps))
		runner.info("starting automatic load stage",
			"stage", len(plan.RPSList), "rps", rps)
		err = runStage(ctx, stageConfig)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil && !isLoadFailure(err) {
			return fmt.Errorf("automatic load stopped at %d RPS: %w", rps, err)
		}
		if err != nil {
			plan.FailureRPS, failureErr = rps, err
		} else {
			plan.SuccessRPS = rps
		}
		err = writeJSON(filepath.Join(config.OutputDir, autoPlanFileName), plan)
		if err != nil {
			return err
		}
		if plan.SuccessRPS == 0 {
			return fmt.Errorf("starting load failed at %d RPS: %w", rps, failureErr)
		}
		if plan.FailureRPS != 0 {
			gap := plan.FailureRPS - plan.SuccessRPS
			if gap <= autoRPSPrecision {
				runner.info("automatic load boundary found",
					"successRPS", plan.SuccessRPS,
					"failureRPS", plan.FailureRPS,
					"precision", autoRPSPrecision)
				return fmt.Errorf(
					"automatic boundary: highest successful %d RPS, "+
						"lowest failing %d RPS (precision %d): %w",
					plan.SuccessRPS, plan.FailureRPS, autoRPSPrecision, failureErr)
			}
			halfStep := (gap / (2 * autoRPSPrecision)) * autoRPSPrecision
			rps = plan.SuccessRPS + halfStep
			continue
		}
		maxInt := int(^uint(0) >> 1)
		if rps > maxInt-step {
			return fmt.Errorf("automatic RPS overflow after %d RPS", rps)
		}
		rps += step
		if step <= maxInt/2 {
			step *= 2
		}
	}
}

// isLoadFailure 排除与负载失败同时返回的执行器、存储或收尾故障。
func isLoadFailure(err error) bool {
	if err == ErrLoadFailed {
		return true
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		children := joined.Unwrap()
		if len(children) == 0 {
			return false
		}
		for _, child := range children {
			if !isLoadFailure(child) {
				return false
			}
		}
		return true
	}
	if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		return isLoadFailure(wrapped.Unwrap())
	}
	return false
}

// readStageResult 读取单流负载报告，仅用于检测空负载和非成功响应。
func readStageResult(config Config) (stageResult, error) {
	result := stageResult{}
	reportPath := filepath.Join(config.OutputDir, vegetaReportFileName)
	content, err := os.ReadFile(reportPath)
	if err != nil {
		return result, err
	}

	inErrorSet := false
	for _, line := range strings.Split(string(content), "\n") {
		value := strings.TrimSpace(line)
		if inErrorSet && value != "" {
			result.HasErrors = true
		}
		switch {
		case value == "Error Set:":
			inErrorSet = true
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
