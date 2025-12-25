package plogger

import (
	"fmt"
	"sort"
	"strings"

	kLog "github.com/go-kratos/kratos/v2/log"
	"github.com/pancake-lee/pgo/pkg/putil"
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
// 2：pLogger实现了kratos的log.Logger接口，依赖注入到kratos包内
// PS：需要固定嵌入到每行日志的值，都在依赖注入前的log.With中定义，可以但不建议分散在其他地方在log.With追加动态的参数
// 3：myHelper参考kratos的log.Helper做一次包装，提供给业务代码使用
// PS：不直接使用log.Helper是因为我在业务代码中需要封装更加方便的日志打印方法
// 4：业务代码使用myHelper打印日志

// 客户端程序可以通过log.SetLogger直接设置kratos包的全局日志，然后log.Debug
// 为了统一代码，可以提供客户端专用的rpcCtx，log对象不用从service对象传递一遍，直接构造就好了

// log.Logger接口实现检查
var _ kLog.Logger = (*pLogger)(nil)

type pLogger struct {
	log *zap.Logger
}

func FromZap(zLogger *zap.Logger) *pLogger {
	return &pLogger{log: zLogger}
}

func (l *pLogger) Log(level kLog.Level, keyVals ...any) error {
	if isJsonLog {
		return l.logJson(level, keyVals...)
	}
	return l.logConsole(level, keyVals...)
}

func (l *pLogger) logJson(level kLog.Level, keyVals ...any) error {
	if len(keyVals) == 0 || len(keyVals)%2 != 0 {
		l.log.Warn(fmt.Sprint("kv must appear in pairs: ", keyVals))
		return nil
	}

	var msg string
	var fields []zap.Field

	for i := 0; i < len(keyVals); i += 2 {
		k := fmt.Sprint(keyVals[i])
		v := keyVals[i+1]

		if k == "msg" {
			msg = fmt.Sprint(v)
			continue
		}
		fields = append(fields, zap.Any(k, v))
	}

	switch level {
	case kLog.LevelDebug:
		l.log.Debug(msg, fields...)
	case kLog.LevelInfo:
		l.log.Info(msg, fields...)
	case kLog.LevelWarn:
		l.log.Warn(msg, fields...)
	case kLog.LevelError:
		l.log.Error(msg, fields...)
	case kLog.LevelFatal:
		l.log.Fatal(msg, fields...)
	default:
		l.log.Error(msg, fields...)
	}
	return nil
}

func (l *pLogger) logConsole(level kLog.Level, keyVals ...any) error {
	if len(keyVals) == 0 || len(keyVals)%2 != 0 {
		l.log.Warn(fmt.Sprint("kv must appear in pairs: ", keyVals))
		return nil
	}

	// 字符串拼接可能是性能瓶颈，
	// 但zap的sugar实现也是用fmt.Sprintf而已，
	// 所以出现性能问题再考虑优化吧
	var sb strings.Builder

	prefixData := make(map[string]string)
	otherData := make(map[string]string)
	var caller string
	var msg string
	for i := 0; i < len(keyVals); i += 2 {
		k := fmt.Sprint(putil.AnyToStr(keyVals[i]))
		v := fmt.Sprint(putil.AnyToStr(keyVals[i+1]))
		switch k {
		case "caller":
			caller = v
		case "msg":
			msg = v
		default:
			if globalPrefixKeys[k] {
				prefixData[k] = v
			} else {
				otherData[k] = v
			}
		}
	}

	if len(prefixData) > 0 {
		for _, k := range globalSortedPrefixKey {
			v := prefixData[k]
			if v == "" {
				continue
			}
			sb.WriteString("[")
			sb.WriteString(k)
			sb.WriteString(":")
			sb.WriteString(v)
			sb.WriteString("] ")
		}
	}

	sb.WriteString(msg)

	if len(otherData) > 0 {
		for k, v := range otherData {
			if v == "" {
				continue
			}
			sb.WriteString(" [")
			sb.WriteString(k)
			sb.WriteString(":")
			sb.WriteString(v)
			sb.WriteString("]")
		}
	}

	sb.WriteString(" [")
	sb.WriteString(caller)
	sb.WriteString("]")

	switch level {
	case kLog.LevelDebug:
		l.log.Debug(sb.String())
	case kLog.LevelInfo:
		l.log.Info(sb.String())
	case kLog.LevelWarn:
		l.log.Warn(sb.String())
	case kLog.LevelError:
		l.log.Error(sb.String())
	case kLog.LevelFatal:
		l.log.Fatal(sb.String())
	default:
		l.log.Error("unknown log level: " + sb.String())
	}
	return nil
}

// --------------------------------------------------
// WithPrefix是希望把某几个key打印在msg前面，方便肉眼查看
// 但是因为利用了kratos的Logger接口，增加接口我首先需要自己实现比如log.With方法
/*
prefixKeys      map[string]bool
sortedPrefixKey []string
func (l *pLogger) WithPrefix(kv ...interface{}) kLog.Logger {
	for i := 0; i < len(kv); i += 2 {
		key := fmt.Sprint(kv[i])
		l.prefixKeys[key] = true
		l.sortedPrefixKey = append(l.sortedPrefixKey, key)
	}
	sort.Strings(l.sortedPrefixKey)
	return kLog.With(l, kv...)
}
*/
// 所以直接用全局变量传递prefixKeys了
var globalPrefixKeys = make(map[string]bool)
var globalSortedPrefixKey []string

func SetPrefixKeys(keys ...string) {
	for _, k := range keys {
		if globalPrefixKeys[k] {
			continue
		}
		globalPrefixKeys[k] = true
		globalSortedPrefixKey = append(globalSortedPrefixKey, k)
	}
	sort.Strings(globalSortedPrefixKey)
}

// --------------------------------------------------
// 把trace_id存入context的key定义
type pgoTid int

const PgoTidKey pgoTid = 0
