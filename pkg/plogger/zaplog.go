package plogger

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logLevel     *zap.AtomicLevel
	origLogLevel string
	resetTimer   *time.Timer
	isJsonLog    bool = true
)

func SetJsonLog(isJson bool) {
	isJsonLog = isJson
}

func GetLoggerLevel() zapcore.Level {
	if logLevel == nil {
		return zap.InfoLevel
	}
	return logLevel.Level()
}

// 临时设置日志级别，10分钟后恢复
func SetLoggerLevel(lv string) {
	if logLevel == nil {
		return
	}

	Errorf("SetLoggerLevel: %s, origLogLevel: %s", lv, origLogLevel)
	logLevel.SetLevel(StrToLoggerLevel(lv))

	// 只能临时修改10分钟，如果需要长期修改，应该修改配置文件，然后重启程序
	if resetTimer != nil {
		resetTimer.Stop()
	}
	resetTimer = time.AfterFunc(10*time.Minute, func() {
		// 恢复为配置文件中的日志级别
		if origLogLevel != "" {
			Errorf("reset origLogLevel: %s", origLogLevel)
			logLevel.SetLevel(StrToLoggerLevel(origLogLevel))
		}
	})
}

func newZapLogger(isLogConsole bool, level zapcore.Level, fullPath, linkPath string) *zap.Logger {

	logConf := zapcore.EncoderConfig{
		TimeKey:        "T",
		LevelKey:       "L",
		NameKey:        "N",
		MessageKey:     "M",
		StacktraceKey:  "S",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    levelEncoder, // zapcore.CapitalLevelEncoder
		EncodeTime:     timeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}

	var encoder zapcore.Encoder

	if isJsonLog {
		encoder = zapcore.NewJSONEncoder(logConf)
	} else {
		encoder = zapcore.NewConsoleEncoder(logConf)
	}

	syncWriter := getWriter(fullPath, linkPath)

	l := zap.NewAtomicLevelAt(level)
	logLevel = &l

	var core zapcore.Core
	if isLogConsole {
		core = zapcore.NewCore(
			encoder,
			zapcore.AddSync(os.Stdout),
			logLevel,
		)
	} else {
		var coreList []zapcore.Core // 支持多个core

		// var wsList []zapcore.WriteSyncer // 一个core支持多个writer
		// wsList = append(wsList, zapcore.AddSync(syncWriter))
		core = zapcore.NewCore(
			encoder,
			zapcore.AddSync(syncWriter),
			// zapcore.NewMultiWriteSyncer(wsList...),
			logLevel,
		)
		coreList = append(coreList, core)

		graylogCore := NewGraylogCore("")
		if graylogCore != nil {
			coreList = append(coreList, graylogCore)
		}
		core = zapcore.NewTee(coreList...)
	}

	ret := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	zap.ReplaceGlobals(ret) // 将 zap.Logger 作为全局 logger
	zap.RedirectStdLog(ret) // 重定向标准输出和错误输出
	return ret
}

// --------------------------------------------------
var levelMap = map[string]zapcore.Level{
	"debug":  zapcore.DebugLevel,
	"info":   zapcore.InfoLevel,
	"warn":   zapcore.WarnLevel,
	"error":  zapcore.ErrorLevel,
	"dpanic": zapcore.DPanicLevel,
	"panic":  zapcore.PanicLevel,
	"fatal":  zapcore.FatalLevel,
}

func StrToLoggerLevel(lvl string) zapcore.Level {
	if level, ok := levelMap[lvl]; ok {
		return level
	}
	return zapcore.InfoLevel
}

// --------------------------------------------------
func timeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("060102 15:04:05.000"))
}

// 自定义级别显示
func levelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	switch level {
	case zapcore.DebugLevel:
		enc.AppendString("D")
	case zapcore.InfoLevel:
		enc.AppendString("I")
	case zapcore.WarnLevel:
		enc.AppendString("W")
	case zapcore.ErrorLevel:
		enc.AppendString("E")
	case zapcore.FatalLevel:
		enc.AppendString("F")
	default:
		enc.AppendString(fmt.Sprintf("%d", level))
	}
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
