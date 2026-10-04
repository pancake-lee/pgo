//go:build windows

package pclient

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/pterm/pterm"
	"golang.org/x/sys/windows"
)

// cliChildEnv 标识已继承有效标准句柄的 CLI 子进程。
const cliChildEnv = "PGO_INTERNAL_CLI_CHILD"

// RunApp 启动 GUI，或建立控制台并等待继承标准句柄的 CLI 子进程。
func RunApp(cli func(), ui func()) {
	isChild := os.Getenv(cliChildEnv) == "1"
	if !isChild && len(os.Args) == 1 {
		ui()
		return
	}

	// 兼容历史模式：client.exe cli。
	if len(os.Args) > 1 && os.Args[1] == "cli" {
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}

	err := initCLIConsole()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "初始化 CLI 控制台失败: %v\n", err)
		os.Exit(1)
	}
	if isChild {
		// 后续启动其他 pgo 命令时，仍走正常的模式选择入口。
		_ = os.Unsetenv(cliChildEnv)
		cli()
		return
	}
	err = runCLIChild()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		_, _ = fmt.Fprintf(os.Stderr, "启动 CLI 子进程失败: %v\n", err)
		os.Exit(1)
	}
}

// runCLIChild 在库初始化前传入有效标准句柄，并等待子进程退出。
func runCLIChild() error {
	path, err := os.Executable()
	if err != nil {
		return err
	}
	command := exec.Command(path, os.Args[1:]...)
	command.Env = append(os.Environ(), cliChildEnv+"=1")
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

// initCLIConsole 附着父控制台或创建新控制台，恢复 Go 与菜单库的输入输出。
func initCLIConsole() error {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	attach := kernel32.NewProc("AttachConsole")
	result, _, err := attach.Call(uintptr(^uint32(0)))
	if result == 0 && err != windows.ERROR_ACCESS_DENIED {
		if err != windows.ERROR_INVALID_HANDLE {
			return fmt.Errorf("AttachConsole: %w", err)
		}
		// 父进程没有控制台时，为显式 CLI 调用创建控制台。
		allocate := kernel32.NewProc("AllocConsole")
		result, _, err = allocate.Call()
		if result == 0 {
			return fmt.Errorf("AllocConsole: %w", err)
		}
	}

	os.Stdin, err = restoreConsoleStream(os.Stdin,
		windows.STD_INPUT_HANDLE, "CONIN$", os.O_RDWR)
	if err != nil {
		return err
	}
	os.Stdout, err = restoreConsoleStream(os.Stdout,
		windows.STD_OUTPUT_HANDLE, "CONOUT$", os.O_RDWR)
	if err != nil {
		return err
	}
	os.Stderr, err = restoreConsoleStream(os.Stderr,
		windows.STD_ERROR_HANDLE, "CONOUT$", os.O_RDWR)
	if err != nil {
		return err
	}

	// 键盘库使用 syscall 的启动时句柄，需与新控制台同步。
	syscall.Stdin = syscall.Handle(os.Stdin.Fd())
	syscall.Stdout = syscall.Handle(os.Stdout.Fd())
	syscall.Stderr = syscall.Handle(os.Stderr.Fd())
	windows.Stdin = windows.Handle(os.Stdin.Fd())
	windows.Stdout = windows.Handle(os.Stdout.Fd())
	windows.Stderr = windows.Handle(os.Stderr.Fd())
	pterm.SetDefaultOutput(os.Stdout)
	return nil
}

// restoreConsoleStream 保留有效的继承或重定向流，否则打开控制台设备。
func restoreConsoleStream(file *os.File, standard uint32,
	device string, flag int,
) (*os.File, error) {
	valid := false
	if file != nil {
		_, err := windows.GetFileType(windows.Handle(file.Fd()))
		valid = err == nil && file.Fd() != 0 &&
			windows.Handle(file.Fd()) != windows.InvalidHandle
	}
	if !valid {
		var err error
		file, err = os.OpenFile(device, flag, 0)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", device, err)
		}
	}
	err := windows.SetStdHandle(standard, windows.Handle(file.Fd()))
	if err != nil {
		if !valid {
			_ = file.Close()
		}
		return nil, fmt.Errorf("set %s standard handle: %w", device, err)
	}
	return file, nil
}
