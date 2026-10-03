package main

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestCommandGroupsBuildTwoLevelCommands 验证分组工具及批次恢复入口。
func TestCommandGroupsBuildTwoLevelCommands(t *testing.T) {
	root := newRootCommand()
	pathList := [][]string{
		{"devops", "make"},
		{"devops", "initProj"},
		{"devops", "cd"},
		{"performance", "login"},
		{"performance", "permissions"},
		{"performance", "permissions", "cleanup"},
		{"application", "course-swap"},
		{"tools", "pretty"},
		{"tools", "psql"},
		{"tools", "curd"},
		{"tools", "gorm"},
		{"tools", "sheet2mysql"},
		{"tools", "health"},
		{"tools", "metrics-url"},
		{"tools", "profile"},
		{"tools", "interaction"},
	}
	for _, path := range pathList {
		testCommandPath(t, root, path)
	}
}

func testCommandPath(t *testing.T, root *cobra.Command, path []string) {
	t.Helper()
	command, _, err := root.Find(path)
	if err != nil {
		t.Fatal(err)
	}
	if command.Name() != path[len(path)-1] {
		t.Fatalf("command path %v resolved to %q", path, command.Name())
	}
}
