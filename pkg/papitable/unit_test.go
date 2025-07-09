package papitable

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

func TestAPITable(t *testing.T) {
	plogger.InitConsoleLogger()

	pconfig.MustInitConfig(filepath.Join(putil.GetCurDir(), "../../configs/pancake.yaml"))

	err := InitAPITableByConfig()
	if err != nil {
		t.Fatal(err)
	}

	myColList := []*AddField{
		NewTextCol("DB_ID"),
		NewTextCol("测试字段1"),
		NewTextCol("测试字段2"),
	}
	// --------------------------------------------------
	var doc *MultiTableDoc
	if false {
		doc, err = CreateMultiTable(
			pconfig.GetStringM("APITable.spaceId"),
			"TestTable_"+putil.TimeToStrDefault(time.Now()),
			myColList[0].Name,
		)
		if err != nil {
			t.Fatal(err)
		}
		plogger.Debugf("doc space[%v] datasheet[%v]", doc.SpaceId, doc.DatasheetId)

	} else {
		// 初期调试，直接记录一个文档id在配置文件，后续还要存储起来，企微不支持查询现存文档的id
		doc = NewMultiTableDoc(
			pconfig.GetStringM("APITable.spaceId"),
			pconfig.GetStringM("APITable.datasheetId"),
			pconfig.GetStringM("APITable.viewId"),
		)
	}

	// --------------------------------------------------
	// curd表格数据
	// --------------------------------------------------
	// 先删除所有行
	// err = doc.DelAllRows()
	// if err != nil {
	// 	t.Fatalf("failed to delete all rows: %v", err)
	// }

	// --------------------------------------------------

	isEdited, err := doc.SetColList(myColList, false)
	if err != nil {
		t.Fatal(err)
	}

	if isEdited { // 修改字段后，重新导入数据，经常会导致频繁请求
		plogger.Debugf("columns updated, now we can add rows")
	}

	// --------------------------------------------------
	getRandomRecord := func() map[string]any {
		rowValues := make(map[string]any)
		for _, col := range myColList {
			if col.Type == FIELD_TYPE_TEXT {
				rowValues[col.Name] = NewTextValue("测试值_" + putil.GetRandStr(4))
			}
		}
		return rowValues
	}

	// 当前这里用go test运行有点问题，本来打算30*1000写入3w数据的，微信官方文档明确单表限制4w行
	// 首先企微API并发不能太快，一次1000行，30s间隔也会报错，1min顺利
	// https://developer.work.weixin.qq.com/document/path/90312
	// 但是go test本身有时间限制，我配置了5min，所以只能插入3k行做测试，更多性能测试后面再说
	for range 1 {
		// for range 400 {
		var rowList []*AddRecord
		for range 10 {
			rowList = append(rowList, &AddRecord{Fields: getRandomRecord()})
		}
		err = doc.AddRow(rowList)
		if err != nil {
			plogger.LogErr(err)
		} else {
			plogger.Debugf("added %d rows", len(rowList))
		}
		time.Sleep(500 * time.Millisecond) // 避免请求过快
	}
}
