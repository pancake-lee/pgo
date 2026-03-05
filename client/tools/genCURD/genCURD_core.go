package genCURD

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	tbl.UpperCamelName = putil.StrToCamelCase(tblName)
	tbl.LowerCamelName = putil.StrFirstToLower(tbl.UpperCamelName)

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

		c.ormFieldType = originCol.ScanType().String()
		c.apiFieldType = originCol.ScanType().String()
		c.pbFieldType = originCol.ScanType().String()

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

		plogger.Debugf("Field[%s] Type[%s] sqlType[%v] orm[%v][%v] api[%v][%v]",
			originCol.Name(), originCol.ScanType().String(),
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

// --------------------------------------------------
func runGenerate(dsn string) error {
	tblMap = make(map[string]*Table)

	err := rmAllGenFile()
	if err != nil {
		return err
	}

	err = pdb.InitMysqlByDsn(dsn)
	if err != nil {
		return err
	}

	tables, err := pdb.GetGormDB().Migrator().GetTables()
	if err != nil {
		return fmt.Errorf("get tables failed: %w", err)
	}
	for _, tblName := range tables {
		if tblName == "abandon_code" {
			continue // 模板表不处理
		}
		err = addTable(tblName, inferServiceName(tblName))
		if err != nil {
			return err
		}
	}

	tplTable, err := newTable("abandon_code", "abandonCode")
	if err != nil {
		return err
	}

	err = genDaoCode(tblMap, tplTable)
	if err != nil {
		return err
	}

	err = genProto(tblMap, tplTable)
	if err != nil {
		return err
	}

	err = runMakeApi()
	if err != nil {
		return err
	}

	return genServiceCode(tblMap, tplTable)
}

func runMakeApi() error {
	cmd := exec.Command("make", "api")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("make api failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func StrFirstToLowerButID(f string) string {
	if strings.HasPrefix(f, "ID") {
		return f
	}
	return putil.StrFirstToLower(f)
}

func rmAllGenFile() error {
	err := filepath.Walk("internal", func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if strings.Contains(path, "pkg") {
			return nil // 不删除 pkg 目录下的文件
		}
		if strings.Contains(path, "gen.go") {
			plogger.Debug("rm file: ", path)
			err := os.Remove(path)
			if err != nil {
				plogger.Debug("rm file failed: ", err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return filepath.Walk("proto", func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if strings.Contains(path, "gen.proto") {
			plogger.Debug("rm file: ", path)
			err := os.Remove(path)
			if err != nil {
				plogger.Debug("rm file failed: ", err)
			}
		}
		return nil
	})
}

func inferServiceName(tableName string) string {
	if strings.HasPrefix(tableName, "task") {
		return "task"
	}
	if strings.HasPrefix(tableName, "course") {
		return "school"
	}
	if strings.HasPrefix(tableName, "abandon") {
		return "abandonCode"
	}
	if strings.HasPrefix(tableName, "user") {
		return "user"
	}
	if strings.HasPrefix(tableName, "proj") {
		return "user"
	}
	return "default"
}
