package pconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pancake-lee/pgo/pkg/putil"
)

var cacheLock sync.Mutex

func GetDefaultCachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "cache.json"
	}
	// 目录是~/pgo/但不同项目按项目名称区分缓存文件
	return filepath.Join(home, "pgo",
		putil.NewPathS(putil.GetCurDir()).GetLast()+"cache.json")
}

// GetCacheValue reads a string value from a JSON file using a dot-separated key path.
// Returns empty string if file doesn't exist or key not found.
func GetCacheValue(filePath string, keyPath string) string {
	cacheLock.Lock()
	defer cacheLock.Unlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}

	var current interface{}
	if err := json.Unmarshal(data, &current); err != nil {
		return ""
	}

	keys := strings.Split(keyPath, ".")
	for _, key := range keys {
		if m, ok := current.(map[string]interface{}); ok {
			if val, exists := m[key]; exists {
				current = val
			} else {
				return ""
			}
		} else {
			return ""
		}
	}

	if str, ok := current.(string); ok {
		return str
	}
	return ""
}

// SetCacheValue writes a string value to a JSON file using a dot-separated key path.
// Creates the file and directories if they don't exist.
func SetCacheValue(filePath string, keyPath string, value string) error {
	cacheLock.Lock()
	defer cacheLock.Unlock()

	dir := filepath.Dir(filePath)
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	_, err = os.Stat(filePath)
	if os.IsNotExist(err) {
		err = os.WriteFile(filePath, []byte("{}"), 0644)
		if err != nil {
			return err
		}
	}

	var root map[string]interface{}
	data, err := os.ReadFile(filePath)
	if err == nil {
		_ = json.Unmarshal(data, &root)
	}
	if root == nil {
		root = make(map[string]interface{})
	}

	keys := strings.Split(keyPath, ".")
	current := root
	for i, key := range keys {
		if i == len(keys)-1 {
			current[key] = value
		} else {
			if next, ok := current[key].(map[string]interface{}); ok {
				current = next
			} else {
				next := make(map[string]interface{})
				current[key] = next
				current = next
			}
		}
	}

	// Prepare output
	outData, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, outData, 0644)
}
