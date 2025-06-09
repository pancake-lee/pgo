package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"pgo/pkg/util"
	"runtime"
	"strings"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// 实现我们自己的日志格式
// 1：caller后置，方便message位置对齐
// 2：支持输入callerSkip，则调用栈往上层跳跃，方便封装的函数调用时，打印调用者的位置
func myLog(logFunc func(as ...any),
	callerSkip int, prefix []any, args ...any) {

	tmpArgs := make([]any, 0, len(args)+2+len(prefix)*4+5)
	if len(prefix) > 0 {
		tmpArgs = append(tmpArgs, "[")
		isFirst := true
		for _, v := range prefix {
			if isFirst {
				isFirst = false
			} else {
				tmpArgs = append(tmpArgs, " ")
			}
			tmpArgs = append(tmpArgs, v)
		}
		tmpArgs = append(tmpArgs, "] ")
	}

	tmpArgs = append(tmpArgs, args...)

	_, file, line, _ := runtime.Caller(callerSkip)
	index := strings.LastIndex(file, "/")
	tmpArgs = append(tmpArgs, " [", file[index+1:], ":", line, "]")

	if !isInit {
		log.Println(tmpArgs...)
		return
	}
	logFunc(tmpArgs...)
}

func myLogf(logFunc func(t string, as ...any),
	callerSkip int, prefix []any, template string, args ...any) {

	var sb strings.Builder
	if len(prefix) > 0 {
		sb.WriteString("[")
		isFirst := true
		for _, v := range prefix {
			if isFirst {
				isFirst = false
			} else {
				sb.WriteString(" ")
			}

			vs := util.InterfaceToString(v, "")
			sb.WriteString(vs)
		}
		sb.WriteString("] ")
	}

	sb.WriteString(template)

	_, file, line, _ := runtime.Caller(callerSkip)
	index := strings.LastIndex(file, "/")
	fileName := file[index+1:]

	sb.WriteString(" ")
	sb.WriteString("[")
	sb.WriteString(fileName)
	sb.WriteString(":")
	sb.WriteString(util.IntToStr(line))
	sb.WriteString("]")

	if !isInit {
		log.Printf(sb.String(), args...)
		return
	}
	logFunc(sb.String(), args...)
}

var levelMap = map[string]zapcore.Level{
	"debug":  zapcore.DebugLevel,
	"info":   zapcore.InfoLevel,
	"warn":   zapcore.WarnLevel,
	"error":  zapcore.ErrorLevel,
	"dpanic": zapcore.DPanicLevel,
	"panic":  zapcore.PanicLevel,
	"fatal":  zapcore.FatalLevel,
}

func GetLoggerLevel(lvl string) zapcore.Level {
	if level, ok := levelMap[lvl]; ok {
		return level
	}
	return zapcore.InfoLevel
}

func timeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("060102 15:04:05.000"))
}

// 自定义级别显示
func levelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	switch level {
	case zapcore.DebugLevel:
		enc.AppendString("[D]")
	case zapcore.InfoLevel:
		enc.AppendString("[I]")
	case zapcore.WarnLevel:
		enc.AppendString("[W]")
	case zapcore.ErrorLevel:
		enc.AppendString("[E]")
	case zapcore.FatalLevel:
		enc.AppendString("[F]")
	default:
		enc.AppendString(fmt.Sprintf("[%d]", level))
	}
}

func newZapLogger(isLogConsole bool, level zapcore.Level, fullPath, linkPath string) *zap.Logger {
	encoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		// encoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:        "T",
		LevelKey:       "L",
		NameKey:        "N",
		MessageKey:     "M",
		StacktraceKey:  "S",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    levelEncoder, // zapcore.CapitalLevelEncoder
		EncodeTime:     timeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	})
	syncWriter := getWriter(fullPath, linkPath)

	var core zapcore.Core
	if isLogConsole {
		core = zapcore.NewCore(
			encoder,
			zapcore.AddSync(os.Stdout),
			zap.NewAtomicLevelAt(level),
		)
	} else {
		var wsList []zapcore.WriteSyncer
		wsList = append(wsList, zapcore.AddSync(syncWriter))

		core = zapcore.NewCore(
			encoder,
			zapcore.NewMultiWriteSyncer(wsList...),
			zap.NewAtomicLevelAt(level),
		)
	}

	ret := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	zap.ReplaceGlobals(ret) // 将 zap.Logger 作为全局 logger
	zap.RedirectStdLog(ret) // 重定向标准输出和错误输出
	return ret
}

func getWriter(filePath, linkPath string) io.Writer {
	if runtime.GOOS == "windows" {
		linkPath = ""
	}

	var optList []rotatelogs.Option
	if linkPath != "" {
		optList = append(optList, rotatelogs.WithLinkName(linkPath)) // 是否为日志文件建立软连接
	}

	//rotatelogs.WithMaxAge(-1), // 日志文件保留时间。默认保留7天
	//rotatelogs.WithRotationTime(time.Hour), // 多久切换一次日志文件，默认24小时
	//rotatelogs.WithRotationSize(20 * 1024 * 1024), //bytes

	hook, err := rotatelogs.New(filePath, optList...)
	if err != nil {
		log.Println(err)
	}

	return hook
}
