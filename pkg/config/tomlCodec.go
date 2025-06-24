package config

import (
	"github.com/BurntSushi/toml"
	"github.com/go-kratos/kratos/v2/encoding"
)

type tomlCodec struct{}

func (c *tomlCodec) Marshal(v any) ([]byte, error) {
	var buf []byte
	encoder := toml.NewEncoder(&bufferWriter{buf: &buf})
	err := encoder.Encode(v)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (c *tomlCodec) Unmarshal(data []byte, v any) error {
	_, err := toml.Decode(string(data), v)
	return err
}

func (c *tomlCodec) Name() string {
	return "toml"
}

// bufferWriter 实现 io.Writer 接口，用于 toml.NewEncoder
type bufferWriter struct {
	buf *[]byte
}

func (w *bufferWriter) Write(p []byte) (n int, err error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}

func init() {
	encoding.RegisterCodec(&tomlCodec{})
}
