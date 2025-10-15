package plogger

import (
	"context"
	"fmt"

	kLogger "github.com/go-kratos/kratos/v2/log"
)

type PLogWarper struct {
	kLog kLogger.Logger
}

func NewPLogWarper(kLog kLogger.Logger) *PLogWarper {
	return &PLogWarper{kLog: kLog}
}

func (l *PLogWarper) WithContext(ctx context.Context) *PLogWarper {
	return NewPLogWarper(kLogger.WithContext(ctx, l.kLog))
}

func (l *PLogWarper) AddCallerLevel(callerLevel int) *PLogWarper {
	return NewPLogWarper(kLogger.With(l.kLog,
		"caller", kLogger.Caller(callerLevel+4), //改写caller的取值
	))
}
func (l *PLogWarper) GetLogger() kLogger.Logger {
	return l.kLog
}

// --------------------------------------------------
func (l *PLogWarper) LogErr(err error) error {
	if err == nil {
		return nil
	}
	l.kLog.Log(kLogger.LevelError, "msg", fmt.Sprintf("got err[%s]", err.Error()))
	return err
}
func LogErr(err error) error {
	return defaultLogWarper.LogErr(err)
}

// 务必定义0表示成功
func (l *PLogWarper) LogErrNo(errNo int32) int32 {
	if errNo == 0 {
		return 0
	}
	l.kLog.Log(kLogger.LevelError, "msg", fmt.Sprintf("got err[%d]", errNo))
	return errNo
}
func LogErrNo(errNo int32) int32 {
	return defaultLogWarper.LogErrNo(errNo)
}

func (l *PLogWarper) LogErrToErrNo(err error, errNo int32) int32 {
	if err == nil {
		return 0
	}
	l.kLog.Log(kLogger.LevelError, "msg", fmt.Sprintf("got err[%s] ret[%v]", err.Error(), errNo))
	return errNo
}
func LogErrToErrNo(err error, errNo int32) int32 {
	return defaultLogWarper.LogErrToErrNo(err, errNo)
}

// --------------------------------------------------
func (l *PLogWarper) Debug(args ...any) {
	l.kLog.Log(kLogger.LevelDebug, "msg", fmt.Sprint(args...))
}
func Debug(args ...any) {
	defaultLogWarper.Debug(args...)
}

func (l *PLogWarper) Debugf(template string, args ...any) {
	l.kLog.Log(kLogger.LevelDebug, "msg", fmt.Sprintf(template, args...))
}
func Debugf(template string, args ...any) {
	defaultLogWarper.Debugf(template, args...)
}

// --------------------------------------------------
func (l *PLogWarper) Info(args ...any) {
	l.kLog.Log(kLogger.LevelInfo, "msg", fmt.Sprint(args...))
}
func Info(args ...any) {
	defaultLogWarper.Info(args...)
}

func (l *PLogWarper) Infof(template string, args ...any) {
	l.kLog.Log(kLogger.LevelInfo, "msg", fmt.Sprintf(template, args...))
}
func Infof(template string, args ...any) {
	defaultLogWarper.Infof(template, args...)
}

// --------------------------------------------------
func (l *PLogWarper) Warn(args ...any) {
	l.kLog.Log(kLogger.LevelWarn, "msg", fmt.Sprint(args...))
}
func Warn(args ...any) {
	defaultLogWarper.Warn(args...)
}

func (l *PLogWarper) Warnf(template string, args ...any) {
	l.kLog.Log(kLogger.LevelWarn, "msg", fmt.Sprintf(template, args...))
}
func Warnf(template string, args ...any) {
	defaultLogWarper.Warnf(template, args...)
}

// --------------------------------------------------
func (l *PLogWarper) Error(args ...any) {
	l.kLog.Log(kLogger.LevelError, "msg", fmt.Sprint(args...))
}
func Error(args ...any) {
	defaultLogWarper.Error(args...)
}

func (l *PLogWarper) Errorf(template string, args ...any) {
	l.kLog.Log(kLogger.LevelError, "msg", fmt.Sprintf(template, args...))
}
func Errorf(template string, args ...any) {
	defaultLogWarper.Errorf(template, args...)
}

// --------------------------------------------------
func (l *PLogWarper) Fatal(args ...any) {
	l.kLog.Log(kLogger.LevelFatal, "msg", fmt.Sprint(args...))
}
func Fatal(args ...any) {
	defaultLogWarper.Fatal(args...)
}

func (l *PLogWarper) Fatalf(template string, args ...any) {
	l.kLog.Log(kLogger.LevelFatal, "msg", fmt.Sprintf(template, args...))
}
func Fatalf(template string, args ...any) {
	defaultLogWarper.Fatalf(template, args...)
}
