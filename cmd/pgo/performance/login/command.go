package login

import (
	"context"
	"path/filepath"
	"strings"

	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/pancake-lee/pgo/cmd/pgo/common"
	performance "github.com/pancake-lee/pgo/cmd/pgo/performance/core"
)

const (
	loginUserCount  = 100
	usersFileName   = "01-users.json"
	targetsFileName = "02-login-targets.jsonl"
)

// Entrypoint exposes login load testing without knowing its menu placement.
var Entrypoint = performance.NewEntrypoint(performance.Scenario{
	Name:  "login",
	Short: "Run the user login load-test workflow",
	NewPreparer: func(config performance.Config, logger klog.Logger,
	) performance.Preparer {
		return newPreparer(config, logger)
	},
})

type preparer struct {
	config performance.Config
	logger klog.Logger
}

func newPreparer(config performance.Config, logger klog.Logger) *preparer {
	return &preparer{config: config, logger: logger}
}

func (preparer *preparer) Prepare(ctx context.Context, outputDir string,
) (string, func() error, error) {
	baseURL := normalizeHTTPURL(preparer.config.APIURL)
	client, err := common.NewClient(baseURL)
	if err != nil {
		return "", nil, err
	}

	manifestPath := filepath.Join(outputDir, usersFileName)
	targetPath := filepath.Join(outputDir, targetsFileName)

	preparer.info("preparing login users", "step", "1/7")
	manifest, err := Prepare(ctx, client, loginUserCount, manifestPath)
	var cleanup func() error
	if manifest != nil {
		cleanup = func() error {
			preparer.info("cleaning test users", "step", "7/7")
			return Cleanup(context.WithoutCancel(ctx), client, manifest)
		}
	}
	if err != nil {
		return "", cleanup, err
	}

	preparer.info("verifying identities and authenticated access", "step", "2/7")
	err = Verify(ctx, client, manifest)
	if err != nil {
		return "", cleanup, err
	}

	preparer.info("generating Vegeta login targets", "step", "3/7")
	err = manifest.WriteTargets(targetPath)
	if err != nil {
		return "", cleanup, err
	}

	preparer.info(
		"scenario artifact",
		"file",
		usersFileName,
		"description",
		"login user manifest; contains tokens",
	)
	preparer.info(
		"scenario artifact",
		"file",
		targetsFileName,
		"description",
		"Vegeta login request definitions",
	)
	return targetPath, cleanup, nil
}

func (preparer *preparer) info(message string, keyvals ...any) {
	values := append([]any{"msg", message}, keyvals...)
	_ = preparer.logger.Log(klog.LevelInfo, values...)
}

func normalizeHTTPURL(address string) string {
	address = strings.TrimSpace(address)
	if !strings.Contains(address, "://") {
		return "http://" + address
	}
	return address
}
