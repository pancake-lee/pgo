package genCURD

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jinzhu/inflection"
	"github.com/pancake-lee/pgo/pkg/pdb"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
	"gorm.io/gorm"
)

type indexInfo struct {
	originIdx gorm.Index
	Name      string     // 索引名
	Fields    []*colInfo // 索引包含的字段
}

type colInfo struct {
	originCol gorm.ColumnType // field_name

	ormFieldName string // FieldName
	ormFieldType string
	apiFieldName string // FieldName
	apiFieldType string
	pbFieldName  string // fieldName
	pbFieldType  string
}

type Table struct {
	TblName     string
	ServiceName string
	ColList     []*colInfo
	PriCol      *colInfo // 暂时只支持单主键，符合主键后续扩展
	IdxList     []indexInfo

	// 生成代码需要的值
	HyphenName     string // 中横线[-]命名
	LowerCamelName string // 驼峰命名，首字母小写
	UpperCamelName string // 驼峰命名，首字母大写
	SnakeName      string // 单数 snake_case，用于 proto 字段名
}

func (t *Table) String() string {
	return fmt.Sprintf("tbl[%v] ServiceName[%v] "+
		"HyphenName[%v] LowerCamelName[%v] UpperCamelName[%v] "+
		"IdxColName[%v] IdxColType[%v] IdxParmName[%v]",
		t.TblName, t.ServiceName,
		t.HyphenName, t.LowerCamelName, t.UpperCamelName,
		t.PriCol.ormFieldName, t.PriCol.ormFieldType, t.PriCol.ormFieldName)
}

// --------------------------------------------------
var tblMap = make(map[string]*Table)

func addTable(tblName string, svcName string) error {
	tbl, err := newTable(tblName, svcName)
	if err != nil {
		return err
	}
	tblMap[tblName] = tbl
	return nil
}

func newTable(tblName string, svcName string) (*Table, error) {
	tbl := Table{
		TblName:     tblName,
		ServiceName: svcName,
	}
	tbl.HyphenName = strings.ReplaceAll(tblName, "_", "-")
	tbl.UpperCamelName = inflection.Singular(putil.StrToCamelCase(tblName))
	tbl.LowerCamelName = putil.StrFirstToLower(tbl.UpperCamelName)
	tbl.SnakeName = inflection.Singular(tblName)

	cols, err := pdb.GetGormDB().Migrator().ColumnTypes(tblName)
	if err != nil {
		return nil, fmt.Errorf("get columns failed: %w", err)
	}

	isMultiPriKey := false
	for _, originCol := range cols {
		var c colInfo
		c.originCol = originCol

		fieldName := putil.StrToCamelCase(originCol.Name())
		if strings.HasSuffix(fieldName, "Id") { // 统一把Id改成ID
			fieldName = strings.TrimSuffix(fieldName, "Id") + "ID"
		} else if strings.HasSuffix(fieldName, "Url") {
			fieldName = strings.TrimSuffix(fieldName, "Url") + "URL"
		}
		c.ormFieldName = fieldName
		c.apiFieldName = fieldName
		c.pbFieldName = StrFirstToLowerButID(fieldName)

		scanTypeName, err := getColumnScanTypeName(originCol)
		if err != nil {
			return nil, err
		}
		c.ormFieldType = scanTypeName
		c.apiFieldType = scanTypeName
		c.pbFieldType = scanTypeName

		// SQLite3 驱动返回 sql.Null* 类型，归一化为基本类型
		c.ormFieldType = normalizeSqlNullType(c.ormFieldType)
		c.apiFieldType = normalizeSqlNullType(c.apiFieldType)
		c.pbFieldType = normalizeSqlNullType(c.pbFieldType)

		if strings.EqualFold(originCol.DatabaseTypeName(), "date") ||
			strings.EqualFold(originCol.DatabaseTypeName(), "datetime") {
			c.ormFieldType = "time.Time"
			c.apiFieldType = "int64"
			c.pbFieldType = "int64"
		}
		if strings.EqualFold(originCol.DatabaseTypeName(), "DECIMAL") {
			c.ormFieldType = "decimal.Decimal"
			c.apiFieldType = "string"
			c.pbFieldType = "string"
		}

		// 将 Go 类型映射为合法的 proto 类型
		c.pbFieldType = goTypeToPBTYPE(c.pbFieldType)

		plogger.Debugf("Field[%s] Type[%s] sqlType[%v] orm[%v][%v] api[%v][%v]",
			originCol.Name(), scanTypeName,
			originCol.DatabaseTypeName(),
			c.ormFieldName, c.ormFieldType, c.apiFieldName, c.apiFieldType)

		tbl.ColList = append(tbl.ColList, &c)

		is, ok := originCol.PrimaryKey()
		if ok && is {
			if tbl.PriCol != nil {
				isMultiPriKey = true
			}
			tbl.PriCol = &c
		}
	}

	if isMultiPriKey {
		plogger.Warnf("found multi pri key, skip")
		tbl.PriCol = nil
	}

	idxList, err := pdb.GetGormDB().Migrator().GetIndexes(tblName)
	if err != nil {
		return nil, fmt.Errorf("get indexes failed: %w", err)
	}
	for _, originIdx := range idxList {
		var idx indexInfo
		idx.originIdx = originIdx
		idx.Name = originIdx.Name()

		for _, idxColName := range originIdx.Columns() {
			var idxCol *colInfo
			for _, c := range tbl.ColList {
				if c.originCol.Name() == idxColName {
					idxCol = c
					break
				}
			}

			if idxCol == nil {
				return nil, fmt.Errorf("idxCol[%v] for idx[%v] not found", idxColName, idx.Name)
			}
			idx.Fields = append(idx.Fields, idxCol)
		}
		tbl.IdxList = append(tbl.IdxList, idx)
	}

	plogger.Debugf("found indexes num: %d", len(tbl.IdxList))
	plogger.Debugf("got %v table info---------------------------", tblName)
	return &tbl, nil
}

func getColumnScanTypeName(originCol gorm.ColumnType) (string, error) {
	if scanType := originCol.ScanType(); scanType != nil {
		return scanType.String(), nil
	}
	switch strings.ToUpper(originCol.DatabaseTypeName()) {
	case "INT", "INTEGER", "BIGINT":
		return "int64", nil
	case "FLOAT", "DOUBLE", "REAL":
		return "float64", nil
	case "BOOL", "BOOLEAN":
		return "bool", nil
	case "TEXT", "VARCHAR", "CHAR":
		return "string", nil
	default:
		return "", fmt.Errorf("column %q has no Go scan type for database type %q", originCol.Name(), originCol.DatabaseTypeName())
	}
}

// --------------------------------------------------
func runGenerate(dbType, dsn string) error {
	tblMap = make(map[string]*Table)
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get project root failed: %w", err)
	}

	switch dbType {
	case "mysql":
		err = pdb.InitMysqlByDsn(dsn)
	case "sqlite3":
		err = pdb.InitSqlite(dsn)
	default:
		return fmt.Errorf("db %q is not supported, only mysql and sqlite3 are supported", dbType)
	}
	if err != nil {
		return err
	}

	tableNameList, err := pdb.GetGormDB().Migrator().GetTables()
	if err != nil {
		return fmt.Errorf("get tables failed: %w", err)
	}
	tableToServiceMap, err := readTableToServiceMap(tableNameList)
	if err != nil {
		return err
	}
	for _, tblName := range tableNameList {
		if tblName == "abandon_code" {
			continue // 模板表不处理
		}
		serviceName := tableToServiceMap[tblName]
		if serviceName == "" {
			serviceName = "default"
		}
		err = addTable(tblName, serviceName)
		if err != nil {
			return err
		}
	}

	tplTable, err := newTable("abandon_code", "abandonCode")
	if err != nil {
		return err
	}
	session, err := newGenerationSession(".")
	if err != nil {
		return err
	}
	defer func() {
		if err := session.rollback(); err != nil {
			plogger.Error("rollback generated files failed: ", err)
		}
	}()

	err = genDaoCode(tblMap, tplTable)
	if err != nil {
		return err
	}

	err = genProto(tblMap, tplTable)
	if err != nil {
		return err
	}
	err = removeStaleGeneratedProtoFiles(tblMap)
	if err != nil {
		return err
	}

	err = runAPIGenerator(projectRoot)
	if err != nil {
		return err
	}

	err = genServiceCode(tblMap, tplTable)
	if err != nil {
		return err
	}

	err = genMainCode(tblMap, tplTable)
	if err != nil {
		return err
	}
	if err = session.commit(); err != nil {
		return err
	}
	logTableMappingSummary(tblMap)
	return nil
}

func removeStaleGeneratedProtoFiles(tblToSvrMap map[string]*Table) error {
	expectedPathMap := make(map[string]struct{})
	for _, tbl := range tblToSvrMap {
		expectedPathMap[filepath.Join("proto", "z_"+tbl.ServiceName+"Service.gen.proto")] = struct{}{}
	}
	return filepath.Walk("proto", func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || !strings.HasSuffix(path, ".gen.proto") {
			return nil
		}
		if _, ok := expectedPathMap[path]; ok {
			return nil
		}
		return os.Remove(path)
	})
}

func logTableMappingSummary(tblToSvrMap map[string]*Table) {
	serviceNameSet := make(map[string]struct{})
	for _, tbl := range tblToSvrMap {
		serviceNameSet[tbl.ServiceName] = struct{}{}
	}
	var serviceNameList []string
	for serviceName := range serviceNameSet {
		serviceNameList = append(serviceNameList, serviceName)
	}
	sort.Strings(serviceNameList)
	for _, serviceName := range serviceNameList {
		plogger.Infof("genCURD service[%s] tables=%s", serviceName,
			strings.Join(tableNameListForService(tblToSvrMap, serviceName), ","))
	}
	if defaultTableNameList := tableNameListForService(tblToSvrMap, "default"); len(defaultTableNameList) > 0 {
		plogger.Infof("genCURD default service contains unmapped tables=%s; move their pgo.tables options to a custom service proto and rerun", strings.Join(defaultTableNameList, ","))
	}
}

func getProtoFileList(projectRoot string) ([]string, error) {
	protoRoot := filepath.Join(projectRoot, "proto")
	var protoFileList []string
	err := filepath.Walk(protoRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || filepath.Ext(path) != ".proto" {
			return nil
		}
		relativePath, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return err
		}
		protoFileList = append(protoFileList, "./"+filepath.ToSlash(relativePath))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("collect proto files failed: %w", err)
	}
	if len(protoFileList) == 0 {
		return nil, fmt.Errorf("no proto files found under %s", protoRoot)
	}
	sort.Strings(protoFileList)
	return protoFileList, nil
}

func runMakeApi(projectRoot string) error {
	protoFileList, err := getProtoFileList(projectRoot)
	if err != nil {
		return err
	}
	cmd := exec.Command("make", "api", "API_PROTO_FILES="+strings.Join(protoFileList, " "))
	cmd.Dir = projectRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("make api failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

var runAPIGenerator = runMakeApi

func StrFirstToLowerButID(f string) string {
	if strings.HasPrefix(f, "ID") {
		return f
	}
	return putil.StrFirstToLower(f)
}

type generationSession struct {
	root        string
	backupDir   string
	snapshotMap map[string]struct{}
	committed   bool
}

func newGenerationSession(root string) (*generationSession, error) {
	backupDir, err := os.MkdirTemp("", "pgo-gencurd-")
	if err != nil {
		return nil, fmt.Errorf("create generated-file backup: %w", err)
	}
	s := &generationSession{root: root, backupDir: backupDir, snapshotMap: make(map[string]struct{})}
	pathList, err := s.generatedPathList()
	if err != nil {
		_ = os.RemoveAll(backupDir)
		return nil, err
	}
	for _, path := range pathList {
		if err = s.copyToBackup(path); err != nil {
			_ = os.RemoveAll(backupDir)
			return nil, err
		}
		s.snapshotMap[path] = struct{}{}
	}
	return s, nil
}

func (s *generationSession) generatedPathList() ([]string, error) {
	var pathList []string
	for _, dir := range []string{"internal", "proto"} {
		base := filepath.Join(s.root, dir)
		err := filepath.Walk(base, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				if os.IsNotExist(walkErr) {
					return nil
				}
				return walkErr
			}
			if !info.IsDir() && isGeneratedPath(filepath.ToSlash(path)) {
				pathList = append(pathList, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	openAPIPath := filepath.Join(s.root, "openapi.yaml")
	if _, err := os.Stat(openAPIPath); err == nil {
		pathList = append(pathList, openAPIPath)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return pathList, nil
}

func isGeneratedPath(path string) bool {
	return strings.HasSuffix(path, ".gen.go") ||
		strings.HasSuffix(path, ".gen.proto") ||
		strings.Contains(path, "internal/pkg/api/") && strings.HasSuffix(path, ".pb.go")
}

func (s *generationSession) copyToBackup(path string) error {
	relPath, err := filepath.Rel(s.root, path)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	backupPath := filepath.Join(s.backupDir, relPath)
	if err = os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(backupPath, content, 0644)
}

func (s *generationSession) rollback() error {
	if s.committed {
		return os.RemoveAll(s.backupDir)
	}
	pathList, err := s.generatedPathList()
	if err != nil {
		return err
	}
	for _, path := range pathList {
		if err = os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	for path := range s.snapshotMap {
		relPath, err := filepath.Rel(s.root, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(filepath.Join(s.backupDir, relPath))
		if err != nil {
			return err
		}
		if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err = os.WriteFile(path, content, 0644); err != nil {
			return err
		}
	}
	return os.RemoveAll(s.backupDir)
}

func (s *generationSession) commit() error {
	pathList, err := s.generatedPathList()
	if err != nil {
		return err
	}
	pathMap := make(map[string]struct{}, len(pathList))
	for _, path := range pathList {
		pathMap[path] = struct{}{}
	}
	for path := range s.snapshotMap {
		if _, ok := pathMap[path]; !ok {
			if err = os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	s.committed = true
	return os.RemoveAll(s.backupDir)
}

// goTypeToPBTYPE 将 Go 类型映射为合法的 proto 类型
func goTypeToPBTYPE(t string) string {
	switch t {
	case "float64":
		return "double"
	case "float32":
		return "float"
	default:
		return t
	}
}

// normalizeSqlNullType 将 SQLite3 驱动返回的 sql.Null* 类型归一化为基本 Go 类型
func normalizeSqlNullType(t string) string {
	if !strings.HasPrefix(t, "sql.Null") {
		return t
	}
	switch t {
	case "sql.NullString":
		return "string"
	case "sql.NullInt64":
		return "int32"
	case "sql.NullInt32":
		return "int32"
	case "sql.NullInt16":
		return "int32"
	case "sql.NullByte":
		return "int32"
	case "sql.NullFloat64":
		return "float64"
	case "sql.NullBool":
		return "bool"
	case "sql.NullTime":
		return "time.Time"
	default:
		// sql.Null 开头的未知类型，回退为 string
		return "string"
	}
}
