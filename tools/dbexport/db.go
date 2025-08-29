package dbexport

import (
	"database/sql"
	"fmt"

	"github.com/pancake-lee/pgo/pkg/pdb"
	"github.com/pancake-lee/pgo/pkg/plogger"
)

var logFlag = false

func GetScopeToTblMap(scopeToTblMap map[string][]*Table) map[string][]*Table {
	for scope, tblList := range scopeToTblMap {
		for _, tbl := range tblList {
			tbl.scope = scope

			columns, err := getTableColumns(tbl.TableName)
			if err != nil {
				plogger.Errorf("Failed to get columns for table %s: %v", tbl.TableName, err)
				continue
			}
			tbl.colList = columns

			// 打印表结构
			if logFlag {
				plogger.Debugf("=== 表: %s ===", tbl.TableName)
				plogger.Debugf("%-20s %-15s %-10s %-10s %-10s %-20s",
					"Field", "Type", "Null", "Key", "Default", "Extra")
				for _, col := range columns {
					plogger.Debugf("%-20s %-15s %-10s %-10s %-10s %-20s",
						col.Field, col.Type, col.Null, col.Key, col.Default, col.Extra)
				}
			}
		}
	}
	return scopeToTblMap
}

// --------------------------------------------------
type Table struct {
	scope     string
	TableName string
	colList   []*TableColumn
	ColMap    map[string]string //列名 -> 实际关联的列名(tblName_colName)
}

func GetTblInScopeMap(tblName string, scopeToTblMap map[string][]*Table) *Table {
	for _, tblList := range scopeToTblMap {
		for _, tbl := range tblList {
			if tbl.TableName == tblName {
				return tbl
			}
		}
	}
	return nil
}

// TableColumn 表示表列的结构
type TableColumn struct {
	Field   string
	Type    string
	Null    string
	Key     string
	Default string
	Extra   string
}

// getTableColumns 获取指定表的列信息
func getTableColumns(tableName string) ([]*TableColumn, error) {
	db, err := pdb.GetDB()
	if err != nil {
		plogger.Errorf("Failed to get database connection: %v", err)
		return nil, err
	}

	query := fmt.Sprintf("DESCRIBE %s", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []*TableColumn
	for rows.Next() {
		var col TableColumn
		var defaultVal sql.NullString

		err := rows.Scan(&col.Field, &col.Type, &col.Null, &col.Key, &defaultVal, &col.Extra)
		if err != nil {
			return nil, err
		}

		if defaultVal.Valid {
			col.Default = defaultVal.String
		} else {
			col.Default = "NULL"
		}

		columns = append(columns, &col)
	}

	return columns, nil
}
