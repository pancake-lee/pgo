package logger

import (
	"testing"
)

func TestLogger(t *testing.T) {
	InitLogger(false)
	Debug("test logger Debug, 中文测试，1234567890")
	Info("test logger Info, 中文测试，1234567890")
	Warn("test logger Warn, 中文测试，1234567890")
	Error("test logger Error, 中文测试，1234567890")
}
