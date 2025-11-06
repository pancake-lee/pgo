//go:build windows

package putil

import (
	"os/exec"
	"syscall"
)

func getBash() string {
	return `C:\Program Files\Git\git-bash.exe`
}

func execDefaultSetting(cmd *exec.Cmd) {
	// 更新环境变量，当程序自身安装了新的软件，改变了环境变量，手动刷新一下
	// 失败：其实不行的，cmd窗口自己都刷不到，要重启才行，cmd窗口启动的一个程序更加刷不到
	// cmd.Env = append(os.Environ(), "PATH="+os.Getenv("PATH"))

	// 不需要弹出新的cmd窗口
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

func ExecBG(command string) error {
	cmd := exec.Command(command)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	return cmd.Start()
}
