package genCURD

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	glebarez "github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestRunGenerateMovesTableFromDefaultService(t *testing.T) {
	root := t.TempDir()
	writeGeneratorFixture(t, root)
	dbPath := filepath.Join(root, "fixture.sqlite")
	db, err := gorm.Open(glebarez.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Exec("CREATE TABLE abandon_code (idx1 INTEGER PRIMARY KEY, col1 TEXT)").Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Exec("CREATE TABLE task (id INTEGER PRIMARY KEY, title TEXT)").Error; err != nil {
		t.Fatal(err)
	}

	oldWorkDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWorkDir)
	oldRunAPIGenerator := runAPIGenerator
	runAPIGenerator = func() error { return nil }
	defer func() { runAPIGenerator = oldRunAPIGenerator }()

	if err = runGenerate("sqlite3", dbPath); err != nil {
		t.Fatal(err)
	}
	defaultProtoPath := filepath.Join("proto", "z_defaultService.gen.proto")
	defaultProto, err := os.ReadFile(defaultProtoPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(defaultProto), `option (pgo.tables) = "task";`) {
		t.Fatalf("default service does not contain task mapping: %s", defaultProto)
	}

	customProto := `syntax = "proto3";
import "pgo/options.proto";
service Task {
  option (pgo.tables) = "task";
}
`
	if err = os.WriteFile(filepath.Join("proto", "taskService.proto"), []byte(customProto), 0644); err != nil {
		t.Fatal(err)
	}
	defaultProto = []byte(strings.Replace(string(defaultProto), "    option (pgo.tables) = \"task\";\n", "", 1))
	if err = os.WriteFile(defaultProtoPath, defaultProto, 0644); err != nil {
		t.Fatal(err)
	}

	if err = runGenerate("sqlite3", dbPath); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(defaultProtoPath); !os.IsNotExist(err) {
		t.Fatalf("default proto remains after mapping migration: %v", err)
	}
	taskProto, err := os.ReadFile(filepath.Join("proto", "z_taskService.gen.proto"))
	if err != nil || !strings.Contains(string(taskProto), "service taskCURD") {
		t.Fatalf("custom service proto was not generated: %v\n%s", err, taskProto)
	}
}

func writeGeneratorFixture(t *testing.T, root string) {
	t.Helper()
	sourceRoot, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	sourceRoot = filepath.Clean(filepath.Join(sourceRoot, "../../../.."))
	for _, relativePath := range []string{
		"internal/abandonCodeService/data/dao_AbandonCode.go",
		"internal/abandonCodeService/service/svc_AbandonCode.go",
		"internal/abandonCodeService/service/svr_AbandonCode.go",
		"internal/abandonCodeService/abandonService.go",
		"proto/abandonCode.proto",
	} {
		content, err := os.ReadFile(filepath.Join(sourceRoot, relativePath))
		if err != nil {
			t.Fatal(err)
		}
		targetPath := filepath.Join(root, relativePath)
		if err = os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(targetPath, content, 0644); err != nil {
			t.Fatal(err)
		}
	}
}
