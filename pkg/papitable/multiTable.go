package papitable

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

type MultiTableDoc struct {
	SpaceId     string `json:"spaceId"`     // 空间ID
	DatasheetId string `json:"datasheetId"` // 数据表ID
	ViewId      string `json:"viewId"`      // 视图ID
}

func NewMultiTableDoc(spaceId, datasheetId, viewId string) *MultiTableDoc {
	return &MultiTableDoc{
		SpaceId:     spaceId,
		DatasheetId: datasheetId,
		ViewId:      viewId,
	}
}

// APITable 首列不能隐藏/删除/拖拽等等操作，所以需要指定业务的主键列名
func CreateMultiTable(spaceId, tblName string, keyCol *AddField) (doc *MultiTableDoc, err error) {
	url := fmt.Sprintf("%s/fusion/v1/spaces/%s/datasheets", g_baseUrl, spaceId)

	// 构建请求体
	reqBody := createDatasheetRequest{
		Name:        tblName,
		Description: "",
		Fields:      []*AddField{keyCol},
	}

	req, err := putil.NewHttpRequestJson(http.MethodPost, url,
		getTokenHeader(),
		nil, reqBody)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	resp, err := putil.HttpDo(req)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	var respData createDatasheetResponse
	err = json.Unmarshal(resp, &respData)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	// 检查响应错误
	if !respData.Success {
		return nil, plogger.LogErr(fmt.Errorf("create datasheet failed: code=%d, message=%s", respData.Code, respData.Message))
	}

	plogger.Debug("CreateMultiTable success, datasheet_id:", respData.Data.Id)
	return NewMultiTableDoc(spaceId, respData.Data.Id, ""), nil
}

// 创建数据表请求结构
type createDatasheetRequest struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	FolderId    string      `json:"folderId,omitempty"`
	PreNodeId   string      `json:"preNodeId,omitempty"`
	Fields      []*AddField `json:"fields,omitempty"`
}

// 创建数据表响应结构
type createDatasheetResponse struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Id        string `json:"id"`
		CreatedAt int64  `json:"createdAt"`
		Fields    []struct {
			Id   string `json:"id"`
			Name string `json:"name"`
		} `json:"fields"`
	} `json:"data"`
}
