package plogger

import (
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

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
