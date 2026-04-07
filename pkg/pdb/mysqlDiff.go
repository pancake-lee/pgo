package pdb

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

// RunMysqlSchemaUpgrade 通过对比目标数据库与规范SQL文件，执行仅增量的Schema升级。
func RunMysqlSchemaUpgrade(conf MysqlConfig) error {
	plogger.Info("Starting SQL-diff schema upgrade...")

	host, portStr, err := splitMysqlAddr(conf.Mysql.Addr)
	if err != nil {
		return err
	}

	sqlDir, err := resolveSchemaSQLDir()
	if err != nil {
		return err
	}
	plogger.Infof("Schema SQL dir: %s", sqlDir)

	plannedSQL, warnings, err := planSchemaUpgradeSQL(conf, host, portStr, sqlDir)
	if err != nil {
		return fmt.Errorf("failed to plan schema upgrade sql: %w", err)
	}

	for _, w := range warnings {
		plogger.Warnf("Schema diff warning: %s", w)
	}

	if len(plannedSQL) == 0 {
		plogger.Info("Schema is up to date, no additive SQL to execute.")
		return nil
	}

	if err := execSchemaUpgradeSQL(conf, host, portStr, plannedSQL); err != nil {
		return fmt.Errorf("failed to execute schema upgrade sql: %w", err)
	}

	plogger.Infof("SQL-diff schema upgrade finished, executed %d statements.", len(plannedSQL))
	return nil
}

// splitMysqlAddr 将MySQL地址"host:port"拆分为host和port。
func splitMysqlAddr(addr string) (string, string, error) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid mysql addr: %s", addr)
	}
	return parts[0], parts[1], nil
}

// resolveSchemaSQLDir 从多个候选路径中定位schema SQL文件目录。
func resolveSchemaSQLDir() (string, error) {
	candidates := []string{
		filepath.Join(putil.GetExecFolder(), "database"),
		filepath.Join(putil.GetExecFolder(), "..", "database"),
		filepath.Join(putil.GetCurDir(), "internal", "pkg", "db"),
		filepath.Join("internal", "pkg", "db"),
	}

	for _, dir := range candidates {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".sql") {
				return dir, nil
			}
		}
	}

	return "", fmt.Errorf("schema sql directory not found, expected .sql files under ./internal/pkg/db or ./database")
}

// planSchemaUpgradeSQL 通过将目标数据库与临时数据库中的规范schema进行对比，计划增量SQL语句。
func planSchemaUpgradeSQL(conf MysqlConfig, host, portStr, sqlDir string) ([]string, []string, error) {
	adminDB, err := openSQLDB(host, portStr, conf.Mysql.User, conf.Mysql.Password, "")
	if err != nil {
		return nil, nil, err
	}
	defer adminDB.Close()

	targetDB, err := openSQLDB(host, portStr, conf.Mysql.User, conf.Mysql.Password, conf.Mysql.DbName)
	if err != nil {
		return nil, nil, err
	}
	defer targetDB.Close()

	tmpDBName := fmt.Sprintf("%s_tmp_bootcheck_%d", conf.Mysql.DbName, time.Now().Unix())
	if err := recreateDatabase(adminDB, tmpDBName); err != nil {
		return nil, nil, err
	}
	defer dropDatabase(adminDB, tmpDBName)

	tmpDB, err := openSQLDB(host, portStr, conf.Mysql.User, conf.Mysql.Password, tmpDBName)
	if err != nil {
		return nil, nil, err
	}
	defer tmpDB.Close()

	if err := execSQLFilesInDir(tmpDB, sqlDir); err != nil {
		return nil, nil, err
	}

	oldColMap, oldIdxMap, err := getTableInfoMap(targetDB)
	if err != nil {
		return nil, nil, err
	}
	newColMap, newIdxMap, err := getTableInfoMap(tmpDB)
	if err != nil {
		return nil, nil, err
	}

	newTblCreateSQLMap, err := getTableCreateSQLMap(tmpDB)
	if err != nil {
		return nil, nil, err
	}

	plan, warnings := buildAdditiveSchemaDiffSQL(
		oldColMap,
		newColMap,
		oldIdxMap,
		newIdxMap,
		newTblCreateSQLMap,
	)
	return plan, warnings, nil
}

// execSchemaUpgradeSQL 对目标数据库执行计划的SQL语句，每执行一条记录一条。
func execSchemaUpgradeSQL(conf MysqlConfig, host, portStr string, sqlList []string) error {
	db, err := openSQLDB(host, portStr, conf.Mysql.User, conf.Mysql.Password, conf.Mysql.DbName)
	if err != nil {
		return err
	}
	defer db.Close()

	recordFile, recordCloser := prepareMigrationRecord()
	if recordCloser != nil {
		defer recordCloser.Close()
	}

	for _, statement := range sqlList {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("exec failed, sql=%s, err=%w", statement, err)
		}
		if recordCloser != nil {
			recordOneSQL(recordFile, statement)
		}
	}
	return nil
}

// prepareMigrationRecord 准备迁移记录文件，返回文件句柄和关闭函数。
func prepareMigrationRecord() (*os.File, io.Closer) {
	folder := fmt.Sprintf("%v/record", putil.GetExecFolder())
	if err := os.MkdirAll(folder, 0755); err != nil {
		plogger.Errorf("Failed to create record folder: %v", err)
		return nil, nil
	}

	filename := fmt.Sprintf("%v/db_update_%s.sql", folder, time.Now().Format("20060102T150405"))
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		plogger.Errorf("Failed to open record file: %v", err)
		return nil, nil
	}
	return file, file
}

// recordOneSQL 追加写入一条SQL语句到记录文件。
func recordOneSQL(file *os.File, statement string) {
	content := statement + ";\n"
	if _, err := file.WriteString(content); err != nil {
		plogger.Errorf("Failed to write SQL record: %v", err)
	}
}

// buildAdditiveSchemaDiffSQL 生成仅用于新增表、列和索引的增量SQL，
// 对于检测到的删除或修改操作返回警告，需要人工审查。
func buildAdditiveSchemaDiffSQL(
	oldTblColMap, newTblColMap map[string]map[string]*columnInfo,
	oldTblIdxMap, newTblIdxMap map[string]map[string][]*indexInfo,
	newTblCreateSQLMap map[string]string,
) ([]string, []string) {
	plan := make([]string, 0)
	warnings := make([]string, 0)

	oldTblList := mapKeys(oldTblColMap)
	newTblList := mapKeys(newTblColMap)
	addTblList := strListExcept(newTblList, oldTblList)
	delTblList := strListExcept(oldTblList, newTblList)

	sort.Strings(addTblList)
	for _, tbl := range addTblList {
		createSQL, ok := newTblCreateSQLMap[tbl]
		if !ok || strings.TrimSpace(createSQL) == "" {
			warnings = append(warnings, fmt.Sprintf("new table %s detected but create sql not found", tbl))
			continue
		}
		plan = append(plan, createSQL)
		warnings = append(warnings, fmt.Sprintf("new table %s detected, create sql will be generated from temp schema", tbl))
	}

	for _, tbl := range delTblList {
		warnings = append(warnings, fmt.Sprintf("table %s exists in target but not in schema sql, skip drop (manual review required)", tbl))
	}

	for _, tblName := range newTblList {
		oldColMap, exists := oldTblColMap[tblName]
		if !exists {
			continue
		}
		newColMap := newTblColMap[tblName]

		oldCols := mapKeys(oldColMap)
		newCols := mapKeys(newColMap)

		delCols := strListExcept(oldCols, newCols)
		sort.Strings(delCols)
		for _, col := range delCols {
			warnings = append(warnings, fmt.Sprintf("column %s.%s exists in target but not in schema sql, skip drop (manual review required)", tblName, col))
		}

		addCols := strListExcept(newCols, oldCols)
		sort.Strings(addCols)
		for _, colName := range addCols {
			newCol := newColMap[colName]
			plan = append(plan, buildAddColumnSQL(tblName, newCol))
		}

		sort.Strings(newCols)
		for _, colName := range newCols {
			oldCol, ok := oldColMap[colName]
			if !ok {
				continue
			}
			newCol := newColMap[colName]
			if !isSameColumn(oldCol, newCol) {
				warnings = append(warnings,
					fmt.Sprintf("column changed %s.%s from [%s] to [%s], skip modify (manual review required)",
						tblName, colName, oldCol.typeInfo, newCol.typeInfo))
			}
		}
	}

	idxPlan, idxWarnings := buildIndexPlan(oldTblIdxMap, newTblIdxMap)
	plan = append(plan, idxPlan...)
	warnings = append(warnings, idxWarnings...)

	return deduplicateSQL(plan), warnings
}

// buildIndexPlan 生成用于新增索引的SQL，并收集关于删除或修改索引的警告。
func buildIndexPlan(oldTblIdxMap, newTblIdxMap map[string]map[string][]*indexInfo) ([]string, []string) {
	plan := make([]string, 0)
	warnings := make([]string, 0)

	for tblName, newIdxMap := range newTblIdxMap {
		oldIdxMap, ok := oldTblIdxMap[tblName]
		if !ok {
			for _, idxName := range sortedIndexNames(newIdxMap) {
				newIdx := newIdxMap[idxName]
				if len(newIdx) == 0 || newIdx[0].keyName == "PRIMARY" {
					continue
				}
				plan = append(plan, buildAddIndexSQL(tblName, newIdx))
			}
			continue
		}

		newIdxList := sortedIndexNames(newIdxMap)
		oldIdxList := sortedIndexNames(oldIdxMap)

		addIdxList := strListExcept(newIdxList, oldIdxList)
		for _, idxName := range addIdxList {
			newIdx := newIdxMap[idxName]
			if len(newIdx) == 0 || newIdx[0].keyName == "PRIMARY" {
				continue
			}
			plan = append(plan, buildAddIndexSQL(tblName, newIdx))
		}

		delIdxList := strListExcept(oldIdxList, newIdxList)
		for _, idxName := range delIdxList {
			if idxName == "PRIMARY" {
				continue
			}
			warnings = append(warnings,
				fmt.Sprintf("index %s.%s exists in target but not in schema sql, skip drop (manual review required)", tblName, idxName))
		}

		for _, idxName := range newIdxList {
			oldIdx, ok := oldIdxMap[idxName]
			if !ok {
				continue
			}
			newIdx := newIdxMap[idxName]
			if len(newIdx) == 0 || newIdx[0].keyName == "PRIMARY" {
				continue
			}
			if !isSameIndex(oldIdx, newIdx) {
				warnings = append(warnings,
					fmt.Sprintf("index changed %s.%s, skip modify (manual review required)", tblName, idxName))
			}
		}
	}

	return plan, warnings
}

// deduplicateSQL 从列表中移除重复和空的SQL语句。
func deduplicateSQL(sqlList []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(sqlList))
	for _, stmt := range sqlList {
		trimmed := strings.TrimSpace(stmt)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}

// mapKeys 提取并排序map中的所有key。
func mapKeys[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// strListExcept 返回left中存在但right中不存在的元素（集合差集）。
func strListExcept(left, right []string) []string {
	rightSet := make(map[string]bool, len(right))
	for _, v := range right {
		rightSet[v] = true
	}
	out := make([]string, 0)
	for _, v := range left {
		if !rightSet[v] {
			out = append(out, v)
		}
	}
	return out
}

// openSQLDB 建立到MySQL的连接并验证连通性。
func openSQLDB(host, port, user, password, dbName string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, port, dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// recreateDatabase 如果数据库存在则删除，然后创建一个新的数据库。
func recreateDatabase(adminDB *sql.DB, dbName string) error {
	if _, err := adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName)); err != nil {
		return err
	}
	_, err := adminDB.Exec(fmt.Sprintf("CREATE DATABASE `%s` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci", dbName))
	return err
}

// dropDatabase 删除数据库，失败时记录警告日志。
func dropDatabase(adminDB *sql.DB, dbName string) {
	if _, err := adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName)); err != nil {
		plogger.Warnf("drop temp db failed: %v", err)
	}
}

// execSQLFilesInDir 按字母顺序执行目录中的所有.sql文件。
func execSQLFilesInDir(db *sql.DB, sqlDir string) error {
	entries, err := os.ReadDir(sqlDir)
	if err != nil {
		return err
	}

	fileNames := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".sql") {
			continue
		}
		fileNames = append(fileNames, entry.Name())
	}
	sort.Strings(fileNames)

	for _, name := range fileNames {
		path := filepath.Join(sqlDir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		statements := strings.Split(string(content), ";")
		for _, statement := range statements {
			statement = strings.TrimSpace(statement)
			if statement == "" {
				continue
			}
			if _, err := db.Exec(statement); err != nil {
				return fmt.Errorf("exec schema sql file failed, file=%s, err=%w", path, err)
			}
		}
	}

	if len(fileNames) == 0 {
		return fmt.Errorf("no sql files found in %s", sqlDir)
	}
	return nil
}

type columnInfo struct {
	field        string
	typeInfo     string
	null         string
	key          string
	defaultValue sql.NullString
	extra        string
}

type indexInfo struct {
	table        string
	nonUnique    string
	keyName      string
	seqInIndex   int
	columnName   string
	collation    sql.NullString
	cardinality  sql.NullInt64
	subPart      sql.NullInt64
	packed       sql.NullString
	null         string
	indexType    string
	comment      string
	indexComment string
	visible      sql.NullString
	expression   sql.NullString
}

// getTableInfoMap 获取数据库中所有的表、列和索引信息。
func getTableInfoMap(db *sql.DB) (
	tblColMap map[string]map[string]*columnInfo,
	tblIdxMap map[string]map[string][]*indexInfo,
	err error,
) {
	tables, err := getTables(db)
	if err != nil {
		return nil, nil, err
	}

	tblColMap = make(map[string]map[string]*columnInfo)
	tblIdxMap = make(map[string]map[string][]*indexInfo)

	for _, tableName := range tables {
		columns, err := getTableColumns(db, tableName)
		if err != nil {
			return nil, nil, err
		}
		indexes, err := getTableIndexes(db, tableName)
		if err != nil {
			return nil, nil, err
		}
		tblColMap[tableName] = columns
		tblIdxMap[tableName] = indexes
	}

	return tblColMap, tblIdxMap, nil
}

// getTableCreateSQLMap 获取数据库中所有表的CREATE TABLE语句。
func getTableCreateSQLMap(db *sql.DB) (map[string]string, error) {
	tables, err := getTables(db)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(tables))
	for _, tableName := range tables {
		var name string
		var createSQL string
		if err := db.QueryRow(fmt.Sprintf("SHOW CREATE TABLE `%s`", tableName)).Scan(&name, &createSQL); err != nil {
			return nil, err
		}
		out[tableName] = createSQL
	}
	return out, nil
}

// getTables 获取数据库中所有表名并排序返回。
func getTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := make([]string, 0)
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	sort.Strings(tables)
	return tables, nil
}

// getTableColumns 获取表的列定义，以列名为索引。
func getTableColumns(db *sql.DB, tableName string) (map[string]*columnInfo, error) {
	query := fmt.Sprintf("SHOW COLUMNS FROM `%s`", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]*columnInfo)
	for rows.Next() {
		var col columnInfo
		if err := rows.Scan(&col.field, &col.typeInfo, &col.null, &col.key, &col.defaultValue, &col.extra); err != nil {
			return nil, err
		}
		out[col.field] = &col
	}
	return out, nil
}

// getTableIndexes 获取表的所有索引，以索引名为主键索引。
func getTableIndexes(db *sql.DB, tableName string) (map[string][]*indexInfo, error) {
	query := fmt.Sprintf("SHOW INDEX FROM `%s`", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string][]*indexInfo)
	for rows.Next() {
		var idx indexInfo
		if err := rows.Scan(
			&idx.table,
			&idx.nonUnique,
			&idx.keyName,
			&idx.seqInIndex,
			&idx.columnName,
			&idx.collation,
			&idx.cardinality,
			&idx.subPart,
			&idx.packed,
			&idx.null,
			&idx.indexType,
			&idx.comment,
			&idx.indexComment,
			&idx.visible,
			&idx.expression,
		); err != nil {
			return nil, err
		}
		out[idx.keyName] = append(out[idx.keyName], &idx)
	}

	for _, list := range out {
		sort.Slice(list, func(i, j int) bool {
			return list[i].seqInIndex < list[j].seqInIndex
		})
	}
	return out, nil
}

// isSameColumn 比较两个列定义是否相同。
func isSameColumn(oldCol, newCol *columnInfo) bool {
	if oldCol == nil || newCol == nil {
		return false
	}
	return oldCol.typeInfo == newCol.typeInfo &&
		oldCol.null == newCol.null &&
		oldCol.defaultValue.String == newCol.defaultValue.String &&
		oldCol.defaultValue.Valid == newCol.defaultValue.Valid &&
		oldCol.extra == newCol.extra
}

// buildAddColumnSQL 生成ALTER TABLE ADD COLUMN语句。
func buildAddColumnSQL(tblName string, newCol *columnInfo) string {
	defaultVal := ""
	if newCol.defaultValue.Valid {
		dv := newCol.defaultValue.String
		if strings.EqualFold(dv, "NULL") {
			defaultVal = "DEFAULT NULL"
		} else if strings.Contains(strings.ToLower(newCol.typeInfo), "int") ||
			(strings.Contains(strings.ToLower(newCol.typeInfo), "datetime") && dv == "CURRENT_TIMESTAMP") {
			defaultVal = "DEFAULT " + dv
		} else {
			defaultVal = "DEFAULT '" + strings.ReplaceAll(dv, "'", "''") + "'"
		}
	}

	notNull := ""
	if strings.EqualFold(newCol.null, "NO") {
		notNull = "NOT NULL"
	}

	return strings.TrimSpace(fmt.Sprintf(
		"ALTER TABLE `%s` ADD COLUMN `%s` %s %s %s",
		tblName,
		newCol.field,
		newCol.typeInfo,
		notNull,
		defaultVal,
	))
}

// buildAddIndexSQL 生成ALTER TABLE ADD INDEX语句。
func buildAddIndexSQL(tblName string, newIdx []*indexInfo) string {
	colList := make([]string, 0, len(newIdx))
	for _, item := range newIdx {
		colList = append(colList, "`"+item.columnName+"`")
	}

	idxType := "INDEX"
	if newIdx[0].nonUnique == "0" {
		idxType = "UNIQUE KEY"
	}
	if strings.EqualFold(newIdx[0].indexType, "FULLTEXT") {
		idxType = "FULLTEXT"
	}

	idxName := newIdx[0].keyName
	return fmt.Sprintf(
		"ALTER TABLE `%s` ADD %s `%s` (%s)",
		tblName,
		idxType,
		idxName,
		strings.Join(colList, ","),
	)
}

// isSameIndex 比较两个索引定义是否相同。
func isSameIndex(oldIdx, newIdx []*indexInfo) bool {
	if len(oldIdx) != len(newIdx) {
		return false
	}
	for i := range oldIdx {
		o := oldIdx[i]
		n := newIdx[i]
		if o.keyName != n.keyName ||
			o.nonUnique != n.nonUnique ||
			o.seqInIndex != n.seqInIndex ||
			o.columnName != n.columnName ||
			o.indexType != n.indexType {
			return false
		}
	}
	return true
}

// sortedIndexNames 从map中返回排序后的索引名列表。
func sortedIndexNames(m map[string][]*indexInfo) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
