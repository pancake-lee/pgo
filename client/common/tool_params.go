package common

import (
	"fmt"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/putil"
	"github.com/spf13/cobra"
)

// 参数定义
type ParamItem struct {
	Name    string
	Usage   string
	Default string
}

type ParamMap map[string]string

// 把自定义的参数列表，注册到cobra命令行中
// cobra解析参数后，将具体值存放在返回的map中
func RegParamToCobra(cmd *cobra.Command,
	specs []ParamItem) map[string]*string {
	flagRefs := make(map[string]*string, len(specs))
	for _, spec := range specs {
		flagRefs[spec.Name] = cmd.Flags().String(spec.Name, spec.Default, spec.Usage)
	}
	return flagRefs
}

// cobra命令行参数解析后，提取参数值到ParamMap结构中
func ParseParamFromCobra(flagRefs map[string]*string) ParamMap {
	values := make(ParamMap, len(flagRefs))
	for key, valueRef := range flagRefs {
		if valueRef == nil {
			continue
		}
		values[key] = *valueRef
	}
	return values
}

// --------------------------------------------------
// 交互式地提示用户输入参数值，并提示默认值
// 先用代码配置的默认值，后用缓存值覆盖
func GetCachedParamMap(
	cachePath string,
	cachePrefix string,
	specs []ParamItem,
) ParamMap {
	values := make(ParamMap, len(specs))
	for _, spec := range specs {
		values[spec.Name] = GetCachedParam(
			cachePath,
			cachePrefix+spec.Name,
			spec.Usage,
			spec.Default)
	}
	return values
}

func GetCachedParam(cachePath, key, prompt, layout string) string {
	cachedVal := pconfig.GetCacheValue(cachePath, key)
	defaultVal := layout
	if cachedVal != "" {
		defaultVal = cachedVal
	}

	inputPrompt := fmt.Sprintf("%s (Default: %s)", prompt, defaultVal)
	val := putil.Interact.Input(inputPrompt)

	if val == "" {
		val = defaultVal
	}

	if val != cachedVal {
		pconfig.SetCacheValue(cachePath, key, val)
	}
	return val
}
