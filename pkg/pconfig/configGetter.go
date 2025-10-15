package pconfig

import "fmt"

// 支持动态获取配置，包括error/default/must模式

// --------------------------------------------------
func GetInt32D(key string, defaultVal int32) int32 {
	if c == nil {
		return defaultVal
	}
	v, err := MustGetConfig().Value(key).Int()
	if err != nil {
		return defaultVal
	}
	return int32(v)
}

func GetInt32E(key string) (int32, error) {
	if c == nil {
		return 0, fmt.Errorf("config not initialized")
	}
	v, err := MustGetConfig().Value(key).Int()
	if err != nil {
		return 0, err
	}
	return int32(v), nil
}

func GetInt32M(key string) int32 {
	v, err := MustGetConfig().Value(key).Int()
	if err != nil {
		panic(fmt.Errorf("must get config value[%v] error: %v", key, err))
	}
	return int32(v)
}

// --------------------------------------------------
func GetInt64D(key string, defaultVal int64) int64 {
	if c == nil {
		return defaultVal
	}
	v, err := MustGetConfig().Value(key).Int()
	if err != nil {
		return defaultVal
	}
	return v
}

func GetInt64E(key string) (int64, error) {
	if c == nil {
		return 0, fmt.Errorf("config not initialized")
	}
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
	if c == nil {
		return defaultVal
	}
	v, err := MustGetConfig().Value(key).String()
	if err != nil {
		return defaultVal
	}
	return v
}

func GetStringE(key string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("config not initialized")
	}
	return MustGetConfig().Value(key).String()
}

func GetStringM(key string) string {
	v, err := MustGetConfig().Value(key).String()
	if err != nil {
		panic(fmt.Errorf("must get config value[%v] error: %v", key, err))
	}
	return v
}
