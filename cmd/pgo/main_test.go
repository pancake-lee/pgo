package main

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestCommandGroupsBuildTwoLevelCommands 验证工具分组与测试清理命令路径。
func TestCommandGroupsBuildTwoLevelCommands(t *testing.T) {
	root := newRootCommand()
	pathList := [][]string{
		{"devops", "make"},
		{"devops", "initProj"},
		{"devops", "cd"},
		{"performance", "login"},
		{"performance", "login-cleanup"},
		{"performance", "permissions"},
		{"performance", "permissions-cleanup"},
		{"performance", "login", "cleanup"},
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
