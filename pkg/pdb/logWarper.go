package pdb

import "github.com/pancake-lee/pgo/pkg/plogger"

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
