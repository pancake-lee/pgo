package core

import (
	"context"
	"errors"
	"fmt"
	"os"

	klog "github.com/go-kratos/kratos/v2/log"
)

// DataState 表示当前目录中的测试批次是否可以复用。
type DataState int

const (
	// DataMissing 表示没有数据或旧批次已清理完成。
	DataMissing DataState = iota
	// DataReusable 表示现有批次完整且满足当前场景需求。
	DataReusable
	// DataNeedsRebuild 表示旧批次未完成或不满足当前需求。
	DataNeedsRebuild
)

// ErrDataMismatch 标识确定的数据缺失或内容变化，不表示请求失败。
var ErrDataMismatch = errors.New("test data no longer matches the scenario")

// DataLifecycle 由场景实现 HTTP 数据检查、创建和可恢复的精确清理。
type DataLifecycle interface {
	CheckData(context.Context, string) (DataState, error)
	CreateData(context.Context, string) error
	CleanupData(context.Context, string) error
}

// EnsureData 统一复用规则，旧批次清理成功后才允许创建新批次。
func EnsureData(ctx context.Context, directory string,
	data DataLifecycle, logger klog.Logger,
) error {
	err := os.MkdirAll(directory, 0o700)
	if err != nil {
		return err
	}
	state, err := data.CheckData(ctx, directory)
	if err != nil {
		return fmt.Errorf("check test data: %w", err)
	}
	switch state {
	case DataReusable:
		_ = logger.Log(klog.LevelInfo, "msg", "reusing test data",
			"outputDir", directory)
		return nil
	case DataNeedsRebuild:
		_ = logger.Log(klog.LevelInfo, "msg", "cleaning old test data before rebuilding",
			"outputDir", directory)
		err = data.CleanupData(ctx, directory)
		if err != nil {
			return fmt.Errorf("clean old test data; batch retained: %w", err)
		}
	case DataMissing:
	default:
		return fmt.Errorf("unknown test data state %d", state)
	}
	_ = logger.Log(klog.LevelInfo, "msg", "creating test data",
		"outputDir", directory)
	return data.CreateData(ctx, directory)
}
