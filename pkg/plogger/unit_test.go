package plogger

import (
	"errors"
	"fmt"
	"runtime"
	"testing"
	"time"

	kLog "github.com/go-kratos/kratos/v2/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestPLoggerCallerAndConsoleOrder(t *testing.T) {
	originalJSON := isJsonLog
	originalPrefixKeys := globalPrefixKeys
	originalSortedKeys := globalSortedPrefixKey
	t.Cleanup(func() {
		isJsonLog = originalJSON
		globalPrefixKeys = originalPrefixKeys
		globalSortedPrefixKey = originalSortedKeys
	})
	globalPrefixKeys = make(map[string]bool)
	globalSortedPrefixKey = nil
	SetPrefixKeys("tid", "sid")

	for _, jsonMode := range []bool{false, true} {
		for _, explicit := range []string{"absent", "custom.go:42", ""} {
			t.Run(fmt.Sprintf("json=%t/caller=%s", jsonMode, explicit),
				func(t *testing.T) {
					isJsonLog = jsonMode
					core, logs := observer.New(zap.DebugLevel)
					logger := FromZap(zap.New(core))
					kv := []any{
						"msg", "hello", "z", "last", "a", "first",
						"tid", "trace", "sid", "span", "empty", "",
					}
					if explicit != "absent" {
						kv = append(kv, "caller", explicit)
					}
					for range 20 {
						_, file, line, _ := runtime.Caller(0)
						err := logger.Log(kLog.LevelInfo, kv...)
						if err != nil {
							t.Fatal(err)
						}
						caller := explicit
						if explicit == "absent" {
							caller = fmt.Sprintf("%s:%d", file, line+1)
						}
						entry := logs.All()[logs.Len()-1]
						if jsonMode {
							if entry.ContextMap()["caller"] != caller {
								t.Fatalf("wrong caller: %v", entry.ContextMap())
							}
							if entry.Message != "hello" {
								t.Fatalf("wrong message: %q", entry.Message)
							}
						} else {
							want := "[sid:span] [tid:trace] hello" +
								" [a:first] [z:last] [" + caller + "]"
							if entry.Message != want {
								t.Fatalf("got %q, want %q", entry.Message, want)
							}
						}
					}
					for _, kv := range [][]any{nil, {"msg"}} {
						err := logger.Log(kLog.LevelInfo, kv...)
						if err != nil {
							t.Fatal(err)
						}
						entry := logs.All()[logs.Len()-1]
						if entry.Level != zap.WarnLevel {
							t.Fatalf("invalid kv level: %v", entry.Level)
						}
					}
				})
		}
	}
}

func TestZapLogger(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	logger.Debug("test zap Debug, 中文测试，1234567890")
	logger.Info("test zap Info, 中文测试，1234567890")
	logger.Warn("test zap Warn, 中文测试，1234567890")
	logger.Error("test zap Error, 中文测试，1234567890")

	log := logger.Sugar()
	log.Debug("test zap SugaredLogger Debug, 中文测试，1234567890")
	log.Debugf("test zap SugaredLogger Debug, 中文测试，%d", 1234567890)
	log.Debugln("test zap SugaredLogger Debugln, 中文测试，1234567890")
	log.Debugw("test zap SugaredLogger Debugw, 中文测试，1234567890",
		"key1", "value1", "key2", "value2")
}

func TestLogger(t *testing.T) {
	InitLogger(true, zap.DebugLevel, "")
	Debug("test logger Debug, 中文测试，1234567890")
	Info("test logger Info, 中文测试，1234567890")
	Warn("test logger Warn, 中文测试，1234567890")
	Error("test logger Error, 中文测试，1234567890")

	l := NewPLogWarper(GetDefaultLogger())
	l.Debug("test warper Debug, 中文测试，1234567890")
}

func TestTimeLogger(t *testing.T) {
	InitLogger(true, zap.DebugLevel, "")

	tLogger := NewTimeLogger("TestTimeLogger")
	defer tLogger.Log()

	tLogger.AddPoint("start")
	time.Sleep(100 * time.Millisecond)
	tLogger.AddPointInc()
	time.Sleep(200 * time.Millisecond)

	for range 3 {
		tLogger.AddPointIncPrefix("loop")
		time.Sleep(50 * time.Millisecond)
	}
}

func a(log func(args ...any)) error {
	if time.Now().Unix()%2 == 0 {
		// 自身函数错误需要打印日志
		err := errors.New("aa error")
		log(err.Error())
		return err
	}

	err := b(log)
	if err != nil {
		// 如果log方法会打印调用栈
		// 那么调用b不用打印日志
		// 因为b里面会输出调用栈
		// log("bb error")
		return err
	}

	return nil
}

func b(log func(args ...any)) error {
	err := errors.New("bb error")
	log(err.Error())
	return err
}
func TestLogErrReturn(t *testing.T) {
	InitLogger(true, zap.DebugLevel, "")
	a(Error)

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	log := logger.Sugar()
	a(func(args ...any) { log.Warnw(args[0].(string)) })
}

func TestLogErrMsgPreservesPlainText(t *testing.T) {
	InitLogger(true, zap.DebugLevel, "")
	if err := LogErrMsg("literal 100% message"); err == nil || err.Error() != "literal 100% message" {
		t.Fatalf("unexpected plain error: %v", err)
	}
	if err := LogErrfMsg("item %d", 7); err == nil || err.Error() != "item 7" {
		t.Fatalf("unexpected formatted error: %v", err)
	}
}
