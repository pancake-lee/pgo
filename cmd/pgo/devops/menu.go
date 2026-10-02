package devops

import (
	"github.com/pancake-lee/pgo/pkg/pclient"
	"github.com/spf13/cobra"
)

// RegisterTools registers the concrete tools owned by DevOps.
func RegisterTools(group *pclient.CommandGroup) {
	group.RegisterTool("Make", pclient.NewCommandEntry(newMakeCommand, MakeCli))
	group.RegisterTool("Init Project", InitProjEntrypoint)
	group.RegisterTool("CD", pclient.NewCommandEntry(newDeployCommand, DeployCli))
}

func newMakeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "make",
		Short: "Run a Makefile target",
		Run: func(_ *cobra.Command, _ []string) {
			MakeCli()
		},
	}
}

func newDeployCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cd",
		Short: "Run deployment operations",
		Run: func(_ *cobra.Command, _ []string) {
			DeployCli()
		},
	}
}
