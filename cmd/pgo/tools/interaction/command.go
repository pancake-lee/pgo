package interaction

import (
	"github.com/pancake-lee/pgo/pkg/pclient"
	"github.com/pancake-lee/pgo/pkg/pthird"
	"github.com/spf13/cobra"
)

// Entrypoint exposes the interaction component test as a concrete tool.
var Entrypoint = pclient.NewCommandEntry(newCommand, runInteractive)

func newCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "interaction",
		Short: "Test interactive terminal components",
		Run: func(_ *cobra.Command, _ []string) {
			runInteractive()
		},
	}
}

func runInteractive() {
	pthird.Interact.PrintLine()
	pthird.Interact.Infof("开始交互组件测试 (Interactive Component Test)")

	pthird.Interact.Infof("测试日志样式 (Log Style):")
	pthird.Interact.Infof("  -> 这是 Info 消息 (Info Message)")
	pthird.Interact.Debugf("  -> 这是 Debug 消息 (Debug Message)")
	pthird.Interact.Warnf("  -> 这是 Warn 消息 (Warn Message)")
	pthird.Interact.Errorf("  -> 这是 Error 消息 (Error Message)")
	pthird.Interact.PrintLine()

	value := pthird.Interact.Input("测试普通输入 (Input - Optional): ")
	pthird.Interact.Infof("你输入了 (You input): %s", value)

	value = pthird.Interact.MustInput(
		"测试必填输入 (MustInput - Required): ",
	)
	pthird.Interact.Infof("你输入了 (You input): %s", value)

	pthird.Interact.PrintLine()
	pthird.Interact.Infof("即将测试确认框 (Confirm Test)")
	pthird.Interact.MustConfirm("确认继续吗? (Confirm to continue?)")
	pthird.Interact.Infof("已确认 (Confirmed)")

	pthird.Interact.PrintLine()
	pthird.Interact.Infof("即将测试多级选择器 (Nested Selector Test)")
	selector := pthird.Interact.NewSelector("请选择一种颜色 (Pick a color)")
	selector.Reg("红色 (Red)", func() {
		pthird.Interact.Infof("你选择了红色")
	})
	selector.Reg("蓝色 (Blue)", func() {
		pthird.Interact.Infof("你选择了蓝色")
	})
	selector.Loop()

	pthird.Interact.Infof("交互测试完成 (Interactive Test Completed)")
}
