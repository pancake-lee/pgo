package dbexport

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pancake-lee/pgo/pkg/pdb"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

// colAndIdMap : colName -> oldId -> newId
// 注意，这里只处理数据，不做建表工作，两个数据库的结构应该手动先同步好
func ImportTable(tbl *Table, inputFolder string, colAndIdMap map[string]map[int64]int64) error {
	filePath := filepath.Join(inputFolder, tbl.TableName+".sql")
	// 读取整个文件内容以避免bufio.Scanner的token长度限制
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	content := string(contentBytes)
	lines := strings.Split(content, "\n")

	var currentSQL strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 跳过注释和空行
		if line == "" || strings.HasPrefix(line, "--") || strings.HasPrefix(line, "/*") {
			continue
		}

		currentSQL.WriteString(line)

		// 检查是否为完整的SQL语句（以分号结尾）
		if strings.HasSuffix(line, ";") {
			sqlStatement := currentSQL.String()
			if err := processInsertStatement(tbl, sqlStatement, colAndIdMap); err != nil {
				plogger.Errorf("Failed to process SQL statement: %v", err)
				return err
			}
			currentSQL.Reset()

		} else {
			//多行命令用空格拼接
			currentSQL.WriteString(" ")
		}
	}

	return nil
}

// processInsertStatement 处理INSERT语句，替换主键映射并执行
func processInsertStatement(tbl *Table, sqlStatement string, colAndIdMap map[string]map[int64]int64) error {
	// 检查是否为INSERT语句
	if !strings.HasPrefix(strings.ToUpper(sqlStatement), "INSERT") {
		return nil
	}

	// 确定该表的自增主键列，记录旧id
	var oldKeyVal int64
	keyCol := ""
	for _, col := range tbl.colList {
		if col.Extra == "auto_increment" {
			keyCol = col.Field
			plogger.Debugf("Table %s has auto_increment key column: %s", tbl.TableName, keyCol)
			break
		}
	}

	// 解析INSERT语句获取表名和列名
	tableName, columns, valuesList, err := parseInsertStatement(sqlStatement)
	if err != nil {
		return err
	}

	for _, values := range valuesList {
		newValues := make([]string, len(values))
		copy(newValues, values)

		for i, colName := range columns {
			if colName == keyCol {
				// 如果是主键列，需要记录旧id
				oldKeyVal, err = putil.StrToInt64(strings.Trim(values[i], "'\""))
				if err != nil {
					plogger.Errorf("Failed to convert tbl [%v] old key value: %v", tbl, err)
					return err
				}
			}

			if oldId, err := putil.StrToInt64(strings.Trim(values[i], "'\"")); err == nil {
				realColName, ok := tbl.ColMap[colName]
				if ok {
					colName = realColName
				}

				if idMap, ok := colAndIdMap[colName]; ok {
					// 尝试转换为int64并查找映射
					newId, ok := idMap[oldId]
					if ok {
						plogger.Debugf("Mapped %s: %d -> %d", colName, oldId, newId)
						newValues[i] = putil.Int64ToStr(newId)
					}
				}
			}
		}

		var sqlCol string
		var sqlVal string
		for i, colName := range columns {
			if colName == keyCol {
				continue
			}
			sqlCol += fmt.Sprintf("`%s`,", colName) // 确保列名被反引号包裹
			sqlVal += fmt.Sprintf("%s,", newValues[i])
		}
		sqlCol = strings.TrimSuffix(sqlCol, ",")
		sqlVal = strings.TrimSuffix(sqlVal, ",")

		// 重新构建SQL语句
		newSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);",
			tableName, sqlCol, sqlVal)

		// 执行SQL语句
		plogger.Debugf("Executing SQL: %s", newSQL)
		res, err := pdb.Exec(newSQL)
		if err != nil {
			plogger.Errorf("failed to execute SQL statement: %v", err)
			return err
		}

		if keyCol != "" && oldKeyVal != 0 {
			lastInsertId, err := res.LastInsertId()
			if err != nil {
				plogger.Errorf("failed to get last insert id: %v", err)
				return err
			}
			if colAndIdMap[keyCol] == nil {
				colAndIdMap[keyCol] = make(map[int64]int64)
			}
			colAndIdMap[keyCol][oldKeyVal] = lastInsertId
		}
	}

	return nil
}

// parseInsertStatement 解析INSERT语句
// 注意valuesList是双层的，第一层是行，第二层是一行中每列的值
func parseInsertStatement(sql string) (tableName string, columns []string, valuesList [][]string, err error) {

	if len(sql) > 2048 {
		plogger.Debugf("Parsing SQL: %s", sql[0:2048])
	} else {
		plogger.Debugf("Parsing SQL: %s", sql)
	}

	// 去除首尾空格并转为大写进行匹配
	upperSQL := strings.ToUpper(strings.TrimSpace(sql))

	// 检查是否以INSERT INTO开头
	if !strings.HasPrefix(upperSQL, "INSERT INTO") {
		return "", nil, nil, fmt.Errorf("not an INSERT statement")
	}

	// 移除INSERT INTO前缀
	remaining := strings.TrimSpace(sql[11:]) // len("INSERT INTO") = 11

	// 查找表名（到第一个空格或左括号为止）
	tableEndIdx := strings.Index(remaining, "(")
	if tableEndIdx == -1 {
		return "", nil, nil, fmt.Errorf("cannot find table name")
	}

	tableName = remaining[:tableEndIdx]
	tableName = strings.TrimSpace(tableName)
	plogger.Debugf("Table name: %s", tableName)

	remaining = strings.TrimSpace(remaining[tableEndIdx:])

	// 查找列名部分（在括号内）
	columnsEndIdx := strings.Index(remaining, ")")
	if columnsEndIdx == -1 {
		return "", nil, nil, fmt.Errorf("cannot find end of column list")
	}

	// 提取列名
	columnsStr := remaining[1:columnsEndIdx] // 1开始，是为了去掉括号
	plogger.Debugf("columnsStr: %s", columnsStr)
	columns = parseColumnNames(columnsStr)
	for i, c := range columns {
		columns[i] = strings.Trim(c, "` ") // 去掉反引号和空格
	}

	// 查找VALUES关键字
	remaining = strings.TrimSpace(remaining[columnsEndIdx+1:])
	valuesIdx := strings.Index(strings.ToUpper(remaining), "VALUES")
	if valuesIdx == -1 {
		return "", nil, nil, fmt.Errorf("cannot find VALUES keyword in sql : %v", sql)
	}

	// 跳过VALUES关键字
	remaining = strings.TrimSpace(remaining[valuesIdx+6:]) // len("VALUES") = 6

	for {
		if len(remaining) > 512 {
			plogger.Debugf("remaining: %s", remaining[0:512])
		} else {
			plogger.Debugf("remaining: %s", remaining)
		}

		valuesStartIdx := strings.Index(remaining, "(")
		if valuesStartIdx == -1 {
			break // 没有更多的值部分了
		}

		// 找到值部分的结束括号
		valuesEndIdx := strings.Index(remaining, "),")
		if valuesEndIdx == -1 {
			valuesEndIdx = strings.Index(remaining, ");")
			if valuesEndIdx == -1 {
				return "", nil, nil, fmt.Errorf("cannot find end of values list")
			}
		}

		// 提取值
		valuesStr := remaining[valuesStartIdx+1 : valuesEndIdx]
		values := parseValues(valuesStr)
		valuesList = append(valuesList, values)

		remaining = strings.TrimSpace(remaining[valuesEndIdx+1:])
	}

	return tableName, columns, valuesList, nil
}

// parseColumnNames 解析列名字符串
func parseColumnNames(columnsStr string) []string {
	var columns []string
	var current strings.Builder

	for _, char := range columnsStr {
		if char == ',' {
			if current.Len() > 0 {
				columns = append(columns, strings.TrimSpace(current.String()))
				current.Reset()
			}
			continue
		}
		if char == ' ' || char == '\t' || char == '\n' {
			continue
		}

		current.WriteRune(char)
	}

	if current.Len() > 0 {
		columns = append(columns, strings.TrimSpace(current.String()))
	}

	return columns
}

// parseValues 解析VALUES部分的值
func parseValues(valuesStr string) []string {
	var values []string
	var current strings.Builder

	// 值可能被单双引号包裹，所以需要特殊处理
	inQuotes := false
	quoteChar := byte(0)

	for i := 0; i < len(valuesStr); i++ {
		char := valuesStr[i]

		if !inQuotes && (char == '\'' || char == '"') {
			inQuotes = true
			quoteChar = char
			current.WriteByte(char)
		} else if inQuotes && char == quoteChar {
			inQuotes = false
			current.WriteByte(char)
		} else if !inQuotes && char == ',' {
			values = append(values, strings.TrimSpace(current.String()))
			current.Reset()
		} else {
			current.WriteByte(char)
		}
	}

	if current.Len() > 0 {
		values = append(values, strings.TrimSpace(current.String()))
	}

	return values
}
