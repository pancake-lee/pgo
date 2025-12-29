package main

import (
	"fmt"

	"github.com/pancake-lee/pgo/client/courseSwap"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/spf13/cobra"
)

func main() {
	plogger.SetJsonLog(false)
	plogger.InitConsoleLogger()

	runApp()
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
	fmt.Println("请选择功能:")
	fmt.Println("1. 调课 (Course Swap)")
	// Add more functions here in the future

	var choice int
	fmt.Print("请输入选项 (默认1): ")
	_, err := fmt.Scanln(&choice)
	if err != nil {
		choice = 1 // Default
	}

	switch choice {
	case 1:
		courseSwap.CourseSwapCli()
	default:
		fmt.Println("无效选项，默认进入调课功能")
		courseSwap.CourseSwapCli()
	}
}
