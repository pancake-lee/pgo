package putil

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

// pwd = Print Working Directory = 获取当前工作目录
// 类似的，我们还会获取，当前可执行程序名，当前函数名，当前行号等
// 这种“定位当前工作位置”的功能都归类到本文件中

// 获取当前程序/可执行文件，所在的绝对路径
func GetExecPath() string {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Println("Error:", err)
		return ""
	}
	return exePath
}

func GetExecName() (n string) {
	execName := os.Args[0]

	var index int
	if runtime.GOOS == "windows" {
		index = strings.LastIndex(execName, "\\")
		if index == -1 {
			n = execName
		}
		n = strings.TrimSuffix(execName[index+1:], ".exe")

	} else {
		index = strings.LastIndex(execName, "/")
		if index != -1 {
			n = execName
		}
		n = execName[index+1:]
	}

	return n
}

// 获取当前程序/可执行文件，所在的文件夹，的绝对路径
func GetExecFolder() string {
	return filepath.Dir(GetExecPath())
}

// os.Getwd() 获取当前工作目录
func GetCurDir() string {
	currentDir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return currentDir
}

func GetCallerFuncName(skip int) string {
	pc, _, _, ok := runtime.Caller(skip + 1)
	if !ok {
		return ""
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return ""
	}

	// 获取完整的函数名（包含包路径）
	fullName := fn.Name()
	return path.Base(fullName)
}

func GetFuncName(i any) string {
	// 获取函数名称
	fn := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	// 用 sep 进行分割，不要文件名，也不要类名
	sepList := []rune{'.', '/', '\\'}
	fields := strings.FieldsFunc(fn, func(sep rune) bool {
		for _, s := range sepList {
			if sep == s {
				return true
			}
		}
		return false
	})
	if size := len(fields); size > 0 {
		// 取最后一个段
		name := fields[size-1]
		// 如果注册的方法是一个单例的方法，则会带有-fm后缀
		nameList := strings.Split(name, "-")
		return nameList[0]
	}
	return ""
}
