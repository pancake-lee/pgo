package logger

import (
	"log"

	"go.uber.org/zap/zapcore"
)

func init() {
	log.SetFlags(log.Ltime | log.Lshortfile)
}

// 该文件包含的主要是实际业务代码中打印日志使用的方法封装

func Debug(args ...interface{}) {
	myLog(errorLogger.Debug, 2, nil, args...)
}

func Debugf(template string, args ...interface{}) {
	myLogf(errorLogger.Debugf, 2, nil, template, args...)
}

// --------------------------------------------------
func Info(args ...interface{}) {
	myLog(errorLogger.Info, 2, nil, args...)
}

func Infof(template string, args ...interface{}) {
	myLogf(errorLogger.Infof, 2, nil, template, args...)
}

// --------------------------------------------------
func Warn(args ...interface{}) {
	myLog(errorLogger.Warn, 2, nil, args...)
}

func Warnf(template string, args ...interface{}) {
	myLogf(errorLogger.Warnf, 2, nil, template, args...)
}

// --------------------------------------------------
func Error(args ...interface{}) {
	myLog(errorLogger.Error, 2, nil, args...)
}

func Errorf(template string, args ...interface{}) {
	myLogf(errorLogger.Errorf, 2, nil, template, args...)
}

// --------------------------------------------------
// 可以指定打印出调用者的信息，0表示打印当前函数位置，1表示上一级调用位置
func Log(lv zapcore.Level, callerLevel int, prefix []interface{},
	args ...interface{}) {

	switch lv {
	case zapcore.InfoLevel:
		myLog(errorLogger.Info, callerLevel+2, prefix, args...)

	case zapcore.WarnLevel:
		myLog(errorLogger.Warn, callerLevel+2, prefix, args...)

	case zapcore.ErrorLevel:
		myLog(errorLogger.Error, callerLevel+2, prefix, args...)

	default:
		myLog(errorLogger.Debug, callerLevel+2, prefix, args...)
	}
}

// 可以指定打印出调用者的信息，0表示打印当前函数位置，1表示上一级调用位置
func Logf(lv zapcore.Level, callerLevel int, prefix []interface{},
	template string, args ...interface{}) {

	switch lv {
	case zapcore.InfoLevel:
		myLogf(errorLogger.Infof, callerLevel+2, prefix, template, args...)

	case zapcore.WarnLevel:
		myLogf(errorLogger.Warnf, callerLevel+2, prefix, template, args...)

	case zapcore.ErrorLevel:
		myLogf(errorLogger.Errorf, callerLevel+2, prefix, template, args...)

	default:
		myLogf(errorLogger.Debugf, callerLevel+2, prefix, template, args...)
	}
}

// --------------------------------------------------
func LogErrCode(errNo int32) int32 {
	if errNo != 0 {
		myLogf(errorLogger.Warnf, 2, nil, "return errNo[%v]", errNo)
	}
	return errNo
}

func LogErr(err error, errNo int32) int32 {
	myLogf(errorLogger.Warnf, 2, nil, "get err[%s], return errNo[%v]", err.Error(), errNo)
	return errNo
}
