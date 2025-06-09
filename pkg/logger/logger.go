package logger

import (
	"log"
	"path"
	"path/filepath"
	"pgo/pkg/config"
	"pgo/pkg/util"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	log.SetFlags(log.Ltime | log.Lshortfile)
}

var zapLogger *zap.SugaredLogger

var isInit bool

func InitServiceLogger(isLogConsole bool) {
	level := config.GetStringD("Log.Level", "debug")
	lv := GetLoggerLevel(level)
	folder := config.GetStringD("Log.Path", "")
	InitLogger(isLogConsole, lv, folder)
}

func InitLogger(isLogConsole bool,
	lv zapcore.Level, folder string) {
	logPath := filepath.Join(util.GetExecFolder(), "./logs/")
	if folder != "" {
		logPath = folder
	}

	logName := util.GetExecName()

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
}

// --------------------------------------------------
// 该文件包含的主要是实际业务代码中打印日志使用的方法封装

func Debug(args ...interface{}) {
	myLog(zapLogger.Debug, 2, nil, args...)
}

func Debugf(template string, args ...interface{}) {
	myLogf(zapLogger.Debugf, 2, nil, template, args...)
}

// --------------------------------------------------
func Info(args ...interface{}) {
	myLog(zapLogger.Info, 2, nil, args...)
}

func Infof(template string, args ...interface{}) {
	myLogf(zapLogger.Infof, 2, nil, template, args...)
}

// --------------------------------------------------
func Warn(args ...interface{}) {
	myLog(zapLogger.Warn, 2, nil, args...)
}

func Warnf(template string, args ...interface{}) {
	myLogf(zapLogger.Warnf, 2, nil, template, args...)
}

// --------------------------------------------------
func Error(args ...interface{}) {
	myLog(zapLogger.Error, 2, nil, args...)
}

func Errorf(template string, args ...interface{}) {
	myLogf(zapLogger.Errorf, 2, nil, template, args...)
}

// --------------------------------------------------
func Fatal(args ...interface{}) {
	myLog(zapLogger.Fatal, 2, nil, args...)
}

func Fatalf(template string, args ...interface{}) {
	myLogf(zapLogger.Fatalf, 2, nil, template, args...)
}

// --------------------------------------------------
// 可以指定打印出调用者的信息，0表示打印当前函数位置，1表示上一级调用位置
func Log(lv zapcore.Level, callerLevel int, prefix []interface{},
	args ...interface{}) {

	switch lv {
	case zapcore.InfoLevel:
		myLog(zapLogger.Info, callerLevel+2, prefix, args...)

	case zapcore.WarnLevel:
		myLog(zapLogger.Warn, callerLevel+2, prefix, args...)

	case zapcore.ErrorLevel:
		myLog(zapLogger.Error, callerLevel+2, prefix, args...)

	default:
		myLog(zapLogger.Debug, callerLevel+2, prefix, args...)
	}
}

// 可以指定打印出调用者的信息，0表示打印当前函数位置，1表示上一级调用位置
func Logf(lv zapcore.Level, callerLevel int, prefix []interface{},
	template string, args ...interface{}) {

	switch lv {
	case zapcore.InfoLevel:
		myLogf(zapLogger.Infof, callerLevel+2, prefix, template, args...)

	case zapcore.WarnLevel:
		myLogf(zapLogger.Warnf, callerLevel+2, prefix, template, args...)

	case zapcore.ErrorLevel:
		myLogf(zapLogger.Errorf, callerLevel+2, prefix, template, args...)

	default:
		myLogf(zapLogger.Debugf, callerLevel+2, prefix, template, args...)
	}
}

// --------------------------------------------------
func LogErrCode(errNo int32) int32 {
	if errNo != 0 {
		myLogf(zapLogger.Warnf, 2, nil, "return errNo[%v]", errNo)
	}
	return errNo
}

func LogErr(err error, errNo int32) int32 {
	myLogf(zapLogger.Warnf, 2, nil, "get err[%s], return errNo[%v]", err.Error(), errNo)
	return errNo
}
