package plogger

import (
	"log"
	"path"
	"path/filepath"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/putil"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	log.SetFlags(log.Ltime | log.Lshortfile)
}

var zapLogger *zap.SugaredLogger

var isInit bool

func InitServiceLogger(isLogConsole bool) {
	level := pconfig.GetStringD("Log.Level", "debug")
	lv := GetLoggerLevel(level)
	folder := pconfig.GetStringD("Log.Path", "")
	InitLogger(isLogConsole, lv, folder)
}

func InitConsoleLogger() {
	InitLogger(true, zap.DebugLevel, "")
}

func InitLogger(isLogConsole bool,
	lv zapcore.Level, folder string) {
	logPath := filepath.Join(putil.GetExecFolder(), "./logs/")
	if folder != "" {
		logPath = folder
	}

	logName := putil.GetExecName()

	fileName := logName + "_" + "%Y%m%d.log"
	fullPath := path.Join(logPath, fileName)

	//软连接名 LogName
	linkName := logName
	linkPath := path.Join(logPath, linkName)

	zLogger := newZapLogger(isLogConsole, lv, fullPath, linkPath)
	zapLogger = zLogger.Sugar()

	//使用同一个ZapLog对象，提供kratos的日志接口，这样kratos底层日志就能打印到我们自己的日志文件中
	initKratosLogger(zapLogger)

	isInit = true
	if !isLogConsole {
		// 将 zap.Logger 作为全局 logger
		zap.ReplaceGlobals(zLogger)
		// 重定向标准输出和错误输出
		zap.RedirectStdLog(zLogger)
	}
	Info("Logger initialized -------------------------------")
}

// --------------------------------------------------
// 以下主要是实际业务代码中打印日志使用的方法封装

func LogErr(err error) error {
	myLogf(zapLogger.Warnf, 2, nil, "got err[%s]", err.Error())
	return err
}

// --------------------------------------------------
func Debug(args ...any) {
	myLog(zapLogger.Debug, 2, nil, args...)
}

func Debugf(template string, args ...any) {
	myLogf(zapLogger.Debugf, 2, nil, template, args...)
}

// --------------------------------------------------
func Info(args ...any) {
	myLog(zapLogger.Info, 2, nil, args...)
}

func Infof(template string, args ...any) {
	myLogf(zapLogger.Infof, 2, nil, template, args...)
}

// --------------------------------------------------
func Warn(args ...any) {
	myLog(zapLogger.Warn, 2, nil, args...)
}

func Warnf(template string, args ...any) {
	myLogf(zapLogger.Warnf, 2, nil, template, args...)
}

// --------------------------------------------------
func Error(args ...any) {
	myLog(zapLogger.Error, 2, nil, args...)
}

func Errorf(template string, args ...any) {
	myLogf(zapLogger.Errorf, 2, nil, template, args...)
}

// --------------------------------------------------
func Fatal(args ...any) {
	myLog(zapLogger.Fatal, 2, nil, args...)
}

func Fatalf(template string, args ...any) {
	myLogf(zapLogger.Fatalf, 2, nil, template, args...)
}

// --------------------------------------------------
// 可以指定打印出调用者的信息，0表示打印当前函数位置，1表示上一级调用位置
func Log(lv zapcore.Level, callerLevel int, prefix []any, args ...any) {
	logFunc := func(args ...any) {
		zapLogger.Log(lv, args...)
	}
	myLog(logFunc, callerLevel+2, prefix, args...)
}

// 可以指定打印出调用者的信息，0表示打印当前函数位置，1表示上一级调用位置
func Logf(lv zapcore.Level, callerLevel int, prefix []any,
	template string, args ...any) {

	logfFunc := func(template string, args ...any) {
		zapLogger.Logf(lv, template, args...)
	}
	myLogf(logfFunc, callerLevel+2, prefix, template, args...)
}
