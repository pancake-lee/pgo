package performance

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/pancake-lee/pgo/cmd/pgo/performance/login"
)

type performanceScenario interface {
	Prepare(context.Context, string) (targetPath string, cleanup func() error, err error)
}

type loginScenario struct {
	config loginConfig
	now    func() time.Time
	stdout io.Writer
}

func newLoginScenario(config loginConfig, now func() time.Time, stdout io.Writer) *loginScenario {
	return &loginScenario{config: config, now: now, stdout: stdout}
}

func (scenario *loginScenario) Prepare(ctx context.Context, outputDir string) (string, func() error, error) {
	client, err := login.NewClient(normalizeHTTPURL(scenario.config.APIURL), scenario.config.Timeout)
	if err != nil {
		return "", nil, err
	}
	manifestPath := filepath.Join(outputDir, usersFileName)
	targetPath := filepath.Join(outputDir, loginTargetsFileName)
	batchID := scenario.now().UTC().Format("060102150405")

	fmt.Fprintf(scenario.stdout, "[1/7] preparing %d users\n", scenario.config.Users)
	manifest, err := login.Prepare(ctx, client, manifestPath, batchID, scenario.config.Users, scenario.config.Concurrency)
	var cleanup func() error
	if manifest != nil {
		cleanup = func() error {
			fmt.Fprintln(scenario.stdout, "[7/7] cleaning test users")
			cleanupErr := login.Cleanup(context.WithoutCancel(ctx), client, manifest, scenario.config.Concurrency)
			if cleanupErr == nil {
				now := scenario.now().UTC()
				manifest.CleanedAt = &now
				cleanupErr = login.WriteManifest(manifestPath, manifest)
			}
			return cleanupErr
		}
	}
	if err != nil {
		return "", cleanup, err
	}

	fmt.Fprintln(scenario.stdout, "[2/7] verifying identities and authenticated access")
	if err = login.Verify(ctx, client, manifest, scenario.config.Concurrency); err != nil {
		return "", cleanup, err
	}
	fmt.Fprintln(scenario.stdout, "[3/7] generating Vegeta login targets")
	if err = login.WriteTargets(targetPath, manifest); err != nil {
		return "", cleanup, err
	}
	return targetPath, cleanup, nil
}
