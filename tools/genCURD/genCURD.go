package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/pancake-lee/pgo/internal/pkg/db/model"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

type dbModel interface {
	TableName() string
}

// 处理复合键、多个索引的情况，可能要重构了，只是字符串替换的话，不太好处理
// type IndexField struct {
// 	IdxColName  string // 索引列名，model字段名
// 	IdxColType  string // 索引列类型，model字段类型
// 	IdxParmName string // 索引列名，读写值的参数名
// }

type Table struct {
	ServiceName string
	Model       dbModel

	//生成代码需要的值

	HyphenName     string // 中横线[-]命名
	LowerCamelName string // 驼峰命名，首字母小写
	UpperCamelName string // 驼峰命名，首字母大写

	// IdxMap      map[string][]*IndexField // 用map是为了以后只要是唯一索引都可以生成代码
	IdxColName  string // 索引列名，model字段名
	IdxColType  string // 索引列类型，model字段类型
	IdxParmName string // 索引列名，读写值的参数名

	FieldList []*reflect.StructField
}

func (t *Table) String() string {
	return fmt.Sprintf("tbl[%v] ServiceName[%v] "+
		"HyphenName[%v] LowerCamelName[%v] UpperCamelName[%v] "+
		"IdxColName[%v] IdxColType[%v] IdxParmName[%v]",
		t.Model.TableName(), t.ServiceName,
		t.HyphenName, t.LowerCamelName, t.UpperCamelName,
		t.IdxColName, t.IdxColType, t.IdxParmName)
}

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

func main() {
	log.SetFlags(log.Lshortfile | log.Ltime)

	rmAllGenFile()

	addTable(&model.User{}, "user")
	addTable(&model.UserDept{}, "user")
	addTable(&model.UserDeptAssoc{}, "user")
	addTable(&model.UserJob{}, "user")
	addTable(&model.CourseSwapRequest{}, "school")
	addTable(&model.Task{}, "task")

	//读取数据库表结构
	for _, tbl := range tblMap {
		plogger.Debugf("%v---------------------------", tbl.Model.TableName())
		isMultiKey := false
		val := reflect.ValueOf(tbl.Model).Elem()
		for i := 0; i < val.NumField(); i++ {
			field := val.Type().Field(i)
			tbl.FieldList = append(tbl.FieldList, &field)

			plogger.Debugf("Field[%s] Type[%s] Tag[%v]", field.Name, field.Type, field.Tag)
			if strings.Contains(field.Tag.Get("gorm"), "primaryKey") {
				if tbl.IdxColName != "" {
					isMultiKey = true
				}
				tbl.IdxColName = field.Name
				tbl.IdxColType = field.Type.String()
				tbl.IdxParmName = putil.StrFirstToLower(putil.StrIdToLower(tbl.IdxColName))
			}
		}
		if isMultiKey { //TODO
			tbl.IdxColName = ""
			tbl.IdxColType = ""
			tbl.IdxParmName = ""
		}
		plogger.Debug("tbl info : ", tbl)
	}

	tplTable := newTable(&model.AbandonCode{}, "abandonCode")
	tplTable.IdxColName = "Idx1"
	tplTable.IdxColType = "int32"
	tplTable.IdxParmName = "idx1"

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

func DO2DTO_FieldName(f string) string {
	return putil.StrFirstToLower(putil.StrIdToLower(f))
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
