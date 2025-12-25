package plogger

import (
	"encoding/json"
	"log"
	"net"
	"time"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/putil"
	"go.uber.org/zap/zapcore"
)

// graylog其实没有想象中那么好用，因为UI算是比较简陋了一点
// 实现zapcore.Core接口以获得日志level
type GraylogCore struct {
	zapcore.LevelEnabler
	conn         net.Conn
	addr         string
	hostname     string
	writeTimeout time.Duration
	enc          zapcore.Encoder
}

func NewGraylogCore(addr string) zapcore.Core {
	if addr == "" {
		var err error
		addr, err = pconfig.GetStringE("Graylog.Addr")
		if err != nil || addr == "" {
			return nil
		}
	}
	// 改为TCP连接
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Printf("graylog dial error: %v", err)
		return nil
	}

	hostname := putil.GetExecName()

	np, err := pconfig.GetStringE("Graylog.NamePrefix")
	if err == nil || np != "" {
		hostname = np + "_" + hostname
	}

	encCfg := zapcore.EncoderConfig{
		TimeKey:        "T",
		LevelKey:       "L",
		NameKey:        "N",
		MessageKey:     "M",
		StacktraceKey:  "S",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}
	return &GraylogCore{
		LevelEnabler: logLevel, // 使用全局日志级别
		conn:         conn,
		addr:         addr,
		hostname:     hostname,
		writeTimeout: 5 * time.Second,
		enc:          zapcore.NewJSONEncoder(encCfg),
	}
}

func (c *GraylogCore) With(fields []zapcore.Field) zapcore.Core {
	clone := *c
	clone.enc = c.enc.Clone()
	for _, f := range fields {
		f.AddTo(clone.enc)
	}
	return &clone
}

func (c *GraylogCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *GraylogCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	buf, err := c.enc.EncodeEntry(ent, fields)
	if err != nil {
		return err
	}
	gelfMsg := map[string]interface{}{
		"version":       "1.1",
		"host":          c.hostname,
		"short_message": ent.Message,
		"timestamp":     float64(ent.Time.UnixNano()) / 1e9,
		"level":         zapLevelToSyslog(ent.Level),
		"raw":           buf.String(),
	}
	msgBytes, _ := json.Marshal(gelfMsg)
	c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
	_, err = c.conn.Write(msgBytes)
	return err
}

func (c *GraylogCore) Sync() error {
	return nil
}

// zap日志级别转syslog级别
func zapLevelToSyslog(level zapcore.Level) int {
	switch level {
	case zapcore.DebugLevel:
		return 7
	case zapcore.InfoLevel:
		return 6
	case zapcore.WarnLevel:
		return 4
	case zapcore.ErrorLevel:
		return 3
	case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		return 2
	default:
		return 6
	}
}
