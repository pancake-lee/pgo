package pclient

import (
	"github.com/pancake-lee/pgo/pkg/pthird"
	"github.com/spf13/cobra"
)

// CommandEntry is a concrete tool exposed in Cobra and interactive modes.
type CommandEntry interface {
	NewCobraCommand() *cobra.Command
	RunInteractive()
}

// CommandEntryFunc adapts an existing tool while it migrates to Tool.
type CommandEntryFunc struct {
	newCobraCommand func() *cobra.Command
	runInteractive  func()
}

// NewCommandEntry adapts a tool-specific Cobra command and interactive entry.
func NewCommandEntry(
	newCobraCommand func() *cobra.Command,
	runInteractive func(),
) *CommandEntryFunc {
	return &CommandEntryFunc{
		newCobraCommand: newCobraCommand,
		runInteractive:  runInteractive,
	}
}

func (entry *CommandEntryFunc) NewCobraCommand() *cobra.Command {
	return entry.newCobraCommand()
}

func (entry *CommandEntryFunc) RunInteractive() {
	entry.runInteractive()
}

type groupEntry struct {
	label string
	tool  CommandEntry
}

// CommandGroup is a first-level menu containing concrete second-level tools.
type CommandGroup struct {
	use   string
	short string
	tools []groupEntry
}

// NewCommandGroup creates a first-level command and interactive group.
func NewCommandGroup(use, short string) *CommandGroup {
	return &CommandGroup{use: use, short: short}
}

// RegisterTool adds a concrete tool to the second level of the group.
func (group *CommandGroup) RegisterTool(label string, tool CommandEntry) {
	group.tools = append(group.tools, groupEntry{label: label, tool: tool})
}

// NewCobraCommand builds the group's Cobra command tree.
func (group *CommandGroup) NewCobraCommand() *cobra.Command {
	command := &cobra.Command{Use: group.use, Short: group.short}
	for _, entry := range group.tools {
		command.AddCommand(entry.tool.NewCobraCommand())
	}
	return command
}

// RunInteractive opens the group's second-level interactive menu.
func (group *CommandGroup) RunInteractive(label string) {
	selector := pthird.Interact.NewSelector(label)
	for _, entry := range group.tools {
		selector.Reg(entry.label, entry.tool.RunInteractive)
	}
	selector.Loop()
}
