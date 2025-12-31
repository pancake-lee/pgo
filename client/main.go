package main

import (
	"flag"
	"os"

	"github.com/pancake-lee/pgo/client/courseSwap"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

func main() {
	runApp()
}

func runCli() {
	l := flag.Bool("l", false, "log to console, default is false")
	flag.Parse()
	plogger.SetJsonLog(false)
	plogger.InitLogger(*l, zapcore.DebugLevel, "")

	if err := rootCmd.Execute(); err != nil {
		plogger.LogErr(err)
		os.Exit(1)
	}
}

// --------------------------------------------------
var rootCmd = &cobra.Command{
	Use:   "pgo-client",
	Short: "PGO Client Application",
	Long:  `PGO Client Application with CLI and UI support`,
	Run:   runCobra,
}

func runCobra(cmd *cobra.Command, args []string) {
	// CLI Interactive Mode
	// Select function
	putil.Interact.Infof("请选择功能:")
	putil.Interact.Infof("1. 调课 (Course Swap)")
	// Add more functions here in the future

	funcNumStr := "1"
	_input := putil.Interact.Input("请输入选项 (默认1): ")
	if _input != "" {
		funcNumStr = _input
	}
	choice, err := putil.StrToInt(funcNumStr)
	if err != nil {
		choice = 1 // Default
	}

	switch choice {
	case 1:
		courseSwap.CourseSwapCli()
	default:
		putil.Interact.Infof("无效选项，默认进入调课功能")
		courseSwap.CourseSwapCli()
	}
}
