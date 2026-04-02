//go:build !windows

package pclient

import "syscall"

// RunApp is the platform-aware entrypoint
// - No args on Windows: runs UI mode
// - No args on non-Windows: runs CLI (interactive menu)
// - With args: always runs CLI mode
func RunApp(cli func(), ui func()) {
	cli()
}

func GetExecAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
