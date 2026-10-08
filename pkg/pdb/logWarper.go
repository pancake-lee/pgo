package pdb

import (
	"context"
	"errors"
	"fmt"
	"time"

	kLog "github.com/go-kratos/kratos/v2/log"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"gorm.io/gorm"
	dbLogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

// gormLogger 仅在单次日志调用中绑定请求上下文，配置可被并发复用。
type gormLogger struct {
	config dbLogger.Config
}

var _ dbLogger.Interface = gormLogger{}

func newGormLogger(config dbLogger.Config) dbLogger.Interface {
	return gormLogger{config: config}
}

func (l gormLogger) LogMode(level dbLogger.LogLevel) dbLogger.Interface {
	l.config.LogLevel = level
	return l
}

func (l gormLogger) withContext(
	ctx context.Context, caller string,
) *plogger.PLogWarper {
	logger := kLog.With(plogger.GetDefaultLoggerNoCaller(), "caller", caller)
	return plogger.NewPLogWarper(logger).WithContext(ctx)
}

func (l gormLogger) Info(ctx context.Context, msg string, args ...any) {
	if l.config.LogLevel >= dbLogger.Info {
		l.withContext(ctx, utils.FileWithLineNum()).Infof(msg, args...)
	}
}

func (l gormLogger) Warn(ctx context.Context, msg string, args ...any) {
	if l.config.LogLevel >= dbLogger.Warn {
		l.withContext(ctx, utils.FileWithLineNum()).Infof(msg, args...)
	}
}

func (l gormLogger) Error(ctx context.Context, msg string, args ...any) {
	if l.config.LogLevel >= dbLogger.Error {
		l.withContext(ctx, utils.FileWithLineNum()).Infof(msg, args...)
	}
}

func (l gormLogger) Trace(
	ctx context.Context, begin time.Time,
	fc func() (string, int64), err error,
) {
	if l.config.LogLevel <= dbLogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	var prefix string
	switch {
	case err != nil && l.config.LogLevel >= dbLogger.Error &&
		(!errors.Is(err, gorm.ErrRecordNotFound) ||
			!l.config.IgnoreRecordNotFoundError):
		prefix = err.Error() + "\n"
	case l.config.LogLevel >= dbLogger.Warn &&
		l.config.SlowThreshold != 0 && elapsed > l.config.SlowThreshold:
		prefix = fmt.Sprintf("SLOW SQL >= %v\n", l.config.SlowThreshold)
	case l.config.LogLevel == dbLogger.Info:
	default:
		return
	}

	sql, rows := fc()
	var rowCount any = rows
	if rows == -1 {
		rowCount = "-"
	}
	// 直接从 Trace 获取业务位置，避免辅助函数成为第一个非 GORM 栈帧。
	// 延续原 Writer 的 Info 输出级别，仅将来源位置移至 caller 字段。
	l.withContext(ctx, utils.FileWithLineNum()).Infof(
		"%s[%.3fms] [rows:%v] %s", prefix,
		float64(elapsed.Nanoseconds())/1e6, rowCount, sql,
	)
}

// ParamsFilter 保留 GORM 参数化日志配置，避免适配后暴露查询参数。
func (l gormLogger) ParamsFilter(
	_ context.Context, sql string, params ...any,
) (string, []any) {
	if l.config.ParameterizedQueries {
		return sql, nil
	}
	return sql, params
}

// --------------------------------------------------
// canal模块需要一个指定的接口实现
type canalLogger struct {
	*plogger.PLogWarper
}

func newCanalLogger() *canalLogger {
	return &canalLogger{
		PLogWarper: plogger.GetDefaultLogWarper(),
	}
}

func (l *canalLogger) Print(args ...interface{}) {
	l.Debug(args...)
}

func (l *canalLogger) Printf(msg string, args ...interface{}) {
	l.Debugf(msg, args...)
}
func (l *canalLogger) Println(args ...interface{}) {
	l.Debug(args...)
}

func (l *canalLogger) Debugln(args ...interface{}) {
	l.Debug(args...)
}
func (l *canalLogger) Infoln(args ...interface{}) {
	l.Info(args...)
}
func (l *canalLogger) Warnln(args ...interface{}) {
	l.Warn(args...)
}
func (l *canalLogger) Errorln(args ...interface{}) {
	l.Error(args...)
}
func (l *canalLogger) Fatalln(args ...interface{}) {
	l.Fatal(args...)
}
func (l *canalLogger) Panic(args ...interface{}) {
	l.Fatal(args...)
	panic("canal panic")
}
func (l *canalLogger) Panicf(msg string, args ...interface{}) {
	l.Fatalf(msg, args...)
	panic("canal panic")
}

func (l *canalLogger) Panicln(args ...interface{}) {
	l.Fatal(args...)
	panic("canal panic")
}
