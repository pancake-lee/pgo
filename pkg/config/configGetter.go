package config

import "fmt"

// 支持动态获取配置，包括error/default/must模式

// --------------------------------------------------
func GetInt64D(key string, defaultVal int64) int64 {
	v, err := MustGetConfig().Value(key).Int()
	if err != nil {
		return defaultVal
	}
	return v
}

func GetInt64E(key string) (int64, error) {
	return MustGetConfig().Value(key).Int()
}

func GetInt64M(key string) int64 {
	v, err := MustGetConfig().Value(key).Int()
	if err != nil {
		panic(fmt.Errorf("must get config value[%v] error: %v", key, err))
	}
	return v
}

// --------------------------------------------------
func GetStringD(key string, defaultVal string) string {
	v, err := MustGetConfig().Value(key).String()
	if err != nil {
		return defaultVal
	}
	return v
}

func GetStringE(key string) (string, error) {
	return MustGetConfig().Value(key).String()
}

func GetStringM(key string) string {
	v, err := MustGetConfig().Value(key).String()
	if err != nil {
		panic(fmt.Errorf("must get config value[%v] error: %v", key, err))
	}
	return v
}
