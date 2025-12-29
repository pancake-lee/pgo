package putil

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/pterm/pterm"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// --------------------------------------------------
// 首先调用者可以不关注os，内部直接兼容win和linux两个os
// 已经实现以下细节
// 1：win，不要额外弹出命令行窗口
// 2：win，如果执行sh脚本，尝试用git-bash.exe执行
// 3：linux，如果执行sh脚本，用/bin/sh执行
// 4：废弃：应该外部自己用GetBash和-c参数来执行复杂命令，
// 	linux涉及符号'|'，则管道符号，则需要用/bin/sh -c执行
// 5：除了exec.Command原本的参数方式，额外封装一个只需要输入整个命令的方法，自动以空格切分参数
// 以上将封装出两个函数，在命令执行完之后，再直接输出string

// linux上支持使用管道，将用/bin/sh -c来执行
func ExecSplit(command string) (string, error) {
	// if strings.Contains(command, "|") {
	// 	return Exec(getBash(), "-c", command)
	// }

	// 和命令行一样，如果遇到'和"，则认为是字符串，不切分
	// 但是没有处理嵌套引号的情况

	var parts []string
	var currentPart []rune
	inSingleQuote := false
	inDoubleQuote := false

	for _, char := range command {
		switch char {
		case ' ':
			if inSingleQuote || inDoubleQuote {
				currentPart = append(currentPart, char)
			} else if len(currentPart) > 0 {
				parts = append(parts, string(currentPart))
				currentPart = nil
			}
		case '\'':
			if inDoubleQuote {
				currentPart = append(currentPart, char)
			} else {
				inSingleQuote = !inSingleQuote
				// currentPart = append(currentPart, char)//引号自己不需要
			}
		case '"':
			if inSingleQuote {
				currentPart = append(currentPart, char)
			} else {
				inDoubleQuote = !inDoubleQuote
				// currentPart = append(currentPart, char)//引号自己不需要
			}
		default:
			currentPart = append(currentPart, char)
		}
	}

	if len(currentPart) > 0 {
		parts = append(parts, string(currentPart))
	}
	if len(parts) == 0 {
		return "", errors.New("invalid command")
	}
	if len(parts) == 1 {
		return Exec(parts[0])
	}
	return Exec(parts[0], parts[1:]...)
}

func Exec(name string, args ...string) (string, error) {

	if strings.HasSuffix(name, ".sh") {
		// "a.sh b c" -> "/bin/sh a.sh b c"
		args = append([]string{name}, args...)
		name = getBash()
	}

	cmd := exec.Command(name, args...)
	execDefaultSetting(cmd)

	out, err := cmd.CombinedOutput()

	outStr := string(out)

	// 之前想过在这里自动尝试各种编码的转换，但是并不顺利，有些乱码也会符合utf8编码检测
	// 用out判断其实是不靠谱的，可能因为多语言，可能因为不同的编码，都无法准确判断，打印一下就算了

	// 	if runtime.GOOS == "windows" {
	// 	out2, err2 := StrToUTF8(outStr)
	// 	if err2 == nil {
	// 		outStr = out2
	// 	}
	// }

	return outStr, err
}

func Conv_gbk2utf8(s string) (string, error) {
	reader := transform.NewReader(strings.NewReader(s),
		simplifiedchinese.GBK.NewDecoder())

	// 读取转换后的内容
	utf8Bytes, err := io.ReadAll(reader)
	if err != nil {
		return s, err
	}
	return string(utf8Bytes), nil
}

func Conv_gb2312_utf8(s string) (string, error) {
	reader := transform.NewReader(strings.NewReader(s),
		simplifiedchinese.HZGB2312.NewDecoder())

	// 读取转换后的内容
	utf8Bytes, err := io.ReadAll(reader)
	if err != nil {
		return s, err
	}
	return string(utf8Bytes), nil
}

func Conv_utf16be_utf8(s string) (string, error) {
	reader := transform.NewReader(strings.NewReader(s),
		unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewDecoder())

	// 读取转换后的内容
	utf8Bytes, err := io.ReadAll(reader)
	if err != nil {
		return s, err
	}
	return string(utf8Bytes), nil
}

func Conv_utf16le_utf8(s string) (string, error) {
	reader := transform.NewReader(strings.NewReader(s),
		unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder())

	// 读取转换后的内容
	utf8Bytes, err := io.ReadAll(reader)
	if err != nil {
		return s, err
	}
	return string(utf8Bytes), nil
}

// --------------------------------------------------
// 命令行交互
type _interact struct{}

var Interact _interact

func (_interact) MustInput(msg string) (value string) {
	for i := 0; i < 3; i++ {
		value = Interact.Input(msg)
		if value != "" {
			Interact.Debugf("Your input: %v", value)
			break
		} else {
			Interact.Warnf("No input! retry[%v] ...", i+1)
		}
	}
	if value == "" {
		Interact.Warnf("No input! exit")
		os.Exit(1)
	}
	return value
}
func (_interact) Input(msg string) (value string) {
	value, _ = pterm.DefaultInteractiveTextInput.
		WithDefaultText(msg).Show()
	return value
}

func (_interact) MustConfirm(msg string) {
	result, _ := pterm.DefaultInteractiveConfirm.
		WithDefaultText(msg).WithDefaultValue(true).Show()
	if result {
		Interact.Debugf("Confirmed")
	} else {
		Interact.Warnf("Exited")
		os.Exit(1)
	}
}

func (_interact) Warnf(msg string, args ...interface{}) {
	pterm.Println(pterm.Yellow(fmt.Sprintf(msg, args...)))
}
func (_interact) Errorf(msg string, args ...interface{}) {
	pterm.Println(pterm.Red(fmt.Sprintf(msg, args...)))
}

func (_interact) Debugf(msg string, args ...interface{}) {
	pterm.Println(pterm.Cyan(fmt.Sprintf(msg, args...)))
}

func (_interact) Infof(msg string, args ...interface{}) {
	pterm.Printf(pterm.Green(fmt.Sprintf(msg, args...)))
}

func (_interact) PrintLine() {
	msg := "---------------------------------------------"
	pterm.Println(pterm.Cyan(msg))
}
