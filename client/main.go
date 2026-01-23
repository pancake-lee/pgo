package main

import (
	"flag"

	"github.com/pancake-lee/pgo/client/courseSwap"
	"github.com/pancake-lee/pgo/client/devops"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
	"go.uber.org/zap/zapcore"
)

func main() {
	runApp()
}

func runCli() {
	logToConsole := flag.Bool("l", false, "log to console")
	flag.Parse()

	// 初始化日志
	plogger.SetJsonLog(false)
	plogger.InitLogger(*logToConsole, zapcore.DebugLevel, "./logs/")

	// --------------------------------------------------
	sel := putil.Interact.NewSelector("请选择功能 (Select Function)")
	sel.Reg("Devops CI", devops.MakeCli)
	sel.Reg("Devops CD", devops.DeployCli)

	sel.Reg("调课 (Course Swap)", courseSwap.CourseSwapCli)

	sel.Reg("测试交互 (Test Interact)", testInteraction)

	sel.Loop()
}

func testInteraction() {
	putil.Interact.PrintLine()
	putil.Interact.Infof("开始交互组件测试 (Interactive Component Test)")

	// 1. Log Style
	putil.Interact.Infof("测试日志样式 (Log Style):")
	putil.Interact.Infof("  -> 这是 Info 消息 (Info Message)")
	putil.Interact.Debugf("  -> 这是 Debug 消息 (Debug Message)")
	putil.Interact.Warnf("  -> 这是 Warn 消息 (Warn Message)")
	putil.Interact.Errorf("  -> 这是 Error 消息 (Error Message)")
	putil.Interact.PrintLine()

	// 2. Input
	val := putil.Interact.Input("测试普通输入 (Input - Optional): ")
	putil.Interact.Infof("你输入了 (You input): %s", val)

	// 3. MustInput
	val = putil.Interact.MustInput("测试必填输入 (MustInput - Required): ")
	putil.Interact.Infof("你输入了 (You input): %s", val)

	// 4. MustConfirm
	putil.Interact.PrintLine()
	putil.Interact.Infof("即将测试确认框 (Confirm Test)")
	// 注意，如果用户选 No，MustConfirm 会 os.Exit(1)，所以这里仅仅是测试 Confirm 流程
	putil.Interact.MustConfirm("确认继续吗? (Confirm to continue?)")
	putil.Interact.Infof("已确认 (Confirmed)")

	// 5. Selector (Nested)
	putil.Interact.PrintLine()
	putil.Interact.Infof("即将测试多级选择器 (Nested Selector Test)")
	s := putil.Interact.NewSelector("请选择一种颜色 (Pick a color)")
	s.Reg("红色 (Red)", func() { putil.Interact.Infof("你选择了红色") })
	s.Reg("蓝色 (Blue)", func() { putil.Interact.Infof("你选择了蓝色") })
	s.Loop()

	putil.Interact.Infof("交互测试完成 (Test Completed)")
}
