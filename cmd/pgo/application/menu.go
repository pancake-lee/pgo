package application

import (
	"github.com/pancake-lee/pgo/cmd/pgo/application/courseSwap"
	"github.com/pancake-lee/pgo/pkg/pclient"
	"github.com/spf13/cobra"
)

// RegisterTools registers application tools owned by courseSwap.
func RegisterTools(group *pclient.CommandGroup) {
	group.RegisterTool(
		"调课 (Course Swap)",
		pclient.NewCommandEntry(
			newCourseSwapCommand,
			courseSwap.CourseSwapCli,
		),
	)
}

func newCourseSwapCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "course-swap",
		Short: "Find and execute a course swap",
		Run: func(_ *cobra.Command, _ []string) {
			courseSwap.CourseSwapCli()
		},
	}
}
