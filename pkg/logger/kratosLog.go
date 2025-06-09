package logger

import (
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"go.uber.org/zap"
)

// https://github.com/go-kratos/examples/blob/main/log/zap.go

// 这一行是一个断言，没有业务逻辑意义，但可以让代码检查出kratosZapLogger是否实现了log.Logger接口
var _ log.Logger = (*kratosZapLogger)(nil)

// kratosZapLogger is a logger impl.
type kratosZapLogger struct {
	log  *zap.SugaredLogger
	Sync func() error
}

var DefaultKratosLogger *kratosZapLogger

func initKratosLogger(zLogger *zap.SugaredLogger) {
	DefaultKratosLogger = &kratosZapLogger{log: zLogger, Sync: zLogger.Sync}
}

// Log Implementation of logger interface.
func (l *kratosZapLogger) Log(level log.Level, keyVals ...any) error {
	if len(keyVals) == 0 || len(keyVals)%2 != 0 {
		l.log.Warn(fmt.Sprint("kv must appear in pairs: ", keyVals))
		return nil
	}

	var sb strings.Builder
	sb.WriteString("[kratos] ")

	var data []zap.Field
	var caller string
	var msg string
	for i := 0; i < len(keyVals); i += 2 {
		k := fmt.Sprint(keyVals[i])
		v := fmt.Sprint(keyVals[i+1])
		switch k {
		case "caller":
			caller = v
		case "msg":
			msg = v
		default:
			data = append(data, zap.Any(k, v))
		}
	}
	sb.WriteString(msg)

	if len(data) > 0 {
		sb.WriteString(fmt.Sprint(data))
	}

	sb.WriteString(" [")
	sb.WriteString(caller)
	sb.WriteString("]")

	switch level {
	case log.LevelDebug:
		l.log.Debug(sb.String())
	case log.LevelInfo:
		l.log.Info(sb.String())
	case log.LevelWarn:
		l.log.Warn(sb.String())
	case log.LevelError:
		l.log.Error(sb.String())
	case log.LevelFatal:
		l.log.Fatal(sb.String())
	default:
		l.log.Error(sb.String())
	}
	return nil
}
