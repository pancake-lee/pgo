package papitable

import (
	"os"
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
	// --------------------------------------------------
	if true {
		// 用户列表
		userList, err := GetUserList(pconfig.GetStringM("APITable.spaceId"))
		if err != nil {
			t.Fatal(err)
		}
		plogger.Debugf("获取到 %d 个用户", len(userList))
		for _, user := range userList {
			plogger.Debugf("User: ID=%s, Name=%s, Email=%s", user.UnitId, user.Name, user.Email)
		}
		return
	}

	// --------------------------------------------------
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
			myColList[0],
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
		)
	}
	// --------------------------------------------------
	// 获取数据，简单看看接口通不通
	if true {
		resp, err := doc.GetRow(&GetRecordRequest{
			PageNum:  1,
			PageSize: 100,
		})
		if err != nil {
			plogger.LogErr(err)
			return
		}

		plogger.Debugf("get rows cnt [%d] total [%v]", resp.Data.PageSize, resp.Data.Total)

		for _, r := range resp.Data.Records {
			plogger.Debugf("row: %+v", r.Fields)
		}

		return
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

	for range 400 {
		var rowList []*AddRecord
		for range 10 {
			rowList = append(rowList, &AddRecord{Values: getRandomRecord()})
		}
		_, err = doc.AddRow(rowList)
		if err != nil {
			plogger.LogErr(err)
		} else {
			plogger.Debugf("added %d rows", len(rowList))
		}
		time.Sleep(10 * time.Millisecond) // 避免请求过快
	}
}

func TestFileCol(t *testing.T) {
	var err error
	plogger.InitConsoleLogger()

	pconfig.MustInitConfig(filepath.Join(putil.GetCurDir(), "../../configs/pancake.yaml"))

	err = InitAPITableByConfig()
	if err != nil {
		t.Fatal(err)
	}

	doc := NewMultiTableDoc(
		pconfig.GetStringM("APITable.spaceId"),
		pconfig.GetStringM("APITable.datasheetId"))

	// 确保附件列存在
	_, err = doc.SetColList([]*AddField{
		NewAttachmentCol("测试附件"),
	}, false)
	if err != nil {
		t.Fatal(err)
	}

	if true { // 关闭上传，测试/确认附件字段值的结构
		filePath := "../../bin/2.jpg"
		// 打开文件
		f, err := os.Open(filePath)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		fi, err := f.Stat()
		if err != nil {
			t.Fatal(err)
		}

		// 上传文件并获取 token
		upResp, err := doc.UploadAttachmentWithPresignedUrl(
			filepath.Base(filePath), putil.NewPath(filePath).MIME(), fi.Size(), f)
		if err != nil {
			t.Fatal(err)
		}
		plogger.Debugf("upload response: %+v", upResp.Data)

		// att := NewAttachmentValue(upResp.Data.Name, upResp.Data.Token, upResp.Data.MimeType, upResp.Data.Size)
		att := NewAttachmentValue("cover.jpg", upResp.Data.Token)

		// 添加一行，附件字段为数组形式
		row := &AddRecord{Values: map[string]any{
			"测试附件": []any{att},
		}}
		_, err = doc.AddRow([]*AddRecord{row})
		if err != nil {
			t.Fatal(err)
		}
	}

	// 读取回数据并验证
	resp, err := doc.GetRow(&GetRecordRequest{
		PageNum: 1, PageSize: 10,
		Fields: []string{"测试附件"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data.Records) == 0 {
		t.Fatalf("no records returned")
	}
	rec := resp.Data.Records[len(resp.Data.Records)-1] // 只看最后一条
	v, ok := rec.Fields["测试附件"]
	if !ok {
		t.Fatalf("attachment field missing in returned record")
	}

	// 解析附件字段并打印结构化结果
	parsed, err := ParseAttachmentValue(v)
	if err != nil {
		t.Fatalf("failed to parse attachment value: %v", err)
	}
	plogger.Debugf("parsed attachment value: %+v", parsed)
	if len(parsed) == 0 || parsed[0].Name == "" || parsed[0].Size == 0 {
		t.Fatalf("parsed attachment value seems incorrect")
	}
}
