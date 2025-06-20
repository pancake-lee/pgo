package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"pgo/pkg/util"
	"reflect"
	"strconv"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
)

var c config.Config

func MustInitConfig(confFolder string) {
	err := InitConfig(confFolder)
	if err != nil {
		panic(err)
	}
}

func InitConfig(confPath string) (err error) {
	if confPath == "" {
		confPath = filepath.Join(util.GetExecFolder(), "configs")
	}

	f, err := os.Stat(confPath)
	if err != nil || !f.IsDir() {
		c = config.New(config.WithSource(
			file.NewSource(confPath),
		),
		)
	} else {
		execName := util.GetExecName()
		c = config.New(config.WithSource(
			file.NewSource(filepath.Join(confPath, "common.yaml")),
			file.NewSource(filepath.Join(confPath, execName+".yaml")),
		),
		)
	}

	err = c.Load()
	if err != nil {
		// 从框架上来说，配置文件不是必须的
		// return err
	}

	return nil
}

func MustGetConfig() config.Config {
	if c == nil {
		log.Fatalf("config Uninitialized, please call InitConfig first")
		return nil
	}
	return c
}

// 支持定义结构体来解析配置
func Scan(v any) (err error) {
	err = SetDefaults(v)
	if err != nil {
		return err
	}
	err = MustGetConfig().Scan(v)
	if err != nil {
		return err
	}
	return nil
}

// --------------------------------------------------
// 支持代码配置默认值，支持字符串/整形/浮点型/布尔型
// 通过Tag如`default:"10"`来设置默认值，值都使用字符串来填写
func SetDefaults(ptr any) error {
	v := reflect.ValueOf(ptr).Elem() // 获取指针指向的结构体
	t := v.Type()                    // 获取结构体类型

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)   // 获取字段值
		tag := t.Field(i).Tag // 获取字段Tag

		// fmt.Println(field.Type().Name())

		// 递归处理嵌套结构体（忽略time.Time等特殊类型）
		if field.Kind() == reflect.Struct &&
			field.CanAddr() {
			err := SetDefaults(field.Addr().Interface())
			if err != nil {
				return err
			}
			continue
		}

		// 如果字段已经是零值且有default Tag，则设置默认值
		if field.IsZero() && tag.Get("default") != "" {
			defaultValue := tag.Get("default")
			// fmt.Println(defaultValue)
			err := setFieldValue(field, defaultValue)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// 根据字段类型设置值
func setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(intVal)
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatVal)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}
	return nil
}
