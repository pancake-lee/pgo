package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/pancake-lee/pgo/pkg/plogger"
)

// 生成 curd 的 proto 定义，包括数据结构和接口定义
const pbTplPath = "proto/AbandonCode.proto"
const pbOutputPath = "./proto/"

func genProto(
	tblToSvrMap map[string]*Table,
	tplTable *Table,
) {
	//以service为单位生成proto文件，一个service包含多个表的curd接口
	svcNameToTblMap := make(map[string][]*Table)
	for _, tbl := range tblToSvrMap {
		svcNameToTblMap[tbl.ServiceName] = append(svcNameToTblMap[tbl.ServiceName], tbl)
	}

	pbTplBytes, err := os.ReadFile(pbTplPath)
	if err != nil {
		plogger.Fatalf("read tpl file failed, err: %v", err)
	}
	pbTpl := string(pbTplBytes)

	// 获取一个表的接口定义内容，接下来生成多个表的代码并拼接
	apiTpl := markPairTool.GetContent(pbTpl, "MARK REPEAT API")
	// 获取一个表的数据定义内容，接下来生成多个表的代码并拼接
	msgTpl := markPairTool.GetContent(pbTpl, "MARK REPEAT MSG")

	// 每个模块写一个文件
	for svcName, tblList := range svcNameToTblMap {
		genProtoForOneService(svcName, tblList, pbTpl, apiTpl, msgTpl, tplTable)
	}
}
func genProtoForOneService(svcName string, tblList []*Table,
	pbTpl, apiTpl, msgTpl string,
	tplTable *Table,
) {
	apiCodeForAllTable := ""
	msgCodeForAllTable := ""

	sort.Slice(tblList, func(i, j int) bool {
		return tblList[i].Model.TableName() < tblList[j].Model.TableName()
	})
	for _, tbl := range tblList {
		plogger.Debugf("gen pb code for tbl[%v]", tbl.Model.TableName())

		// --------------------------------------------------
		// 处理接口定义代码
		apiCode := pbReplace(apiTpl, tplTable, tbl)
		apiCodeForAllTable += apiCode

		// --------------------------------------------------
		// 处理数据定义代码
		msgCode := pbReplace(msgTpl, tplTable, tbl)

		// 构造pb结构体的字段列表
		pbColList := ""
		for i, field := range tbl.FieldList {
			pbTypeName := ""
			// TODO 示例所有数据库类型，并且以替换的形式处理，而不是“生成固定代码”
			switch field.Type.String() {
			case "time.Time":
				pbTypeName = "int64"
			default:
				pbTypeName = field.Type.String()
			}

			pbColList += fmt.Sprintf("    %v %v = %v;\n",
				pbTypeName, DO2DTO_FieldName(field.Name), i+1)
		}

		msgCode = markPairTool.ReplaceAll("MARK REPLACE PB COL",
			msgCode, pbColList)

		//TODO 扩展多个索引的情况
		pbKeyColList := ""
		if tbl.IdxColName != "" {
			pbKeyColList = fmt.Sprintf("    repeated %v %vList = 1;\n",
				tbl.IdxColType, DO2DTO_FieldName(tbl.IdxColName))
		}
		msgCode = markPairTool.ReplaceAll("MARK REPLACE REQUEST IDX",
			msgCode, pbKeyColList)

		msgCodeForAllTable += msgCode
	}

	pbCodeStr := `// Code generated by tools/genCURD. DO NOT EDIT.` + "\n\n"
	pbCodeStr += pbTpl //先写入模板原本的内容

	// 替换service定义名称
	pbCodeStr = strings.ReplaceAll(pbCodeStr, tplTable.ServiceName, svcName)
	// 替换整份接口定义代码
	pbCodeStr = markPairTool.ReplaceAll("MARK REPEAT API", pbCodeStr, apiCodeForAllTable)
	// 替换整份数据定义代码
	pbCodeStr = markPairTool.ReplaceAll("MARK REPEAT MSG", pbCodeStr, msgCodeForAllTable)

	os.MkdirAll(pbOutputPath, 0755)
	err := os.WriteFile(pbOutputPath+"z_"+svcName+"Service.gen.proto", []byte(pbCodeStr), 0644)
	if err != nil {
		log.Fatalf("write pb code failed, err: %v", err)
	}
}

func pbReplace(
	tplCode string, tplTable *Table, tbl *Table,
) string {

	codeStr := tplCode

	// 如果没有主键，则删除相关代码块
	if tbl.IdxColName == "" {
		codeStr = markPairTool.ReplaceAll("MARK REMOVE IF NO PRIMARY KEY", codeStr, "")
	} else {
		codeStr = markPairTool.RemoveMarkSelf("MARK REMOVE IF NO PRIMARY KEY", codeStr)
		codeStr = tblIdxReplace(codeStr, tplTable, tbl)
	}

	codeStr = tblNameReplace(codeStr, tplTable, tbl)

	return codeStr
}
