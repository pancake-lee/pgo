package common

import (
	"fmt"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/putil"
)

func GetCachedInput(cachePath, key, prompt, layout string) string {
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
