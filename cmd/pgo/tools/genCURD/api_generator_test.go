package genCURD

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestGetProtoFileListUsesCurrentSortedFiles(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "proto", "z_new.gen.proto"), "syntax = \"proto3\";")
	writeTestFile(t, filepath.Join(root, "proto", "common.proto"), "syntax = \"proto3\";")
	writeTestFile(t, filepath.Join(root, "proto", "pgo", "options.proto"), "syntax = \"proto3\";")
	stalePath := filepath.Join(root, "proto", "z_stale.gen.proto")
	writeTestFile(t, stalePath, "syntax = \"proto3\";")
	if err := os.Remove(stalePath); err != nil {
		t.Fatal(err)
	}

	protoFileList, err := getProtoFileList(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"./proto/common.proto",
		"./proto/pgo/options.proto",
		"./proto/z_new.gen.proto",
	}
	if !reflect.DeepEqual(protoFileList, want) {
		t.Fatalf("unexpected proto file list:\ngot  %q\nwant %q", protoFileList, want)
	}
}

func TestRunMakeApiPassesCurrentFilesAndProjectRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake make executable uses a shell script")
	}
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "proto", "z_new.gen.proto"), "syntax = \"proto3\";")
	binDir := t.TempDir()
	outputPath := filepath.Join(t.TempDir(), "make-call.txt")
	fakeMake := `#!/bin/sh
{
  pwd
  printf '%s\n' "$@"
} > "$PGO_TEST_MAKE_OUTPUT"
`
	writeTestFile(t, filepath.Join(binDir, "make"), fakeMake)
	if err := os.Chmod(filepath.Join(binDir, "make"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("PGO_TEST_MAKE_OUTPUT", outputPath)

	outsideRoot := t.TempDir()
	oldWorkDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir(outsideRoot); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWorkDir)

	if err = runMakeApi(root); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	lineList := strings.Split(strings.TrimSpace(string(content)), "\n")
	want := []string{root, "api", "API_PROTO_FILES=./proto/z_new.gen.proto"}
	if !reflect.DeepEqual(lineList, want) {
		t.Fatalf("unexpected make call:\ngot  %q\nwant %q", lineList, want)
	}
}

func TestRunMakeApiIncludesCommandOutputInError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake make executable uses a shell script")
	}
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "proto", "common.proto"), "syntax = \"proto3\";")
	binDir := t.TempDir()
	writeTestFile(t, filepath.Join(binDir, "make"), "#!/bin/sh\necho generator-failed >&2\nexit 2\n")
	if err := os.Chmod(filepath.Join(binDir, "make"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	err := runMakeApi(root)
	if err == nil || !strings.Contains(err.Error(), "make api failed") ||
		!strings.Contains(err.Error(), "generator-failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
