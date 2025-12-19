package pdb

import (
	"context"
	"sync"

	"github.com/go-mysql-org/go-mysql/canal"
)

// Callback defines the function signature for table event handlers
// Callback 定义表事件处理函数的签名
type Callback func(ctx context.Context, e *canal.RowsEvent) error

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Callback)
)

// RegisterCallback registers a callback for a specific table
// RegisterCallback 注册特定表的处理回调
func RegisterCallback(tableName string, cb Callback) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[tableName] = cb
}

// GetCallback retrieves the callback for a specific table
// GetCallback 获取特定表的处理回调
func GetCallback(tableName string) (Callback, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	cb, ok := registry[tableName]
	return cb, ok
}

// GetAllTables returns all registered table names
// GetAllTables 返回所有已注册的表名
func GetAllTables() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	tables := make([]string, 0, len(registry))
	for t := range registry {
		tables = append(tables, t)
	}
	return tables
}
