package permissions

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	klog "github.com/go-kratos/kratos/v2/log"
	performance "github.com/pancake-lee/pgo/cmd/pgo/performance/core"
	"github.com/pancake-lee/pgo/pkg/pclient"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/pthird"
	"github.com/spf13/cobra"
)

// aggregationHeader 标识服务和压测目标采用的聚合算法。
const aggregationHeader = "X-Pgo-Permission-Aggregation"

// options 保存角色规模、唯一权限数及聚合算法。
type options struct {
	Roles       int    `json:"roles"`
	Permissions int    `json:"permissions"`
	Aggregation string `json:"aggregation"`
}

// validate 限制练习数据规模并校验算法选择。
func (options options) validate() error {
	if options.Roles < 1 || options.Roles > 50 {
		return fmt.Errorf("roles must be between 1 and 50")
	}
	if options.Permissions < 1 || options.Permissions > 2000 {
		return fmt.Errorf("permissions must be between 1 and 2000")
	}
	if options.Aggregation != "list" && options.Aggregation != "map" {
		return fmt.Errorf("aggregation must be list or map")
	}
	return nil
}

// entrypoint 适配权限场景的命令行与交互式参数。
type entrypoint struct{}

// Entrypoint 提供权限合并练习入口。
var Entrypoint = &entrypoint{}

// newLoadEntrypoint 将权限准备流程接入公共负载框架。
func newLoadEntrypoint(options *options) *performance.Entrypoint {
	scenario := performance.Scenario{
		Name:                  "permissions",
		Short:                 "练习多角色权限合并的性能排查",
		AutomaticSummaryTitle: "Permissions automatic load result",
		Warmup:                30 * time.Second,
		DefaultRPS:            "10",
		NewPreparer: func(config performance.Config, logger klog.Logger,
		) performance.Preparer {
			return &preparer{config: config, options: *options, logger: logger}
		},
	}
	return performance.NewEntrypoint(scenario)
}

// NewCobraCommand 提供规模、算法与中断后清理参数。
func (*entrypoint) NewCobraCommand() *cobra.Command {
	options := options{Roles: 10, Permissions: 500, Aggregation: "list"}
	command := newLoadEntrypoint(&options).NewCobraCommand()
	command.PreRunE = func(_ *cobra.Command, _ []string) error {
		return options.validate()
	}
	command.Flags().IntVar(&options.Roles, "roles", 10, "每用户的角色数，1 至 50")
	command.Flags().IntVar(
		&options.Permissions, "permissions", 500, "每角色的重复权限数，1 至 2000",
	)
	command.Flags().StringVar(
		&options.Aggregation, "aggregation", "list", "权限合并算法：list 或 map",
	)
	cleanup := &cobra.Command{
		Use:   "cleanup <manifest>",
		Short: "根据批次清单清理中断后的权限练习数据",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manifest, err := readManifest(args[0])
			if err != nil {
				return err
			}
			return manifest.cleanup(cmd.Context())
		},
	}
	command.AddCommand(cleanup)
	return command
}

// RunInteractive 读取权限参数后复用公共负载交互。
func (*entrypoint) RunInteractive() {
	paramList := []pclient.ParamItem{
		{Name: "roles", Usage: "roles per user (1-50)", Default: "10"},
		{
			Name: "permissions", Usage: "permissions per role (1-2000)",
			Default: "500",
		},
		{
			Name: "aggregation", Usage: "aggregation (list/map)",
			Default: "list",
		},
	}
	paramMap := pclient.GetCachedParamMap(
		pconfig.GetDefaultCachePath(), "client.performance.permissions.", paramList,
	)
	roles, rolesErr := strconv.Atoi(paramMap["roles"])
	permissions, permissionsErr := strconv.Atoi(paramMap["permissions"])
	options := options{
		Roles: roles, Permissions: permissions,
		Aggregation: paramMap["aggregation"],
	}
	err := options.validate()
	if rolesErr != nil || permissionsErr != nil || err != nil {
		pthird.Interact.Errorf("Invalid permissions parameters: %v", err)
		return
	}
	newLoadEntrypoint(&options).RunInteractive()
}

// preparer 协调权限数据准备、结果验证与自动清理。
type preparer struct {
	config  performance.Config
	options options
	logger  klog.Logger
}

// Prepare 准备重叠权限并生成已核对算法的请求目标。
func (preparer *preparer) Prepare(ctx context.Context, outputDir string,
) (string, func() error, error) {
	err := preparer.options.validate()
	if err != nil {
		return "", nil, err
	}
	manifest, err := newManifest(preparer.config.APIURL, preparer.options,
		filepath.Join(outputDir, "01-permissions.json"))
	if err != nil {
		return "", nil, err
	}
	cleanup := func() error {
		cleanupCtx, cancel := context.WithTimeout(
			context.WithoutCancel(ctx), 2*time.Minute,
		)
		defer cancel()
		return manifest.cleanup(cleanupCtx)
	}
	_ = preparer.logger.Log(klog.LevelInfo,
		"msg", "preparing overlapping role permissions",
		"roles", preparer.options.Roles,
		"permissions", preparer.options.Permissions,
		"aggregation", preparer.options.Aggregation,
		"manifest", manifest.path,
	)
	err = manifest.prepare(ctx, preparer.logger)
	if err != nil {
		return "", cleanup, err
	}
	err = manifest.verify(ctx)
	if err != nil {
		return "", cleanup, err
	}
	targetPath := filepath.Join(outputDir, "02-permissions-targets.jsonl")
	err = manifest.writeTargets(targetPath)
	return targetPath, cleanup, err
}
