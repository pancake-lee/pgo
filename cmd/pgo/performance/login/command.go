package login

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/pancake-lee/pgo/cmd/pgo/common"
	performance "github.com/pancake-lee/pgo/cmd/pgo/performance/core"
)

const (
	// loginUserCount 保存登录场景需要的用户数。
	loginUserCount = 100
	// usersFileName 保存登录批次清单文件名。
	usersFileName = "01-users.json"
	// targetsFileName 保存登录负载目标文件名。
	targetsFileName = "02-login-targets.jsonl"
)

// Entrypoint 提供登录性能测试入口。
var Entrypoint = performance.NewEntrypoint(performance.Scenario{
	Name: "login", Short: "登录性能测试，复用当前目录的测试用户",
	NewPreparer: func(config performance.Config, logger klog.Logger,
	) performance.Preparer {
		return newPreparer(config, logger)
	},
})

// preparer 保存登录场景数据与 HTTP 客户端。
type preparer struct {
	config   performance.Config
	logger   klog.Logger
	manifest *Manifest
	client   *common.Client
}

// newPreparer 创建一次登录场景调用的状态。
func newPreparer(config performance.Config, logger klog.Logger) *preparer {
	return &preparer{config: config, logger: logger}
}

// CheckData 检查清单规模、批次状态和实际用户身份，刷新复用令牌。
func (preparer *preparer) CheckData(ctx context.Context, directory string,
) (performance.DataState, error) {
	manifest := &Manifest{}
	err := manifest.read(filepath.Join(directory, usersFileName))
	if errors.Is(err, os.ErrNotExist) {
		return performance.DataMissing, nil
	}
	if err != nil {
		return performance.DataMissing, err
	}
	preparer.manifest = manifest
	if manifest.CleanedAt != nil {
		return performance.DataMissing, nil
	}
	baseURL := strings.TrimRight(normalizeHTTPURL(preparer.config.APIURL), "/")
	if baseURL != strings.TrimRight(manifest.BaseURL, "/") {
		return performance.DataNeedsRebuild, nil
	}
	if manifest.Cleaning || len(manifest.Users) != loginUserCount ||
		(manifest.Version > 0 && (!manifest.Ready || manifest.ExpectedCount != loginUserCount)) {
		return performance.DataNeedsRebuild, nil
	}
	preparer.client, err = common.NewClient(manifest.BaseURL)
	if err != nil {
		return performance.DataMissing, err
	}
	controller, token, err := preparer.client.Login(ctx, manifest.Users[0].UserName)
	if err != nil {
		return performance.DataMissing, err
	}
	if controller.ID != manifest.Users[0].ID {
		return performance.DataNeedsRebuild, nil
	}
	userList, err := preparer.client.GetAllUserList(ctx, token)
	if err != nil {
		return performance.DataMissing, err
	}
	userMap := make(map[int32]string, len(userList))
	for _, user := range userList {
		userMap[user.ID] = user.UserName
	}
	for _, user := range manifest.Users {
		if userMap[user.ID] != user.UserName {
			return performance.DataNeedsRebuild, nil
		}
	}
	err = Verify(ctx, preparer.client, manifest)
	return performance.DataReusable, err
}

// CreateData 创建并验证新批次，创建前由 core 保证旧批次已清理。
func (preparer *preparer) CreateData(ctx context.Context, directory string,
) error {
	var err error
	preparer.client, err = common.NewClient(normalizeHTTPURL(preparer.config.APIURL))
	if err != nil {
		return err
	}
	preparer.info("preparing login users", "users", loginUserCount)
	preparer.manifest, err = Prepare(ctx, preparer.client, loginUserCount,
		filepath.Join(directory, usersFileName))
	if err != nil {
		return err
	}
	return Verify(ctx, preparer.client, preparer.manifest)
}

// CleanupData 读取同目录清单并清理，已清理或没有清单时直接完成。
func (preparer *preparer) CleanupData(ctx context.Context, directory string,
) error {
	manifest := &Manifest{}
	err := manifest.read(filepath.Join(directory, usersFileName))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if manifest.CleanedAt != nil {
		return nil
	}
	client, err := common.NewClient(manifest.BaseURL)
	if err != nil {
		return err
	}
	return Cleanup(ctx, client, manifest)
}

// Prepare 由 core 复用或重建数据，只生成本次目标，测试后保留批次。
func (preparer *preparer) Prepare(ctx context.Context, outputDir string,
) (string, func() error, error) {
	err := performance.EnsureData(ctx, outputDir, preparer, preparer.logger)
	if err != nil {
		return "", nil, err
	}
	targetPath := filepath.Join(outputDir, targetsFileName)
	preparer.info("generating Vegeta login targets")
	err = preparer.manifest.WriteTargets(targetPath)
	return targetPath, nil, err
}

// info 输出不包含令牌的场景进度。
func (preparer *preparer) info(message string, keyvals ...any) {
	values := append([]any{"msg", message}, keyvals...)
	_ = preparer.logger.Log(klog.LevelInfo, values...)
}

// normalizeHTTPURL 为没有协议的服务地址补全 HTTP 协议。
func normalizeHTTPURL(address string) string {
	address = strings.TrimSpace(address)
	if !strings.Contains(address, "://") {
		return "http://" + address
	}
	return address
}
