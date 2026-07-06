package pclient

import (
	"fmt"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/pthird"
	"github.com/spf13/cobra"
)

// ParamItem defines a parameter specification
type ParamItem struct {
	Name    string
	Usage   string
	Default string
}

// ParamMap is a map of parameter values
type ParamMap map[string]string

// RegParamToCobra registers parameters to cobra command
// cobra parses parameters and stores values in the returned map
func RegParamToCobra(cmd *cobra.Command,
	specs []ParamItem) map[string]*string {
	flagRefs := make(map[string]*string, len(specs))
	for _, spec := range specs {
		flagRefs[spec.Name] = cmd.Flags().String(spec.Name, spec.Default, spec.Usage)
	}
	return flagRefs
}

// ParseParamFromCobra extracts parameter values from cobra flag refs to ParamMap
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

// GetCachedParamMap interactively prompts user for parameter values with defaults
// Uses cached values from previous runs when available
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
	val := pthird.Interact.Input(inputPrompt)

	if val == "" {
		val = defaultVal
	}

	if val != cachedVal {
		pconfig.SetCacheValue(cachePath, key, val)
	}
	return val
}
