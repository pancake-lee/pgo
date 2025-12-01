package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/pancake-lee/pgo/internal/pkg/db"
	"github.com/pancake-lee/pgo/internal/pkg/db/model"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/pdb"
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

	pconfig.MustInitConfig("./configs/pancake.yaml")
	pdb.MustInitPGByConfig()

	// 通过反射获取所有表
	q := db.GetPG()
	qVal := reflect.ValueOf(q).Elem()
	for i := 0; i < qVal.NumField(); i++ {
		field := qVal.Type().Field(i)
		plogger.Debugf("reflect field[%s] Type[%s]", field.Name, field.Type)
		if field.Name == "db" ||
			field.Name == putil.StrToCamelCase(
				(&model.AbandonCode{}).TableName()) {
			continue
		}
		// DAO 对象有 WithContext 方法返回 DO 对象
		method := qVal.Field(i).Addr().MethodByName("WithContext")
		if !method.IsValid() {
			plogger.Debugf("reflect field[%s] not found WithContext func", field.Name)
			continue
		}

		// Call WithContext
		res := method.Call([]reflect.Value{reflect.ValueOf(context.Background())})
		if len(res) == 0 {
			continue
		}
		doVal := res[0]

		// 获取 Create 方法
		createMethod := doVal.MethodByName("Create")
		if !createMethod.IsValid() {
			plogger.Debugf("reflect field[%s] DO not found Create func", field.Name)
			continue
		}

		// 获取 Create 方法的第一个参数类型 ...*model.User
		// In(0) 是 []*model.User
		// Elem() 是 *model.User
		// Elem() 是 model.User
		modelType := createMethod.Type().In(0).Elem().Elem()

		// 创建实例
		m := reflect.New(modelType).Interface().(dbModel)

		addTable(m, inferServiceName(m.TableName()))
	}

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

		// 尝试从数据库获取索引信息
		indexes, err := pdb.GetGormDB().Migrator().GetIndexes(tbl.Model)
		if err != nil {
			plogger.Errorf("get indexes failed: %v", err)
		} else {
			for _, idx := range indexes {
				pk, _ := idx.PrimaryKey()
				unique, _ := idx.Unique()
				plogger.Debugf("Index Name[%s] Primary[%v] Unique[%v]",
					idx.Name(), pk, unique)
				for _, col := range idx.Columns() {
					plogger.Debugf("  Column[%s]", col)
				}
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
	return "user"
}
