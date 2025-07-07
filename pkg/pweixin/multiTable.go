package pweixin

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

type MultiTableDoc struct {
	Docid   string `json:"docid"`    // 文档ID
	Docurl  string `json:"docurl"`   // 文档访问地址
	SheetId string `json:"sheet_id"` // 表ID
}

func (doc *MultiTableDoc) SetCurSheetId(sheetId string) {
	doc.SheetId = sheetId
}

func NewMultiTableDoc(docid, docurl string) *MultiTableDoc {
	return &MultiTableDoc{
		Docid:  docid,
		Docurl: docurl,
	}
}

// 初步调用返回48002:api forbidden，需要开启一些权限
// https://developer.work.weixin.qq.com/devtool/query?e=48002
// 在【协作->文档】这个页面副标题【可多人实时在线协作的文档、表格和幻灯片。】后面有一个【API】的UI
// 点击后看到选项【可调用接口的应用】，进去勾上我们的应用
// 该接口创建出来，由于没有指定父级位置，所以在企微客户端根本找不到这个文档
// 1：打开resp的url，然后在自己最近访问里就能找到了
// 2：TODO看如何把spaceid和fatherid填进去，创建在指定目录下
func CreateMultiTable(tblName string) (doc *MultiTableDoc, err error) {
	url := "https://qyapi.weixin.qq.com/cgi-bin/wedoc/create_doc"

	req, err := putil.NewHttpRequestJson(http.MethodPost, url, nil,
		getTokenHeader(),
		map[string]any{
			// "spaceid":     "",
			// "fatherid":    "",
			"doc_type": 10,
			"doc_name": tblName,
			// "admin_users": []string{"USERID1", "USERID2", "USERID3"},
		})
	if err != nil {
		return nil, plogger.LogErr(err)
	}
	resp, err := putil.HttpDo(req)
	if err != nil {
		return nil, plogger.LogErr(err)
	}
	var respMap map[string]any
	err = json.Unmarshal(resp, &respMap)
	if err != nil {
		return nil, plogger.LogErr(err)
	}
	err = handleRespErrorByMap(respMap)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	// plogger.Debug("createMultiTable resp : ", respMap)
	return NewMultiTableDoc(
		putil.InterfaceToString(respMap["docid"], ""),
		putil.InterfaceToString(respMap["docurl"], ""),
	), nil
}

// 子表类型常量
type wxSheetType string

const (
	SHEET_TYPE_SMARTSHEET wxSheetType = "smartsheet"
	SHEET_TYPE_DASHBOARD  wxSheetType = "dashboard"
	SHEET_TYPE_EXTERNAL   wxSheetType = "external"
)

// 子表信息结构
type SheetInfo struct {
	SheetId   string      `json:"sheet_id"`
	Title     string      `json:"title"`
	IsVisible bool        `json:"is_visible"`
	Type      wxSheetType `json:"type"`
}

// 查询子表请求结构
type getSheetRequest struct {
	Docid            string `json:"docid"`
	SheetId          string `json:"sheet_id,omitempty"`
	NeedAllTypeSheet bool   `json:"need_all_type_sheet,omitempty"`
}

// 查询子表响应结构
type getSheetResponse struct {
	Errcode   int         `json:"errcode"`
	Errmsg    string      `json:"errmsg"`
	SheetList []SheetInfo `json:"sheet_list"`
}

// 内部方法：查询子表
func (doc *MultiTableDoc) GetSheets(sheetId string) ([]SheetInfo, error) {
	url := "https://qyapi.weixin.qq.com/cgi-bin/wedoc/smartsheet/get_sheet"

	// 构建请求体
	reqBody := getSheetRequest{
		Docid:            doc.Docid,
		SheetId:          sheetId,
		NeedAllTypeSheet: true,
	}

	req, err := putil.NewHttpRequestJson(http.MethodPost, url, nil,
		getTokenHeader(),
		reqBody)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	resp, err := putil.HttpDo(req)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	var respData getSheetResponse
	err = json.Unmarshal(resp, &respData)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	// 检查响应错误
	if respData.Errcode != 0 {
		return nil, plogger.LogErr(fmt.Errorf("get sheet failed: errcode=%d, errmsg=%s", respData.Errcode, respData.Errmsg))
	}

	return respData.SheetList, nil
}
