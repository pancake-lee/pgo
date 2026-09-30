package pclient

import (
	"fmt"
	"strings"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/pthird"
	"github.com/spf13/cobra"
)

// ParamItem defines a parameter specification
type ParamItem struct {
	Name      string
	Usage     string
	Default   string
	Sensitive bool
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
		cachedVal := pconfig.GetCacheValue(cachePath, cachePrefix+spec.Name)
		values[spec.Name] = spec.Default
		if cachedVal != "" {
			values[spec.Name] = cachedVal
		}
	}

	pclientParamInfof("Current parameters:")
	for _, spec := range specs {
		value := values[spec.Name]
		if spec.Sensitive {
			value = "not set"
			if values[spec.Name] != "" {
				value = "set"
			}
		}
		pclientParamInfof("  %s: %s", spec.Usage, value)
	}

	answer := strings.ToLower(strings.TrimSpace(
		pclientParamInput("Use these parameters? (Y/n)")))
	if answer == "" || answer == "y" || answer == "yes" {
		return values
	}

	for _, spec := range specs {
		currentValue := values[spec.Name]
		inputPrompt := fmt.Sprintf("%s (Default: %s)", spec.Usage, currentValue)
		value := pclientParamInput(inputPrompt)
		if value == "" {
			continue
		}
		values[spec.Name] = value
		if value != currentValue {
			pconfig.SetCacheValue(cachePath, cachePrefix+spec.Name, value)
		}
	}
	return values
}

var pclientParamInput = pthird.Interact.Input

var pclientParamInfof = pthird.Interact.Infof

func GetCachedParam(cachePath, key, prompt, layout string) string {
	cachedVal := pconfig.GetCacheValue(cachePath, key)
	defaultVal := layout
	if cachedVal != "" {
		defaultVal = cachedVal
	}

	inputPrompt := fmt.Sprintf("%s (Default: %s)", prompt, defaultVal)
	val := pclientParamInput(inputPrompt)

	if val == "" {
		val = defaultVal
	}

	if val != cachedVal {
		pconfig.SetCacheValue(cachePath, key, val)
	}
	return val
}
