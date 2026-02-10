package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/pdb"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

type dbModel interface {
	TableName() string
}

// 处理复合键、多个索引的情况
type IndexField struct {
	IdxColName   string // 索引列名，model字段名
	IdxColType   string // 索引列类型，model字段类型
	IdxProtoName string // 索引列名，读写值的参数名
}

type IndexInfo struct {
	Name   string        // 索引名
	Fields []*IndexField // 索引包含的字段
}

type Table struct {
	ServiceName string
	Model       dbModel

	//生成代码需要的值

	HyphenName     string // 中横线[-]命名
	LowerCamelName string // 驼峰命名，首字母小写
	UpperCamelName string // 驼峰命名，首字母大写

	PriIdxColName   string // 索引列名，model字段名
	PriIdxColType   string // 索引列类型，model字段类型
	PriIdxProtoName string // 索引列名，读写值的参数名

	IdxList []*IndexInfo // 存储所有唯一索引

	FieldList []*reflect.StructField
}

func (t *Table) String() string {
	return fmt.Sprintf("tbl[%v] ServiceName[%v] "+
		"HyphenName[%v] LowerCamelName[%v] UpperCamelName[%v] "+
		"IdxColName[%v] IdxColType[%v] IdxParmName[%v]",
		t.Model.TableName(), t.ServiceName,
		t.HyphenName, t.LowerCamelName, t.UpperCamelName,
		t.PriIdxColName, t.PriIdxColType, t.PriIdxProtoName)
}

// --------------------------------------------------
var tblMap = make(map[string]*Table)

func addTable(m dbModel, svcName string) {
	tblMap[m.TableName()] = newTable(m, svcName)
}
func newTable(m dbModel, svcName string) *Table {
	tbl := Table{
		ServiceName: svcName,
		Model:       m,
	}
	tblName := tbl.Model.TableName()
	tbl.HyphenName = strings.ReplaceAll(tblName, "_", "-")
	tbl.UpperCamelName = putil.StrToCamelCase(tblName)
	tbl.LowerCamelName = putil.StrFirstToLower(tbl.UpperCamelName)
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

	// 通过数据库读取所有表、列和唯一索引信息（废弃基于 orm 的反射）

	// 获取当前数据库下的所有表名（使用 pdb 封装）
	tables, err := pdb.GetGormDB().Migrator().GetTables()
	if err != nil {
		log.Fatalf("get tables failed: %v", err)
	}
	for _, tblName := range tables {
		if tblName == "abandon_code" {
			continue // 模板表不处理
		}
		addTable(&SimpleModel{name: tblName}, inferServiceName(tblName))
	}

	// 读取每个表的列信息和唯一索引信息
	for _, tbl := range tblMap {
		plogger.Debugf("%v---------------------------", tbl.Model.TableName())
		isMultiPriKey := false

		// 获取列信息（通过 pdb 封装）
		cols, err := pdb.GetGormDB().Migrator().ColumnTypes(tbl.Model.TableName())
		if err != nil {
			plogger.Errorf("get columns failed: %v", err)
			return
		}
		for _, c := range cols {
			fieldName := putil.StrToCamelCase(c.Name())
			if strings.HasSuffix(fieldName, "Id") { // 统一把Id改成ID
				fieldName = strings.TrimSuffix(fieldName, "Id") + "ID"
			} else if strings.HasSuffix(fieldName, "Url") {
				fieldName = strings.TrimSuffix(fieldName, "Url") + "URL"
			}
			// 构造 reflect.StructField 用于后续索引匹配
			t := c.ScanType()
			if t.String() == "sql.NullTime" {
				t = reflect.TypeOf(time.Time{})
			}
			f := reflect.StructField{
				Name: fieldName,
				Type: t,
				Tag:  "",
			}
			tbl.FieldList = append(tbl.FieldList, &f)

			plogger.Debugf("Field[%s] Type[%s] sqlType[%v]",
				fieldName, c.ScanType().String(), c.DatabaseTypeName())
			is, ok := c.PrimaryKey()
			if ok && is {
				if tbl.PriIdxColName != "" {
					isMultiPriKey = true
				}
				tbl.PriIdxColName = fieldName
				tbl.PriIdxColType = c.ScanType().String()
				tbl.PriIdxProtoName = putil.StrFirstToLower(tbl.PriIdxColName)
			}
		}

		// 尝试从数据库获取索引信息
		indexes, err := pdb.GetGormDB().Migrator().GetIndexes(tbl.Model.TableName())
		if err != nil {
			plogger.Errorf("get indexes failed: %v", err)
			return
		}
		plogger.Debugf("found indexes num: %d", len(indexes))
		for _, idx := range indexes {
			pk, _ := idx.PrimaryKey()
			unique, _ := idx.Unique()
			plogger.Debugf("Index Name[%s] Primary[%v] Unique[%v]",
				idx.Name(), pk, unique)
			if !unique {
				continue
			}
			var idxFields []*IndexField
			for _, col := range idx.Columns() {
				// 找到对应的 model 字段
				for _, f := range tbl.FieldList {
					if !strings.EqualFold(f.Name, col) {
						continue
					}
					idxFields = append(idxFields, &IndexField{
						IdxColName:   f.Name, //ID
						IdxColType:   f.Type.String(),
						IdxProtoName: StrFirstToLowerButID(f.Name),
					})
					break
				}
			}
			if len(idxFields) == 0 {
				continue
			}
			tbl.IdxList = append(tbl.IdxList, &IndexInfo{
				Name:   idx.Name(),
				Fields: idxFields,
			})
		}

		if isMultiPriKey { //TODO
			tbl.PriIdxColName = ""
			tbl.PriIdxColType = ""
			tbl.PriIdxProtoName = ""
		}
		plogger.Debug("tbl info : ", tbl)
	}

	tplTable := newTable(&SimpleModel{name: "abandon_code"}, "abandonCode")
	tplTable.PriIdxColName = "Idx1"
	tplTable.PriIdxColType = "int32"
	tplTable.PriIdxProtoName = "idx1"

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
	return "default"
}

// SimpleModel 用于在没有 orm struct 的情况下提供 TableName
type SimpleModel struct {
	name string
}

func (s *SimpleModel) TableName() string {
	return s.name
}
