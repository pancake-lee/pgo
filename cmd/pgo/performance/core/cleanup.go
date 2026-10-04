package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/pancake-lee/pgo/pkg/pclient"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/pthird"
	"github.com/spf13/cobra"
)

// CleanupEntrypoint 暴露与测试入口共享数据目录的独立清理工具。
type CleanupEntrypoint struct {
	entrypoint *Entrypoint
}

// CleanupEntrypoint 为场景生成通用清理入口。
func (entrypoint *Entrypoint) CleanupEntrypoint() *CleanupEntrypoint {
	return &CleanupEntrypoint{entrypoint: entrypoint}
}

// NewCobraCommand 创建菜单同级的清理命令。
func (cleanup *CleanupEntrypoint) NewCobraCommand() *cobra.Command {
	return cleanup.newCommand(cleanup.entrypoint.scenario.Name + "-cleanup")
}

// newCommand 创建独立或场景子命令，不发现服务也不运行负载。
func (cleanup *CleanupEntrypoint) newCommand(name string) *cobra.Command {
	return &cobra.Command{
		Use: name + " <output-dir>", Short: "清理该目录记录的测试数据",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cleanup.run(cmd.Context(), args[0],
				klog.NewStdLogger(cmd.OutOrStdout()))
		},
	}
}

// RunInteractive 复用测试入口的目录缓存，仅询问清理所需的目录。
func (cleanup *CleanupEntrypoint) RunInteractive() {
	paramList := []pclient.ParamItem{
		{Name: "output-dir", Usage: "output directory", Default: ""},
	}
	values := pclient.GetCachedParamMap(pconfig.GetDefaultCachePath(),
		"client.performance."+cleanup.entrypoint.scenario.Name+".", paramList)
	err := cleanup.run(context.Background(), values["output-dir"],
		plogger.GetDefaultLoggerNoCaller())
	if err != nil {
		pthird.Interact.Errorf("Test data cleanup failed: %v", err)
	}
}

// run 调用场景清理契约，所有批次身份与服务地址均从清单读取。
func (cleanup *CleanupEntrypoint) run(ctx context.Context, directory string,
	logger klog.Logger,
) error {
	if strings.TrimSpace(directory) == "" {
		return fmt.Errorf("output directory is required")
	}
	config := Config{OutputDir: directory}
	preparer := cleanup.entrypoint.scenario.NewPreparer(config, logger)
	data, ok := preparer.(DataLifecycle)
	if !ok {
		return fmt.Errorf("%s does not implement data cleanup",
			cleanup.entrypoint.scenario.Name)
	}
	started := time.Now()
	err := data.CleanupData(ctx, directory)
	_ = logger.Log(klog.LevelInfo, "msg", "test data cleanup completed",
		"success", err == nil, "elapsed", time.Since(started),
		"outputDir", directory)
	return err
}
