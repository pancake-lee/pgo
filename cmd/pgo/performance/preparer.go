package performance

import (
	"context"
	"path/filepath"

	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/pancake-lee/pgo/cmd/pgo/common"
	"github.com/pancake-lee/pgo/cmd/pgo/performance/login"
)

const loginUserCount = 100

const (
	usersFileName        = "01-users.json"
	loginTargetsFileName = "02-login-targets.jsonl"
)

// preparer 定义负载执行前的准备契约。
type preparer interface {
	Prepare(context.Context, string) (targetPath string, cleanup func() error, err error)
}

// loginPreparer 保存登录负载准备与清理所需的依赖。
type loginPreparer struct {
	config loadConfig
	logger klog.Logger
}

// newLoginPreparer 创建登录负载准备器。
func newLoginPreparer(config loadConfig, logger klog.Logger) *loginPreparer {
	return &loginPreparer{config: config, logger: logger}
}

// Prepare 创建并验证测试用户，同时生成负载目标与清理函数。
func (preparer *loginPreparer) Prepare(ctx context.Context, outputDir string) (string, func() error, error) {
	baseURL := normalizeHTTPURL(preparer.config.APIURL)
	client, err := common.NewClient(baseURL)
	if err != nil {
		return "", nil, err
	}
	manifestPath := filepath.Join(outputDir, usersFileName)
	targetPath := filepath.Join(outputDir, loginTargetsFileName)

	preparer.info("preparing login users", "step", "1/7")
	manifest, err := login.Prepare(ctx, client, loginUserCount, manifestPath)
	var cleanup func() error
	if manifest != nil {
		cleanup = func() error {
			preparer.info("cleaning test users", "step", "7/7")
			return login.Cleanup(context.WithoutCancel(ctx), client, manifest)
		}
	}
	if err != nil {
		return "", cleanup, err
	}

	preparer.info("verifying identities and authenticated access", "step", "2/7")
	if err = login.Verify(ctx, client, manifest); err != nil {
		return "", cleanup, err
	}
	preparer.info("generating Vegeta login targets", "step", "3/7")
	if err = manifest.WriteTargets(targetPath); err != nil {
		return "", cleanup, err
	}
	return targetPath, cleanup, nil
}

func (preparer *loginPreparer) info(message string, keyvals ...any) {
	values := append([]any{"msg", message}, keyvals...)
	_ = preparer.logger.Log(klog.LevelInfo, values...)
}
