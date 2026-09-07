package genGORM

import (
	"strings"
	"testing"

	"github.com/pancake-lee/pgo/pkg/pclient"
)

func TestRunRejectsUnsupportedDatabaseBeforeWriting(t *testing.T) {
	err := Run(pclient.ParamMap{
		paramNameDB:           "postgres",
		paramNameDSN:          "ignored",
		paramNameOutPath:      t.TempDir(),
		paramNameOutFile:      "query.go",
		paramNameModelPkgName: "model",
	})
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}
