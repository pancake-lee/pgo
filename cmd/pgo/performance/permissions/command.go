package permissions

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/pancake-lee/pgo/cmd/pgo/common"
	performance "github.com/pancake-lee/pgo/cmd/pgo/performance/core"
	"github.com/spf13/cobra"
)

// options 保存权限场景专属参数，公共负载参数继续交给 core。
type options struct {
	ManifestPath string
	PrepareOnly  bool
	KeepData     bool
	WriteRPS     int
	Repeat       int
}

// entrypoint 将权限场景接入通用 CLI 和交互菜单。
type entrypoint struct{}

// Entrypoint 提供权限读写性能测试入口。
var Entrypoint = &entrypoint{}

// defaultOptions 创建首次读写混合实验参数。
func defaultOptions() options {
	return options{KeepData: true, WriteRPS: 1, Repeat: 1}
}

// newTool 为本次调用绑定专属选项和通用负载框架。
func newTool(opt *options) *performance.Entrypoint {
	return performance.NewEntrypoint(performance.Scenario{
		Name: "permissions", Short: "共享角色权限 HTTP 读写混合测试",
		DefaultRPS: "20", DefaultDuration: "120s",
		NewPreparer: func(config performance.Config, logger klog.Logger) performance.Preparer {
			return &preparer{config: config, logger: logger, opt: *opt}
		},
	})
}

// NewCobraCommand 复用公共参数并添加清单复用、准备和清理能力。
func (*entrypoint) NewCobraCommand() *cobra.Command {
	opt := defaultOptions()
	command := newTool(&opt).NewCobraCommand()
	command.Flags().StringVar(&opt.ManifestPath, "manifest", "", "reuse an existing HTTP data manifest")
	command.Flags().BoolVar(&opt.PrepareOnly, "prepare-only", false, "only create and verify HTTP test data")
	command.Flags().BoolVar(&opt.KeepData, "keep-data", true, "keep the batch for later comparisons; clean up explicitly")
	command.Flags().IntVar(&opt.WriteRPS, "write-rps", 1, "administrator updates per second in the mixed window")
	command.Flags().IntVar(&opt.Repeat, "repeat", 1, "number of pure/mixed/recovery cycles using the same batch")
	command.AddCommand(&cobra.Command{
		Use: "cleanup <manifest>", Short: "通过 HTTP 精确清理测试批次",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manifest, err := loadManifest(args[0])
			if err != nil {
				return err
			}
			client, err := refreshAdmin(cmd.Context(), manifest)
			if err != nil {
				return err
			}
			defer client.httpClient.CloseIdleConnections()
			started := time.Now()
			err = cleanupFixture(cmd.Context(), client, manifest, args[0])
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Cleanup completed=%t elapsed=%s\n", err == nil, time.Since(started))
			return err
		},
	})
	return command
}

// RunInteractive 使用通用菜单读取负载参数，专属参数取首次默认值。
func (*entrypoint) RunInteractive() {
	opt := defaultOptions()
	newTool(&opt).RunInteractive()
}

// preparer 保存批次准备和三个窗口共享的场景状态。
type preparer struct {
	config       performance.Config
	logger       klog.Logger
	opt          options
	manifest     *Manifest
	manifestPath string
	client       *apiClient
}

// Prepare 创建或重新登录现有批次，并生成两个读组的目标文件。
func (preparer *preparer) Prepare(ctx context.Context, outputDir string,
) (string, func() error, error) {
	if preparer.opt.WriteRPS < 1 || preparer.opt.Repeat < 1 || preparer.config.RPS < 2 {
		return "", nil, errors.New("write-rps and repeat must be positive; total read rps must be at least 2")
	}
	baseURL := preparer.config.APIURL
	if !strings.Contains(baseURL, "://") {
		baseURL = "http://" + baseURL
	}
	preparer.manifestPath = preparer.opt.ManifestPath
	var err error
	if preparer.manifestPath == "" {
		preparer.manifestPath = filepath.Join(outputDir, "01-permissions.json")
		preparer.info("preparing permission data",
			"projects", defaultScale.Projects,
			"permissions", defaultScale.Projects*defaultScale.Roles*defaultScale.Actions,
			"manifest", preparer.manifestPath)
		preparer.manifest, err = prepareFixture(ctx, baseURL,
			preparer.manifestPath, defaultScale, preparer.logger)
	} else {
		preparer.info("reusing permission data", "manifest", preparer.manifestPath)
		preparer.manifest, err = loadManifest(preparer.manifestPath)
		if err == nil && strings.TrimRight(baseURL, "/") != strings.TrimRight(preparer.manifest.BaseURL, "/") {
			err = errors.New("manifest API URL differs from discovered API URL")
		}
	}
	cleanup := func() error {
		if preparer.client != nil {
			defer preparer.client.httpClient.CloseIdleConnections()
		}
		if preparer.manifest == nil || preparer.opt.KeepData || preparer.opt.PrepareOnly {
			return nil
		}
		preparer.info("cleaning permission data", "manifest", preparer.manifestPath)
		cleanupContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Minute)
		defer cancel()
		client, refreshErr := refreshAdmin(cleanupContext, preparer.manifest)
		if refreshErr != nil {
			return refreshErr
		}
		defer client.httpClient.CloseIdleConnections()
		return cleanupFixture(cleanupContext, client, preparer.manifest, preparer.manifestPath)
	}
	if err != nil {
		return "", cleanup, err
	}
	err = validateFixture(preparer.manifest)
	if err != nil {
		return "", cleanup, err
	}
	preparer.info("refreshing test user tokens",
		"users", len(preparer.manifest.UserList))
	preparer.client, err = refreshAdmin(ctx, preparer.manifest)
	if err != nil {
		return "", cleanup, err
	}
	loginClient, err := common.NewClient(baseURL)
	if err != nil {
		return "", cleanup, err
	}
	// 重新登录保证复用清单时没有过期令牌，登录不计入正式读负载。
	for index := range preparer.manifest.UserList {
		user := &preparer.manifest.UserList[index]
		identity, token, loginErr := loginClient.Login(ctx, user.Name)
		if loginErr != nil {
			return "", cleanup, loginErr
		}
		if identity.ID != user.ID {
			return "", cleanup, errors.New("manifest user identity changed")
		}
		user.Token = token
	}
	err = saveManifest(preparer.manifestPath, preparer.manifest)
	if err == nil {
		preparer.info("verifying permissions and project isolation")
		err = verifyFixture(ctx, preparer.client, preparer.manifest)
	}
	if err != nil {
		return "", cleanup, err
	}
	preparer.info("generating permission read targets")
	for _, group := range []string{"hot", "control"} {
		err = writeTargets(filepath.Join(outputDir, group+"-targets.jsonl"), preparer.manifest, group)
		if err != nil {
			return "", cleanup, err
		}
	}
	_ = preparer.logger.Log(klog.LevelInfo, "msg", "HTTP preparation and verification completed",
		"users", len(preparer.manifest.UserList), "permissions", len(preparer.manifest.RecordIDMap["user-role-permission-assoc"]),
		"manifest", preparer.manifestPath)
	return filepath.Join(outputDir, "hot-targets.jsonl"), cleanup, nil
}

// info 使用通用场景日志入口输出阶段提示，不包含请求令牌。
func (preparer *preparer) info(message string, keyvals ...any) {
	values := append([]any{"msg", message}, keyvals...)
	_ = preparer.logger.Log(klog.LevelInfo, values...)
}

// refreshAdmin 重新登录批次管理员并校验身份，避免令牌过期影响清理。
func refreshAdmin(ctx context.Context, manifest *Manifest) (*apiClient, error) {
	if manifest.Admin.ID <= 0 || manifest.Admin.Name == "" {
		return nil, errors.New("manifest has no administrator identity")
	}
	client, err := common.NewClient(manifest.BaseURL)
	if err != nil {
		return nil, err
	}
	identity, token, err := client.Login(ctx, manifest.Admin.Name)
	if err != nil {
		return nil, err
	}
	if identity.ID != manifest.Admin.ID {
		return nil, errors.New("administrator identity changed")
	}
	manifest.Admin.Token = token
	return newAPIClient(manifest.BaseURL, token), nil
}
