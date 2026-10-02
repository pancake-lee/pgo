package main

import (
	"os"

	"github.com/pancake-lee/pgo/cmd/pgo/application"
	"github.com/pancake-lee/pgo/cmd/pgo/devops"
	"github.com/pancake-lee/pgo/cmd/pgo/performance"
	"github.com/pancake-lee/pgo/cmd/pgo/tools"
	"github.com/pancake-lee/pgo/pkg/pclient"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/pthird"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

func main() {
	pclient.RunApp(runCli, runUI)
}

func initLogger(logToConsole bool) {
	plogger.SetJsonLog(false)
	plogger.InitLogger(logToConsole, zapcore.DebugLevel, "./logs/")
}

// --------------------------------------------------
func runCli() {
	rootCmd := newRootCommand()
	rootCmd.SetArgs(pclient.NormalizeLegacyLongFlagArgs(os.Args[1:]))
	if err := rootCmd.Execute(); err != nil {
		pthird.Interact.Errorf("命令执行失败: %v", err)
		os.Exit(1)
	}
}

// 第一层Cobra菜单
func newRootCommand() *cobra.Command {
	var logToConsole bool

	rootCmd := &cobra.Command{
		Use:          "pgo",
		Short:        "PGO all-in-one client",
		Version:      buildInfo(),
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			initLogger(logToConsole)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			runInteractiveMenu()
			return nil
		},
	}

	rootCmd.PersistentFlags().BoolVarP(
		&logToConsole,
		"log-to-console",
		"l",
		true,
		"log to console",
	)
	for _, registration := range newCommandGroups() {
		rootCmd.AddCommand(registration.group.NewCobraCommand())
	}

	return rootCmd
}

// 第一层交互菜单
func runInteractiveMenu() {
	selector := pthird.Interact.NewSelector("请选择功能 (Select Function)")
	for _, registration := range newCommandGroups() {
		current := registration
		selector.Reg(current.label, func() {
			current.group.RunInteractive(current.title)
		})
	}
	selector.Loop()
}

type commandGroupRegistration struct {
	label string
	title string
	group *pclient.CommandGroup
}

func newCommandGroups() []commandGroupRegistration {
	devopsGroup := pclient.NewCommandGroup("devops", "Run DevOps tools")
	devops.RegisterTools(devopsGroup)

	performanceGroup := pclient.NewCommandGroup(
		"performance",
		"Run repeatable performance tests",
	)
	performance.RegisterTools(performanceGroup)

	applicationGroup := pclient.NewCommandGroup(
		"application",
		"Run application tools",
	)
	application.RegisterTools(applicationGroup)

	toolsGroup := pclient.NewCommandGroup("tools", "Run development tools")
	tools.RegisterTools(toolsGroup)

	return []commandGroupRegistration{
		{label: "DevOps", title: "DevOps", group: devopsGroup},
		{
			label: "性能测试 (Performance)",
			title: "性能测试 (Performance)",
			group: performanceGroup,
		},
		{
			label: "应用 (Application)",
			title: "应用 (Application)",
			group: applicationGroup,
		},
		{
			label: "开发工具 (Tools)",
			title: "开发工具 (Tools)",
			group: toolsGroup,
		},
	}
}
