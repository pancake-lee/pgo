package pconfig

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"

	"github.com/pancake-lee/pgo/pkg/putil"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
)

var c config.Config

// same as InitConfig, but panic if error occurs
func MustInitConfig(paths ...string) {
	err := InitConfig(paths...)
	if err != nil {
		panic(err)
	}
}

// InitConfig 初始化配置
// paths: 可以指定一个或多个配置文件路径，
// 如果是目录，则会尝试加载common.toml和执行文件名的toml
// 如果没有指定路径，则默认加载当前执行文件所在目录下的configs目录中的配置
func InitConfig(paths ...string) (err error) {
	if len(paths) == 0 {
		paths = append(paths, filepath.Join(putil.GetExecFolder(), "configs"))
	}
	var srcList []config.Source
	for _, path := range paths {
		if path == "" {
			path = filepath.Join(putil.GetExecFolder(), "configs")
		}

		f, err := os.Stat(path)
		if err != nil || !f.IsDir() { // 指定文件
			srcList = append(srcList, file.NewSource(path))
			log.Println("config file found:", path)

		} else { // 指定目录，尝试找common和执行文件名的配置文件，但不存在也没关系
			execName := putil.GetExecName()

			// 从框架上来说，配置文件不是必须的
			// kratos封装的Load，一个配置文件Load失败就不继续了
			// 这里要自己判断是否存在
			tryFileNames := []string{
				"common.toml", execName + ".toml",
				"common.yaml", execName + ".yaml",
			}
			if len(tryFileNames) == 0 {
				return errors.New("no config file found")
			}

			for _, n := range tryFileNames {
				path := filepath.Join(path, n)
				_, err := os.Stat(path)
				if err != nil { // 如果文件不存在，继续下一个
					log.Println("try, config file not found:", path)
					continue
				}
				srcList = append(srcList, file.NewSource(path))
				log.Println("config file found:", path)
			}
		}
	}

	c = config.New(config.WithSource(srcList...))

	err = c.Load()
	if err != nil {
		log.Println("config load error:", err)
		return err
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
