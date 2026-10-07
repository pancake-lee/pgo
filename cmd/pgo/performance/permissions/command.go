package permissions

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/pancake-lee/pgo/cmd/pgo/common"
	performance "github.com/pancake-lee/pgo/cmd/pgo/performance/core"
	"github.com/pancake-lee/pgo/pkg/pclient"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/pthird"
	"github.com/spf13/cobra"
)

// options 保存权限场景专属参数，公共负载参数继续交给 core。
type options struct {
	ManifestPath string
	PrepareOnly  bool
	KeepData     bool
	WriteRPS     int
}

// entrypoint 将权限场景接入通用 CLI 和交互菜单。
type entrypoint struct{}

// Entrypoint 提供权限读写性能测试入口。
var Entrypoint = &entrypoint{}

// defaultOptions 创建首次读写混合实验参数。
func defaultOptions() options {
	return options{KeepData: true, WriteRPS: 1}
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
	command.Flags().IntVar(&opt.WriteRPS, "write-rps", 1, "administrator updates per second")

	return command
}

// RunInteractive 说明读写并行并读取管理员写速率与公共负载参数。
func (*entrypoint) RunInteractive() {
	pthird.Interact.Infof("读请求与管理员修改同时运行；read RPS 为两个读组的合计。")
	paramMap := pclient.GetCachedParamMap(pconfig.GetDefaultCachePath(),
		"client.performance.permissions.", []pclient.ParamItem{
			{Name: "write-rps", Usage: "administrator write RPS", Default: "1"},
		})
	writeRPS, err := strconv.Atoi(strings.TrimSpace(paramMap["write-rps"]))
	if err != nil || writeRPS < 1 {
		pthird.Interact.Errorf("write-rps must be a positive integer")
		return
	}
	opt := defaultOptions()
	opt.WriteRPS = writeRPS
	newTool(&opt).RunInteractive()
}

// CleanupEntrypoint 提供与权限测试共享目录的独立清理入口。
func (*entrypoint) CleanupEntrypoint() *performance.CleanupEntrypoint {
	opt := defaultOptions()
	return newTool(&opt).CleanupEntrypoint()
}

// preparer 保存批次准备和混合负载共享的场景状态。
type preparer struct {
	config       performance.Config
	logger       klog.Logger
	opt          options
	manifest     *Manifest
	manifestPath string
	client       *apiClient
}

// ResultDirectory 将当前负载结果保存到固定子目录。
func (*preparer) ResultDirectory(directory string) string {
	return filepath.Join(directory, "current")
}

// dataPath 定位准备子目录清单，同时兼容显式清单和旧清理命令参数。
func (preparer *preparer) dataPath(directory string) string {
	if preparer.opt.ManifestPath != "" {
		return preparer.opt.ManifestPath
	}
	if strings.HasSuffix(directory, ".json") {
		return directory
	}
	return filepath.Join(directory, "data", "01-permissions.json")
}

// CheckData 校验当前规模、清单完整性与服务记录，刷新令牌后复用。
func (preparer *preparer) CheckData(ctx context.Context, directory string,
) (performance.DataState, error) {
	preparer.manifestPath = preparer.dataPath(directory)
	manifest, err := readManifest(preparer.manifestPath)
	if errors.Is(err, os.ErrNotExist) {
		return performance.DataMissing, nil
	}
	if err != nil {
		return performance.DataMissing, err
	}
	preparer.manifest = manifest
	if manifest.Cleaned {
		return performance.DataMissing, nil
	}
	baseURL := preparer.config.APIURL
	if !strings.Contains(baseURL, "://") {
		baseURL = "http://" + baseURL
	}
	if strings.TrimRight(baseURL, "/") != strings.TrimRight(manifest.BaseURL, "/") ||
		manifest.Scale != defaultScale || validateFixture(manifest) != nil {
		return performance.DataNeedsRebuild, nil
	}
	err = preparer.refreshTokens(ctx)
	if errors.Is(err, performance.ErrDataMismatch) {
		return performance.DataNeedsRebuild, nil
	}
	if err != nil {
		return performance.DataMissing, err
	}
	err = verifyRecordIDs(ctx, preparer.client, manifest)
	if errors.Is(err, performance.ErrDataMismatch) {
		return performance.DataNeedsRebuild, nil
	}
	if err != nil {
		return performance.DataMissing, err
	}
	err = verifyFixture(ctx, preparer.client, manifest)
	if errors.Is(err, performance.ErrDataMismatch) {
		return performance.DataNeedsRebuild, nil
	}
	return performance.DataReusable, err
}

// CreateData 创建并验证权限批次，规模由场景自身维护。
func (preparer *preparer) CreateData(ctx context.Context, directory string,
) error {
	if preparer.client != nil {
		preparer.client.httpClient.CloseIdleConnections()
	}
	preparer.manifestPath = preparer.dataPath(directory)
	if err := os.MkdirAll(filepath.Dir(preparer.manifestPath), 0o700); err != nil {
		return err
	}
	baseURL := preparer.config.APIURL
	if !strings.Contains(baseURL, "://") {
		baseURL = "http://" + baseURL
	}
	preparer.info("preparing permission data", "projects", defaultScale.Projects,
		"permissions", defaultScale.Projects*defaultScale.Roles*defaultScale.Actions,
		"manifest", preparer.manifestPath)
	var err error
	preparer.manifest, err = prepareFixture(ctx, baseURL,
		preparer.manifestPath, defaultScale, preparer.logger)
	if err != nil {
		return err
	}
	preparer.client = newAPIClient(baseURL, preparer.manifest.Admin.Token)
	return verifyFixture(ctx, preparer.client, preparer.manifest)
}

// CleanupData 从清单恢复批次并幂等删除，不依赖当前负载参数。
func (preparer *preparer) CleanupData(ctx context.Context, directory string,
) error {
	path := preparer.dataPath(directory)
	manifest, err := readManifest(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if manifest.Cleaned {
		return nil
	}
	manifest.Ready = false
	err = saveManifest(path, manifest)
	if err != nil {
		return err
	}
	// 清理时恢复同批次管理员，包括登录响应丢失或删除后未落盘的情况。
	loginClient, err := common.NewClient(manifest.BaseURL)
	if err != nil {
		return err
	}
	name := "perm_" + manifest.Batch + "_admin"
	identity, token, err := loginClient.Login(ctx, name)
	if err != nil {
		return err
	}
	manifest.Admin = User{ID: identity.ID, Name: name, Token: token}
	client := newAPIClient(manifest.BaseURL, token)
	defer client.httpClient.CloseIdleConnections()
	return cleanupFixture(ctx, client, manifest, path)
}

// refreshTokens 为保留的管理员与测试用户重新登录并保存新令牌。
func (preparer *preparer) refreshTokens(ctx context.Context) error {
	if preparer.client != nil {
		preparer.client.httpClient.CloseIdleConnections()
	}
	var err error
	preparer.client, err = refreshAdmin(ctx, preparer.manifest)
	if err != nil {
		return err
	}
	preparer.info("refreshing test user tokens", "users", len(preparer.manifest.UserList))
	loginClient, err := common.NewClient(preparer.manifest.BaseURL)
	if err != nil {
		return err
	}
	for index := range preparer.manifest.UserList {
		user := &preparer.manifest.UserList[index]
		identity, token, loginErr := loginClient.Login(ctx, user.Name)
		if loginErr != nil {
			return loginErr
		}
		if identity.ID != user.ID {
			return performance.ErrDataMismatch
		}
		user.Token = token
	}
	return saveManifest(preparer.manifestPath, preparer.manifest)
}

// Prepare 由 core 复用或重建批次，并生成本次两个读组的目标文件。
func (preparer *preparer) Prepare(ctx context.Context, outputDir string,
) (string, func() error, error) {
	if preparer.opt.WriteRPS < 1 || preparer.config.RPS < 2 {
		return "", nil, errors.New("write-rps must be positive; total read rps must be at least 2")
	}
	cleanup := func() error {
		if preparer.client != nil {
			defer preparer.client.httpClient.CloseIdleConnections()
		}
		if preparer.opt.KeepData || preparer.opt.PrepareOnly {
			return nil
		}
		cleanupContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Minute)
		defer cancel()
		return preparer.CleanupData(cleanupContext, outputDir)
	}
	if !preparer.opt.PrepareOnly {
		err := clearLoadReports(outputDir)
		if err != nil {
			return "", cleanup, err
		}
	}
	dataDir := filepath.Join(outputDir, "data")
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return "", cleanup, err
	}
	err := performance.EnsureData(ctx, outputDir, preparer, preparer.logger)
	if err != nil {
		return "", cleanup, err
	}
	preparer.info("generating permission read targets")
	for _, group := range []string{"hot", "control"} {
		err = writeTargets(filepath.Join(dataDir, group+"-targets.jsonl"), preparer.manifest, group)
		if err != nil {
			return "", cleanup, err
		}
	}
	_ = preparer.logger.Log(klog.LevelInfo, "msg", "HTTP preparation and verification completed",
		"users", len(preparer.manifest.UserList), "permissions", len(preparer.manifest.RecordIDMap["user-role-permission-assoc"]),
		"manifest", preparer.manifestPath)
	return filepath.Join(dataDir, "hot-targets.jsonl"), cleanup, nil
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
		return nil, performance.ErrDataMismatch
	}
	manifest.Admin.Token = token
	return newAPIClient(manifest.BaseURL, token), nil
}
