package config

import (
	"log"

	"github.com/pelletier/go-toml"
)

// 临时简单实现配置的读取，后续肯定是要优化的
var config *toml.Tree

func LoadConf(confPath string) {
	c, err := toml.LoadFile(confPath)
	if err != nil {
		panic(err)
	}
	config = c
}

func GetConfStr(confKey string) string {
	if v, ok := config.Get(confKey).(string); ok {
		log.Printf("config key: %s, value: %s", confKey, v)
		return v
	}
	return ""
}

func GetConfInt(confKey string) int {
	if v, ok := config.Get(confKey).(int); ok {
		log.Printf("config key: %s, value: %d", confKey, v)
		return v
	}
	return 0
}
