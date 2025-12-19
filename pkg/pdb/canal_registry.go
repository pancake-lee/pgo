package pdb

import (
	"context"
	"sync"

	"github.com/go-mysql-org/go-mysql/schema"
)

// InsertCallback defines the function signature for insert event handlers
type InsertCallback func(ctx context.Context, columns []schema.TableColumn, newRow []interface{}) error

// UpdateCallback defines the function signature for update event handlers
type UpdateCallback func(ctx context.Context, columns []schema.TableColumn, oldRow, newRow []interface{}) error

// DeleteCallback defines the function signature for delete event handlers
type DeleteCallback func(ctx context.Context, columns []schema.TableColumn, oldRow []interface{}) error

// TableHandler holds callbacks for a specific table
type TableHandler struct {
	InsertCallback InsertCallback
	UpdateCallback UpdateCallback
	DeleteCallback DeleteCallback
}

var (
	registryMu sync.RWMutex
	registry   = make(map[string]*TableHandler)
)

func getOrCreateHandler(tableName string) *TableHandler {
	if h, ok := registry[tableName]; ok {
		return h
	}
	h := &TableHandler{}
	registry[tableName] = h
	return h
}

// RegisterInsertCallback registers an insert callback for a specific table
func RegisterInsertCallback(tableName string, cb InsertCallback) {
	registryMu.Lock()
	defer registryMu.Unlock()
	h := getOrCreateHandler(tableName)
	h.InsertCallback = cb
}

// RegisterUpdateCallback registers an update callback for a specific table
func RegisterUpdateCallback(tableName string, cb UpdateCallback) {
	registryMu.Lock()
	defer registryMu.Unlock()
	h := getOrCreateHandler(tableName)
	h.UpdateCallback = cb
}

// RegisterDeleteCallback registers a delete callback for a specific table
func RegisterDeleteCallback(tableName string, cb DeleteCallback) {
	registryMu.Lock()
	defer registryMu.Unlock()
	h := getOrCreateHandler(tableName)
	h.DeleteCallback = cb
}

// GetTableHandler retrieves the handler for a specific table
func GetTableHandler(tableName string) *TableHandler {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[tableName]
}

// GetAllTables returns all registered table names
func GetAllTables() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	tables := make([]string, 0, len(registry))
	for t := range registry {
		tables = append(tables, t)
	}
	return tables
}
