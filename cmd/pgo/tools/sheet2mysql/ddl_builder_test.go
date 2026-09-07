package sheet2mysql

import (
	"strings"
	"testing"

	"github.com/pancake-lee/pgo/pkg/papitable"
)

func TestBuildMysqlCreateTableSQL(t *testing.T) {
	fieldList := []*papitable.Field{
		{Name: "标题", Type: papitable.FIELD_TYPE_TEXT},
		{Name: "完成", Type: papitable.FIELD_TYPE_CHECKBOX},
	}
	sql, warningList, err := BuildMysqlCreateTableSQL("测试任务", fieldList, BuildDDLOptions{DatasheetID: "dst1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(warningList) != 0 || !strings.Contains(sql, "CREATE TABLE") || !strings.Contains(sql, "标题") {
		t.Fatalf("unexpected DDL: warnings=%v sql=%s", warningList, sql)
	}
}
