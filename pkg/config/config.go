package config

import (
	"log"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/pelletier/go-toml"
)

// 临时简单实现配置的读取，后续肯定是要优化的
var conf *toml.Tree

func LoadConf(confPath string) {
	c, err := toml.LoadFile(confPath)
	if err != nil {
		panic(err)
	}
	conf = c
}

func GetConfStr(confKey string) string {
	if v, ok := conf.Get(confKey).(string); ok {
		log.Printf("config key: %s, value: %s", confKey, v)
		return v
	}
	return ""
}

func GetConfInt(confKey string) int {
	if v, ok := conf.Get(confKey).(int64); ok {
		log.Printf("config key: %s, value: %d", confKey, v)
		return int(v)
	}
	return 0
}

// --------------------------------------------------
// 支持定义结构体来解析配置
// 支持动态获取配置，包括error/default/must模式
// 支持代码配置默认值

type myHttpConfig struct {
	Addr    string
	Timeout int
}
type myConfig struct {
	Http myHttpConfig
}

func LoadConfFromFile(path string, conf any) (c config.Config, err error) {
	c = config.New(
		config.WithSource(
			file.NewSource(path),
		),
	)

	err = c.Load()
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	err = c.Scan(conf)
	if err != nil {
		log.Fatalf("scan config failed: %v", err)
	}
	return c, nil
}
