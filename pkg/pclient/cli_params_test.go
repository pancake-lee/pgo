package pclient

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pancake-lee/pgo/pkg/pconfig"
)

func TestGetCachedParamMapSkipsEditingWithDefaultConfirmation(t *testing.T) {
	for _, answer := range []string{"", "y"} {
		t.Run(fmt.Sprintf("answer_%q", answer), func(t *testing.T) {
			cachePath := filepath.Join(t.TempDir(), "cache.json")
			pconfig.SetCacheValue(cachePath, "test.name", "cached")
			inputList := []string{answer}
			setParamTestIO(t, &inputList, nil)

			values := GetCachedParamMap(cachePath, "test.", []ParamItem{
				{Name: "name", Usage: "Name", Default: "default"},
			})

			if values["name"] != "cached" {
				t.Fatalf("name = %q, want cached", values["name"])
			}
			if len(inputList) != 0 {
				t.Fatalf("unused inputs: %v", inputList)
			}
		})
	}
}

func TestGetCachedParamMapEditsValuesAndUpdatesChangedCache(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "cache.json")
	pconfig.SetCacheValue(cachePath, "test.first", "cached")
	inputList := []string{"n", "", "changed"}
	setParamTestIO(t, &inputList, nil)

	values := GetCachedParamMap(cachePath, "test.", []ParamItem{
		{Name: "first", Usage: "First", Default: "default"},
		{Name: "second", Usage: "Second", Default: "second-default"},
	})

	if values["first"] != "cached" || values["second"] != "changed" {
		t.Fatalf("values = %#v", values)
	}
	if got := pconfig.GetCacheValue(cachePath, "test.first"); got != "cached" {
		t.Fatalf("unchanged cache = %q, want cached", got)
	}
	if got := pconfig.GetCacheValue(cachePath, "test.second"); got != "changed" {
		t.Fatalf("changed cache = %q, want changed", got)
	}
}

func TestGetCachedParamMapMasksSensitiveValues(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "cache.json")
	pconfig.SetCacheValue(cachePath, "test.password", "secret-value")
	inputList := []string{""}
	var outputList []string
	setParamTestIO(t, &inputList, &outputList)

	values := GetCachedParamMap(cachePath, "test.", []ParamItem{
		{Name: "password", Usage: "Password", Sensitive: true},
	})

	if values["password"] != "secret-value" {
		t.Fatalf("password = %q, want cached value", values["password"])
	}
	output := strings.Join(outputList, "\n")
	if strings.Contains(output, "secret-value") {
		t.Fatalf("sensitive value leaked in output: %s", output)
	}
	if !strings.Contains(output, "Password: set") {
		t.Fatalf("sensitive status missing from output: %s", output)
	}
}

func setParamTestIO(t *testing.T, inputList *[]string, outputList *[]string) {
	t.Helper()
	originalInput := pclientParamInput
	originalInfof := pclientParamInfof
	pclientParamInput = func(_ string) string {
		if len(*inputList) == 0 {
			t.Fatal("unexpected interactive input")
		}
		value := (*inputList)[0]
		*inputList = (*inputList)[1:]
		return value
	}
	pclientParamInfof = func(format string, args ...interface{}) {
		if outputList != nil {
			*outputList = append(*outputList, fmt.Sprintf(format, args...))
		}
	}
	t.Cleanup(func() {
		pclientParamInput = originalInput
		pclientParamInfof = originalInfof
	})
}
