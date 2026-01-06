package plogger

import (
	"log"
	"path"
	"path/filepath"
	"strings"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/putil"

	kLog "github.com/go-kratos/kratos/v2/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	log.SetFlags(log.Ltime | log.Lshortfile)
}

// --------------------------------------------------
var defaultLogger kLog.Logger
var defaultLogWarper *PLogWarper

func init() { //没有手动调用Init时，默认初始化一个控制台日志
	InitConsoleLogger()
}

func initDefaultLogger(zLogger *zap.Logger) {
	defaultLogger = kLog.With(FromZap(zLogger),
		"caller", kLog.Caller(4),
	)

	defaultLogWarper = NewPLogWarper(defaultLogger).AddCallerLevel(1)
}

func GetDefaultLogger() kLog.Logger {
	if defaultLogger == nil {
		InitConsoleLogger()
	}
	return defaultLogger
}

// 除了默认的初始化外，也可以修改后覆盖默认的logger
func SetDefaultLogger(l kLog.Logger) {
	defaultLogger = l
	defaultLogWarper = NewPLogWarper(defaultLogger).AddCallerLevel(1)
}

func GetDefaultLogWarper() *PLogWarper {
	if defaultLogWarper == nil {
		InitConsoleLogger()
	}
	return defaultLogWarper
}

// --------------------------------------------------
func InitFromConfig(isLogConsole bool) {
	level := pconfig.GetStringD("Log.Level", "debug")
	lv := StrToLoggerLevel(level)
	folder := pconfig.GetStringD("Log.Path", "")
	InitLogger(isLogConsole, lv, folder)
}

func InitConsoleLogger() {
	InitLogger(true, zap.DebugLevel, "")
}

func InitLogger(isLogConsole bool, lv zapcore.Level, logPath string) {
	fullPath := ""
	folderPath := ""

	execName := putil.GetExecName()

	if strings.HasSuffix(logPath, ".log") {
		folderPath = filepath.Dir(logPath)
		fullPath = logPath

	} else {
		folderPath = logPath
		if logPath == "" {
			folderPath = filepath.Join(putil.GetExecFolder(), "./logs/")
		}
		fileName := execName + "_" + "%Y%m%d.log"
		fullPath = path.Join(folderPath, fileName)
	}
	// fmt.Println("Log fullPath:", fullPath)

	//软连接名 LogName
	linkName := execName
	linkPath := path.Join(folderPath, linkName)

	zLogger := newZapLogger(isLogConsole, lv, fullPath, linkPath)

	// 将 zap.Logger 作为全局 logger
	zap.ReplaceGlobals(zLogger)
	// 重定向标准输出和错误输出
	zap.RedirectStdLog(zLogger)

	initDefaultLogger(zLogger)
	// Debugf("Logger initialized -------------------------------")
}
