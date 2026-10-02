package pclient

import (
	"testing"

	"github.com/spf13/cobra"
)

type fakeCommandEntry struct {
	use string
}

func (entry fakeCommandEntry) NewCobraCommand() *cobra.Command {
	return &cobra.Command{Use: entry.use}
}

func (fakeCommandEntry) RunInteractive() {}

func TestCommandMenuBuildsTwoLevelCobraTree(t *testing.T) {
	group := NewCommandGroup("devops", "DevOps tools")
	group.RegisterTool("Init Project", fakeCommandEntry{use: "init-project"})

	root := &cobra.Command{Use: "pgo"}
	root.AddCommand(group.NewCobraCommand())

	groupCommand, _, err := root.Find([]string{"devops"})
	if err != nil {
		t.Fatal(err)
	}
	if groupCommand.Name() != "devops" {
		t.Fatalf("group command = %q", groupCommand.Name())
	}

	toolCommand, _, err := root.Find([]string{"devops", "init-project"})
	if err != nil {
		t.Fatal(err)
	}
	if toolCommand.Name() != "init-project" {
		t.Fatalf("tool command = %q", toolCommand.Name())
	}
}
