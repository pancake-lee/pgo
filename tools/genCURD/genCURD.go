package main

import (
	"fmt"
	"gogogo/pkg/db/dao/model"
	"gogogo/pkg/util"
	"log"
	"os/exec"
	"reflect"
	"strings"
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

var tblToSvrMap = make(map[string]*Table)

func addTable(m dbModel, svcName string) {
	tblToSvrMap[m.TableName()] = newTable(m, svcName)
}
func newTable(m dbModel, svcName string) *Table {
	tbl := Table{
		ServiceName: svcName,
		Model:       m,
	}
	tblName := tbl.Model.TableName()
	tbl.HyphenName = strings.ReplaceAll(tblName, "_", "-")
	tbl.UpperCamelName = util.StrToCamelCase(tblName)
	tbl.LowerCamelName = util.StrFirstToLower(tbl.UpperCamelName)
	return &tbl
}

func main() {
	log.SetFlags(log.Lshortfile | log.Ltime)

	addTable(&model.User{}, "user")
	addTable(&model.UserDept{}, "user")
	addTable(&model.UserDeptAssoc{}, "user")
	addTable(&model.UserJob{}, "user")

	//读取数据库表结构
	for _, tbl := range tblToSvrMap {

		isMultiKey := false
		val := reflect.ValueOf(tbl.Model).Elem()
		for i := 0; i < val.NumField(); i++ {
			field := val.Type().Field(i)
			tbl.FieldList = append(tbl.FieldList, &field)

			log.Printf("Field[%s] Type[%s] Tag[%v]", field.Name, field.Type, field.Tag)
			if strings.Contains(field.Tag.Get("gorm"), "primaryKey") {
				if tbl.IdxColName != "" {
					isMultiKey = true
				}
				tbl.IdxColName = field.Name
				tbl.IdxColType = field.Type.String()
				tbl.IdxParmName = util.StrFirstToLower(util.StrIdToLower(tbl.IdxColName))
			}
		}
		if isMultiKey {
			tbl.IdxColName = ""
			tbl.IdxColType = ""
			tbl.IdxParmName = ""
		}
		log.Println("tbl info : ", tbl)
	}

	tplTable := newTable(&model.AbandonCode{}, "abandonCode")
	tplTable.IdxColName = "Idx1"
	tplTable.IdxColType = "int32"
	tplTable.IdxParmName = "idx1"

	genDaoCode(tblToSvrMap, tplTable)
	genProto(tblToSvrMap, tplTable)

	// 调用 protoc 生成 go 代码
	cmd := exec.Command("make", "api")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("make proto failed, err: %v, out: \n%v", err, string(out))
	}

	genServiceCode(tblToSvrMap, tplTable)
}

func getLastLineEndOfMark(fileStr, mark string) int {
	i := strings.Index(fileStr, mark)
	if i == -1 {
		log.Printf("mark not found, mark: %v\n", mark)
		return -1
	}
	return strings.LastIndex(fileStr[:i], "\n") + 1
}

func getNextLineStartOfMark(fileStr, mark string) int {
	i := strings.Index(fileStr, mark)
	if i == -1 {
		log.Printf("mark not found, mark: %v\n", mark)
		return -1
	}
	return i + strings.Index(fileStr[i:], "\n") + 1
}

// 获取标记的内容
func getMarkContent(fileStr string, mark string) string {
	nextOfStart := getNextLineStartOfMark(fileStr, mark+" START")
	lastOfEnd := getLastLineEndOfMark(fileStr, mark+" END")
	if nextOfStart == -1 || lastOfEnd == -1 {
		log.Printf("mark not found, mark: %v\n", mark)
		return ""
	}
	return fileStr[nextOfStart:lastOfEnd]
}

// 替换标记的内容
func replaceMarkAll(mark, fileStr, replaceStr string) string {
	for strings.Contains(fileStr, mark) {
		fileStr = replaceMarkOnce(mark, fileStr, replaceStr)
	}
	return fileStr
}

func replaceMarkOnce(mark, fileStr, replaceStr string) string {
	iStart := getLastLineEndOfMark(fileStr, mark+" START")
	iEnd := getNextLineStartOfMark(fileStr, mark+" END")
	if iStart == -1 || iEnd == -1 {
		log.Printf("mark not found, mark: %v\n", mark)
		return fileStr
	}
	return fileStr[:iStart] + replaceStr + fileStr[iEnd:]
}

// 标记无需操作，删掉标记注释本身
func removeMarkAll(mark, fileStr string) string {
	for strings.Contains(fileStr, mark) {
		fileStr = removeMarkOnce(mark, fileStr)
	}
	return fileStr
}
func removeMarkOnce(mark, fileStr string) string {
	lastOfStart := getLastLineEndOfMark(fileStr, mark+" START")
	nextOfStart := getNextLineStartOfMark(fileStr, mark+" START")
	lastOfEnd := getLastLineEndOfMark(fileStr, mark+" END")
	nextOfEnd := getNextLineStartOfMark(fileStr, mark+" END")
	if lastOfStart == -1 || nextOfStart == -1 || lastOfEnd == -1 || nextOfEnd == -1 {
		log.Printf("mark not found, mark: %v\n", mark)
		return fileStr
	}
	return fileStr[:lastOfStart] + fileStr[nextOfStart:lastOfEnd] + fileStr[nextOfEnd:]
}

func codeReplace(
	tplCode string, tplTable *Table, tbl *Table,
) string {

	codeStr := tplCode

	if tbl.IdxColName == "" {
		codeStr = replaceMarkAll("MARK REPLACE IDX", codeStr, "")
	} else {
		codeStr = removeMarkAll("MARK REPLACE IDX", codeStr)
	}

	codeStr = strings.ReplaceAll(codeStr,
		util.StrToCamelCase(tplTable.ServiceName)+"CURDServer",
		util.StrToCamelCase(tbl.ServiceName)+"CURDServer")

	codeStr = strings.ReplaceAll(codeStr, tplTable.ServiceName+"Service", tbl.ServiceName+"Service")

	codeStr = strings.ReplaceAll(codeStr, tplTable.Model.TableName(), tbl.Model.TableName())
	codeStr = strings.ReplaceAll(codeStr, tplTable.HyphenName, tbl.HyphenName)
	codeStr = strings.ReplaceAll(codeStr, tplTable.LowerCamelName, tbl.LowerCamelName)
	codeStr = strings.ReplaceAll(codeStr, tplTable.UpperCamelName, tbl.UpperCamelName)

	codeStr = strings.ReplaceAll(codeStr, tplTable.IdxColName, util.StrIdToLower(tbl.IdxColName))
	codeStr = strings.ReplaceAll(codeStr, tplTable.IdxColType, tbl.IdxColType)
	codeStr = strings.ReplaceAll(codeStr, tplTable.IdxParmName, tbl.IdxParmName)

	return codeStr
}
