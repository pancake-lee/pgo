package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pancake-lee/pgo/pkg/pconfig"
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

	//生成代码需要的值
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

func addTable(tblName string, svcName string) {
	tblMap[tblName] = newTable(tblName, svcName)
}
func newTable(tblName string, svcName string) *Table {
	tbl := Table{
		TblName:     tblName,
		ServiceName: svcName,
	}
	tbl.HyphenName = strings.ReplaceAll(tblName, "_", "-")
	tbl.UpperCamelName = putil.StrToCamelCase(tblName)
	tbl.LowerCamelName = putil.StrFirstToLower(tbl.UpperCamelName)

	cols, err := pdb.GetGormDB().Migrator().ColumnTypes(tblName)
	if err != nil {
		plogger.Errorf("get columns failed: %v", err)
		panic(err)
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

		// TODO 在abandonCode扩展所有数据库类型的映射
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

	if isMultiPriKey { //TODO
		plogger.Warnf("found multi pri key, skip")
		tbl.PriCol = nil
	}

	// --------------------------------------------------

	// 尝试从数据库获取索引信息
	idxList, err := pdb.GetGormDB().Migrator().GetIndexes(tblName)
	if err != nil {
		plogger.Errorf("get indexes failed: %v", err)
		panic(err)
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
				panic(fmt.Sprintf("idxCol[%v] for idx[%v] not found", idxColName, idx.Name))
			}
			idx.Fields = append(idx.Fields, idxCol)
		}
		tbl.IdxList = append(tbl.IdxList, idx)
	}

	plogger.Debugf("found indexes num: %d", len(tbl.IdxList))

	plogger.Debugf("got %v table info---------------------------", tblName)
	return &tbl
}

// --------------------------------------------------
// 1：连接数据库
// 2：获取数据库表结构
//    但是当前部分逻辑通过orm结构来获取，应该废弃
//    本来想orm对于编码来说更加准确，但是orm结构并不包含索引等信息
// 3：根据internal/abandonCodeService的代码以及代码中的标记，生成dao/pb/service代码

func main() {
	l := flag.Bool("l", false, "log to console, default is false")
	c := flag.String("c", "./configs/pancake.yaml",
		"config folder, should have common.yaml and ${execName}.yaml")
	flag.Parse()

	rmAllGenFile()

	// TODO 应该和makefile的orm临时表保持一致，而不是读取配置文件运行时的数据库
	pconfig.MustInitConfig(*c)
	plogger.InitFromConfig(*l)
	pdb.MustInitMysqlByConfig()

	// --------------------------------------------------
	// 获取当前数据库下的所有表名（使用 pdb 封装）
	tables, err := pdb.GetGormDB().Migrator().GetTables()
	if err != nil {
		log.Fatalf("get tables failed: %v", err)
	}
	for _, tblName := range tables {
		if tblName == "abandon_code" {
			continue // 模板表不处理
		}
		addTable(tblName, inferServiceName(tblName))
	}

	// --------------------------------------------------
	tplTable := newTable("abandon_code", "abandonCode")

	// --------------------------------------------------
	genDaoCode(tblMap, tplTable)
	genProto(tblMap, tplTable)

	// 调用 protoc 生成 go 代码
	cmd := exec.Command("make", "api")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("make proto failed, err: %v, out: \n%v", err, string(out))
	}

	genServiceCode(tblMap, tplTable)
}

// gorm-gen 工具生成的字段采用ID命名
// protoc   工具生成的字段采用Id命名
// 使用起来依然太复杂，简单的代码替换难以分析是orm代码还是pb代码
// 最后是在proto中也用ID命名，所以生成的代码就统一是ID，只要首字母小写时遇到ID开头就不转换即可
func StrFirstToLowerButID(f string) string {
	if strings.HasPrefix(f, "ID") {
		return f
	}
	return putil.StrFirstToLower(f)
}

func rmAllGenFile() {
	filepath.Walk("internal",
		func(path string, info os.FileInfo, err error) error {
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
	filepath.Walk("proto",
		func(path string, info os.FileInfo, err error) error {
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

// TODO 把sql存储到每个service内部，然后表和模块的对应关系就通过路径分析出来
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

// SimpleModel 用于在没有 orm struct 的情况下提供 TableName
type SimpleModel struct {
	name string
}

func (s *SimpleModel) TableName() string {
	return s.name
}
