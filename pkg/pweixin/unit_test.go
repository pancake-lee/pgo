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

	err = GetUserList()
	if err != nil {
		t.Fatal(err)
	}
	if true {
		return
	}

	// 成功了的
	// sendTo := pconfig.GetStringM("WX.sendTo")
	// err = sendMsg([]string{sendTo}, "我就试一下")
	// if err != nil {
	// t.Fatal(err)
	// }

	// --------------------------------------------------
	var doc *MultiTableDoc
	if false {
		doc, err = CreateMultiTable("TestTable_" + putil.TimeToStrDefault(time.Now()))
		if err != nil {
			t.Fatal(err)
		}
		plogger.Debug("docid : ", doc.Docid, " docurl : ", doc.Docurl)

	} else {
		// 初期调试，直接记录一个文档id在配置文件，后续还要存储起来，企微不支持查询现存文档的id
		doc = NewMultiTableDoc(
			pconfig.GetStringM("WX.docid"),
			pconfig.GetStringM("WX.docurl"),
		)
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
	// 先删除所有行
	err = doc.DelAllRows()
	if err != nil {
		t.Fatalf("failed to delete all rows: %v", err)
	}

	// --------------------------------------------------
	myColList := []*AddField{
		NewTextCol("测试字段1"),
		NewTextCol("测试字段2"),
	}

	isEdited, err := doc.SetColList(myColList, false)
	if err != nil {
		t.Fatal(err)
	}

	if isEdited { // 修改字段后，重新导入数据，经常会导致频繁请求
		for i := range 12 {
			time.Sleep(5 * time.Second)
			plogger.Debugf("wait [%v/%v]s for wx api to cooldown", (i+1)*5, 60)
		}
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
	// https://developer.work.weixin.qq.com/document/path/90312
	// 但是go test本身有时间限制，我配置了5min，所以只能插入3k行做测试，更多性能测试后面再说
	for range 4 {
		var rowList []*AddRecord
		for range 1000 {
			rowList = append(rowList, &AddRecord{Values: getRandomRecord()})
		}
		err = doc.AddRow(rowList)
		if err != nil {
			plogger.LogErr(err)
		} else {
			plogger.Debugf("added %d rows", len(rowList))
		}
		time.Sleep(60 * time.Second) // 避免请求过快
	}

}
