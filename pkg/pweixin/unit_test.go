package pweixin

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

func TestWX(t *testing.T) {
	plogger.InitConsoleLogger()

	pconfig.MustInitConfig(filepath.Join(putil.GetCurDir(), "../../configs/pancake.yaml"))

	err := InitWxApiByConfig()
	if err != nil {
		t.Fatal(err)
	}

	// 权限还没搞好
	// err = getUserList()
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// 成功了的
	// sendTo := pconfig.GetStringM("WX.sendTo")
	// err = sendMsg([]string{sendTo}, "我就试一下")
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// --------------------------------------------------
	var doc *multiTableDoc
	if false {
		doc, err = CreateMultiTable("TestTable_" + putil.TimeToStrDefault(time.Now()))
		if err != nil {
			t.Fatal(err)
		}
		plogger.Debug("docid : ", doc.Docid, " docurl : ", doc.Docurl)

	} else {
		// 初期调试，直接记录一个文档id在配置文件，后续还要存储起来，企微不支持查询现存文档的id
		doc = &multiTableDoc{
			Docid:  pconfig.GetStringM("WX.docid"),
			Docurl: pconfig.GetStringM("WX.docurl"),
		}
	}

	// --------------------------------------------------
	// 获取智能文档的第一个智能表格
	sheets, err := doc.GetSheets("")
	if err != nil {
		t.Fatal(err)
	}
	if len(sheets) == 0 {
		t.Fatal("no sheets found in the document")
	}
	for _, sheet := range sheets {
		if sheet.Type != SHEET_TYPE_SMARTSHEET {
			continue
		}
		plogger.Debugf("found smart sheet[%v] [%s]", sheet.SheetId, sheet.Title)
		if doc.SheetId == "" {
			doc.SetCurSheetId(sheet.SheetId)
		}
	}

	if doc.SheetId == "" {
		t.Fatal("no smart sheet found in the document")
	}

	// --------------------------------------------------
	// curd表格数据
	// --------------------------------------------------
	myColList := []AddField{
		{
			FieldTitle: "测试字段1",
			FieldType:  FIELD_TYPE_TEXT,
		},
		{
			FieldTitle: "测试字段2",
			FieldType:  FIELD_TYPE_TEXT,
		},
	}

	myColNameList := make([]string, 0, len(myColList))
	myColMap := make(map[string]AddField, len(myColList))
	for _, col := range myColList {
		myColNameList = append(myColNameList, col.FieldTitle)
		myColMap[col.FieldTitle] = col
	}

	// --------------------------------------------------
	colList, err := doc.GetCols(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	colNameList := make([]string, 0, len(colList))
	colMap := make(map[string]Field, len(colList))
	for _, col := range colList {
		colNameList = append(colNameList, col.FieldTitle)
		colMap[col.FieldTitle] = col
	}

	// --------------------------------------------------
	addColNameList := putil.StrListExcept(myColNameList, colNameList)
	addColList := make([]AddField, 0, len(addColNameList))
	for _, col := range addColNameList {
		addColList = append(addColList, myColMap[col])
	}
	newFields, err := doc.AddCol(addColList)
	if err != nil {
		t.Fatal(err)
	}
	for _, newField := range newFields {
		plogger.Debugf("added col[%s] title[%s] type[%s]",
			newField.FieldId, newField.FieldTitle, newField.FieldType)
	}

	// --------------------------------------------------
	delColNameList := putil.StrListExcept(colNameList, myColNameList)
	delColList := make([]string, 0, len(delColNameList))
	for _, col := range delColNameList {
		delColList = append(delColList, colMap[col].FieldId)
	}
	err = doc.DelCol(delColList)
	if err != nil {
		t.Fatal(err)
	}
	for _, col := range delColList {
		plogger.Debugf("deleted col[%s] title[%s]", colMap[col].FieldId, colMap[col].FieldTitle)
	}

	// --------------------------------------------------
	// 先删除所有行
	var rowIds []string
	limit := 1000
	offset := 0
	for {
		resp, err := doc.GetRow(&getRecordRequest{
			Docid:   doc.Docid,
			SheetId: doc.SheetId,
			Offset:  uint32(offset),
			Limit:   uint32(limit),
		})
		if err != nil {
			t.Fatal(err)
		}
		offset = int(resp.Next)
		plogger.Debugf("get rows cnt [%d] next[%v] total [%v]", len(resp.Records), resp.Next, resp.Total)

		for _, row := range resp.Records {
			rowIds = append(rowIds, row.RecordId)
		}

		if !resp.HasMore {
			break
		}
	}
	plogger.Debugf("total rows cnt [%d] ", offset)

	err = doc.DelRow(rowIds)
	if err != nil {
		t.Fatal(err)
	}

	// --------------------------------------------------
	getRandomRecord := func() map[string]any {
		rowValues := make(map[string]any)
		for _, col := range myColList {
			if col.FieldType == FIELD_TYPE_TEXT {
				rowValues[col.FieldTitle] = NewTextValue("测试值_" + putil.GetRandStr(4))
			}
		}
		return rowValues
	}

	// 当前这里用go test运行有点问题，本来打算30*1000写入3w数据的，微信官方文档明确单表限制4w行
	// 首先企微API并发不能太快，一次1000行，30s间隔也会报错，1min顺利
	// 但是go test本身有时间限制，我配置了5min，所以只能插入3k行做测试，更多性能测试后面再说

	for range 4 {
		var rowList []AddRecord
		for range 1000 {
			rowList = append(rowList, AddRecord{Values: getRandomRecord()})
		}
		err = doc.AddRow(rowList)
		if err != nil {
			plogger.Errorf("add row failed: %v", err)
			// t.Fatal(err)
		} else {
			plogger.Debugf("added %d rows", len(rowList))
		}
		time.Sleep(60 * time.Second) // 避免请求过快
	}
}
