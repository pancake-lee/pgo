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

	var docid, docurl string
	if false {
		docid, docurl, err = CreateMultiTable("TestTable_" + putil.TimeToStrDefault(time.Now()))
		if err != nil {
			t.Fatal(err)
		}
		plogger.Debug("docid : ", docid, " docurl : ", docurl)

	} else {
		// 初期调试，直接记录一个文档id在配置文件，后续还要存储起来，企微不支持查询现存文档的id
		docid = pconfig.GetStringM("WX.docid")
		docurl = pconfig.GetStringM("WX.docurl")
	}

	// curd表格数据

	// curd字段
	// https://developer.work.weixin.qq.com/document/path/99904

	// curd行，并且测试1k/10k/100k行的性能
	// https://developer.work.weixin.qq.com/document/path/99907

	// 最后再来研究curd子表/视图这类接口

}
