//go:build windows

package pclient

import (
	"os"
	"syscall"
)

// RunApp is the platform-aware entrypoint for Windows
// - No args: runs UI mode
// - With args: runs CLI mode
func RunApp(cli func(), ui func()) {
	if len(os.Args) == 1 {
		ui()
		return
	}

	// 兼容历史模式: client.exe cli
	if os.Args[1] == "cli" {
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}

	cli()
}

func getExecAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true}
}
