package plogger

import (
	"fmt"
	"strings"

	klog "github.com/go-kratos/kratos/v2/log"
	"go.uber.org/zap"
)

// 我们首先要提供一个kratos定义的log接口，本文件就是这个接口的实现
// 我们使用zap实现，参考这两个示例代码
// https://github.com/go-kratos/examples/blob/main/log/zap.go
// https://github.com/go-kratos/kratos/blob/main/contrib/log/zap/zap.go

// 然后log.With方法，kratos提供了几个基本的数据绑定，最重要的是TraceID
// 使用kratos的tracing中间件，它将自动把数据写入到每行日志

// 总结kratos的log接口
// kratos的grpc和http包内的Logger中间件都是废弃了的
// 可以使用/v2/middleware/logging的封装logging.Server(kLogger)，打印每个请求的统计信息
// kratos.New(kratos.Logger(kLogger))定义kratos包内日志的输出
// /v2/log还有log.SetLogger(kLogger)方法，其实上面内部调用了这个方法，所以并不需要我们调用

// 整个log的封装层次：
// 1：zaplog作为最底层的日志实现
// 2：kratosZapLogger实现了kratos的log.Logger接口，依赖注入到kratos包内
// PS：需要固定嵌入到每行日志的值，都在依赖注入前的log.With中定义，可以但不建议分散在其他地方在log.With追加动态的参数
// 3：myHelper参考kratos的log.Helper做一次包装，提供给业务代码使用
// PS：不直接使用log.Helper是因为我在业务代码中需要封装更加方便的日志打印方法
// 4：业务代码使用myHelper打印日志

// 客户端程序可以通过log.SetLogger直接设置kratos包的全局日志，然后log.Debug
// 为了统一代码，可以提供客户端专用的rpcCtx，log对象不用从service对象传递一遍，直接构造就好了

// log.Logger接口实现检查
var _ klog.Logger = (*pLogger)(nil)

// pLogger is a logger impl.
type pLogger struct {
	log *zap.Logger
}

func FromZap(zLogger *zap.Logger) *pLogger {
	return &pLogger{log: zLogger}
}

// Log Implementation of logger interface.
func (l *pLogger) Log(level klog.Level, keyVals ...any) error {
	if len(keyVals) == 0 || len(keyVals)%2 != 0 {
		l.log.Warn(fmt.Sprint("kv must appear in pairs: ", keyVals))
		return nil
	}

	var sb strings.Builder

	var data []zap.Field
	var caller string
	var msg string
	for i := 0; i < len(keyVals); i += 2 {
		k := fmt.Sprint(keyVals[i])
		v := fmt.Sprint(keyVals[i+1])
		switch k {
		case "caller":
			// TODO log.With自定义callerSkip，kratos默认的未必符合需要
			// github.com/go-kratos/kratos/v2@v2.7.3/log/value.go:Caller
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
	case klog.LevelDebug:
		l.log.Debug(sb.String())
	case klog.LevelInfo:
		l.log.Info(sb.String())
	case klog.LevelWarn:
		l.log.Warn(sb.String())
	case klog.LevelError:
		l.log.Error(sb.String())
	case klog.LevelFatal:
		l.log.Fatal(sb.String())
	default:
		l.log.Error(sb.String())
	}
	return nil
}
