package logger

import (
	"testing"

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
	InitLogger(false)
	Debug("test logger Debug, 中文测试，1234567890")
	Info("test logger Info, 中文测试，1234567890")
	Warn("test logger Warn, 中文测试，1234567890")
	Error("test logger Error, 中文测试，1234567890")
}
