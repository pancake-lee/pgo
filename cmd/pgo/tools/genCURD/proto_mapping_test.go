package genCURD

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseProtoTableMappings(t *testing.T) {
	content := `
syntax = "proto3";
import "pgo/options.proto";
service TaskCURD {
  option (pgo.tables) = "task";
  option (pgo.tables) = "task_item";
}
`
	serviceToTableListMap, err := parseProtoTableMappings(content, "task.proto")
	if err != nil {
		t.Fatal(err)
	}
	tableNameList := serviceToTableListMap["task"]
	if strings.Join(tableNameList, ",") != "task,task_item" {
		t.Fatalf("unexpected task mapping: %v", tableNameList)
	}
}

func TestReadTableToServiceMapRejectsDuplicateMappings(t *testing.T) {
	root := t.TempDir()
	protoDir := filepath.Join(root, "proto")
	if err := os.MkdirAll(protoDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := `import "pgo/options.proto";
service One { option (pgo.tables) = "task"; }
service Two { option (pgo.tables) = "task"; }
`
	if err := os.WriteFile(filepath.Join(protoDir, "mapping.proto"), []byte(content), 0644); err != nil {
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
	_, err = readTableToServiceMap([]string{"task"})
	if err == nil || !strings.Contains(err.Error(), "mapped to both") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddDefaultTableMappings(t *testing.T) {
	protoCode := "service defaultCURD {\n}\n"
	tblList := []*Table{{TblName: "z_table"}, {TblName: "a_table"}}
	result := addDefaultTableMappings(protoCode, tblList)
	if !strings.Contains(result, `option (pgo.tables) = "a_table";`) ||
		!strings.Contains(result, `option (pgo.tables) = "z_table";`) ||
		strings.Index(result, "a_table") > strings.Index(result, "z_table") {
		t.Fatalf("unexpected default mapping block: %s", result)
	}
}
