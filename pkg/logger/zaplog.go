package logger

import (
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"pgo/pkg/config"
	"pgo/pkg/util"
	"runtime"
	"strings"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// 业务日志 logger
var errorLogger *zap.SugaredLogger

// 接口req/resp日志 logger
var apiLogger *zap.SugaredLogger

// 为了让go程序的core信息输出到nohup，所以程序的日志就不能输出到nohup导致篇幅太大
// isLogConsole改成通过命令行参数输入
// 因此，不能依赖import时的init，需要自己解析到命令行参数后手动初始化日志摸块
var isInit bool

func InitLogger(isLogConsole bool) {
	logLevel := "debug"
	logPath := filepath.Join(util.GetExecFolder(), "./logs/")

	l, err := config.GetStringE("Log.LogLevel")
	if err == nil && l != "" {
		logLevel = l
	}
	p, err := config.GetStringE("Log.LogPath")
	if err == nil && p != "" {
		logPath = p
	}

	logName := util.GetExecName()
	level := getLoggerLevel(logLevel)

	//日志文件名 LogName_pid_20200702.log
	fileName := logName + "_" + util.IntToStr(os.Getpid()) + "_" + "%Y%m%d.log"
	fullPath := path.Join(logPath, fileName)

	//软连接名 LogName
	linkName := logName
	linkPath := path.Join(logPath, linkName)

	zLogger := newZapLogger(isLogConsole, level, fullPath, linkPath)
	errorLogger = zLogger.Sugar()

	//使用同一个ZapLog对象，提供kratos的日志接口，这样kratos底层日志就能打印到我们自己的日志文件中
	initKratosLogger(errorLogger)

	isInit = true
	// Infof("logger init finish, isConsole[%v]", isLogConsole)
	if !isLogConsole {
		log.Printf("logger init finish, isConsole[%v], log file [%v]\n", isLogConsole, fullPath)

		// 重定向 stdout 和 stderr 到 NUL
		// os.Stdout, _ = os.OpenFile(logPath+"/std.log", os.O_WRONLY|os.O_CREATE, 0)
		// os.Stderr, _ = os.OpenFile(logPath+"/std.log", os.O_WRONLY|os.O_CREATE, 0)

		// 将 zap.Logger 作为全局 logger
		zap.ReplaceGlobals(zLogger)
		// 重定向标准输出和错误输出
		zap.RedirectStdLog(zLogger)

	}

	if !isLogConsole { //调试程序时，并且打印控制台时，不打印这份东西
		//日志文件名 LogName_pid_20200702_api.log
		fileName := logName + "_" + util.IntToStr(os.Getpid()) + "_" + "%Y%m%d_api.log"
		fullPath := path.Join(logPath, fileName)

		//软连接名 LogName_api
		linkName := logName + "_api"
		linkPath := path.Join(logPath, linkName)

		zLogger := newZapLogger(isLogConsole, level, fullPath, linkPath)
		apiLogger = zLogger.Sugar()
	}
}

// -----------------------------------------------------------------------
// 实现我们自己的日志格式
// 1：caller后置，方便message位置对齐
// 2：支持输入callerSkip，则调用栈往上层跳跃，方便封装的函数调用时，打印调用者的位置
func myLog(logFunc func(as ...interface{}),
	callerSkip int, prefix []interface{}, args ...interface{}) {

	tmpArgs := make([]interface{}, 0, len(args)+2+len(prefix)*4+5)
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
		log.Print(tmpArgs...)
		return
	}
	logFunc(tmpArgs...)
}

func myLogf(logFunc func(t string, as ...interface{}),
	callerSkip int, prefix []interface{}, template string, args ...interface{}) {

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

//------------------------------------------------------------
//关于如何构建ZapLog参数的一些封装

var levelMap = map[string]zapcore.Level{
	"debug":  zapcore.DebugLevel,
	"info":   zapcore.InfoLevel,
	"warn":   zapcore.WarnLevel,
	"error":  zapcore.ErrorLevel,
	"dpanic": zapcore.DPanicLevel,
	"panic":  zapcore.PanicLevel,
	"fatal":  zapcore.FatalLevel,
}

func getLoggerLevel(lvl string) zapcore.Level {
	if level, ok := levelMap[lvl]; ok {
		return level
	}
	return zapcore.InfoLevel
}

func timeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("060102 15:04:05.000"))
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
		EncodeLevel:    zapcore.CapitalLevelEncoder,
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

	return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
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

	//日志文件名 LogName_pid_20200702.log
	hook, err := rotatelogs.New(filePath, optList...)
	if err != nil {
		log.Print(err)
	}

	return hook
}

//------------------------------------------------------------
