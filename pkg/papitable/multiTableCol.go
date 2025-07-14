package papitable

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

// 设置多维表格文档的列列表：
// 如果该列已存在，则不修改。如需修改列，为了确保程序不会误删数据，应该手动删除列，再通过该方法重建。
// 如果该列不存在，则添加列。
// 如果deleteExcept为true，则删除除myColList之外的所有列。
func (doc *MultiTableDoc) SetColList(myColList []*AddField, deleteExcept bool) (isEdited bool, err error) {
	if len(myColList) == 0 {
		return isEdited, errors.New("cannot set empty col list")
	}

	myColNameList := make([]string, 0, len(myColList))
	nameToMyColMap := make(map[string]*AddField, len(myColList))
	for _, col := range myColList {
		myColNameList = append(myColNameList, col.Name)
		nameToMyColMap[col.Name] = col
	}

	// --------------------------------------------------
	colList, err := doc.GetCols()
	if err != nil {
		return isEdited, plogger.LogErr(err)
	}

	// plogger.Debugf("current col count[%d] : %+v", len(colList), colList)
	// os.Exit(0)

	colNameList := make([]string, 0, len(colList))
	idToColMap := make(map[string]*Field, len(colList))
	nameToColMap := make(map[string]*Field, len(colList))
	for _, col := range colList {
		colNameList = append(colNameList, col.Name)
		idToColMap[col.Id] = col
		nameToColMap[col.Name] = col
	}

	// --------------------------------------------------
	addColNameList := putil.StrListExcept(myColNameList, colNameList)
	if len(addColNameList) > 0 {
		addColList := make([]*AddField, 0, len(addColNameList))
		for _, col := range addColNameList {
			addColList = append(addColList, nameToMyColMap[col])
		}
		newFields, err := doc.AddCol(addColList)
		if err != nil {
			return isEdited, plogger.LogErr(err)
		}
		isEdited = true
		for _, newField := range newFields {
			plogger.Debugf("added col[%s] name[%s] type[%s]",
				newField.Id, newField.Name, newField.Type)
		}
	}
	// --------------------------------------------------
	if deleteExcept {
		delColNameList := putil.StrListExcept(colNameList, myColNameList)
		if len(delColNameList) > 0 {
			delColIdList := make([]string, 0, len(delColNameList))
			for _, colName := range delColNameList {
				delColIdList = append(delColIdList, nameToColMap[colName].Id)
			}
			err = doc.DelCol(delColIdList)
			if err != nil {
				return isEdited, plogger.LogErr(err)
			}
			isEdited = true
			for _, colId := range delColIdList {
				plogger.Debugf("deleted col[%s] name[%s]", idToColMap[colId].Id, idToColMap[colId].Name)
			}
		}
	}
	return isEdited, nil
}

// 删除字段
func (doc *MultiTableDoc) DelCol(fieldIds []string) error {
	if len(fieldIds) == 0 {
		return nil
	}

	for _, fieldId := range fieldIds {
		url := fmt.Sprintf("%s/fusion/v1/spaces/%s/datasheets/%s/fields/%s",
			g_baseUrl, doc.SpaceId, doc.DatasheetId, fieldId)

		req, err := putil.NewHttpRequestJson(http.MethodDelete, url,
			getTokenHeader(), nil, nil)
		if err != nil {
			return plogger.LogErr(err)
		}

		resp, err := putil.HttpDo(req)
		if err != nil {
			return plogger.LogErr(err)
		}

		var respData deleteFieldResponse
		err = json.Unmarshal(resp, &respData)
		if err != nil {
			return plogger.LogErr(err)
		}

		// 检查响应错误
		if !respData.Success {
			return plogger.LogErr(fmt.Errorf("delete field failed: code=%d, message=%s", respData.Code, respData.Message))
		}
	}

	plogger.Debug("DelCol success, deleted field count:", len(fieldIds))
	return nil
}

// --------------------------------------------------
// 内部方法：查询字段
func (doc *MultiTableDoc) GetCols() ([]*Field, error) {
	url := fmt.Sprintf("%s/fusion/v1/datasheets/%s/fields", g_baseUrl, doc.DatasheetId)

	req, err := putil.NewHttpRequestJson(http.MethodGet, url,
		getTokenHeader(), map[string]string{
			"viewId": doc.ViewId,
		}, nil)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	resp, err := putil.HttpDo(req)
	if err != nil {
		return nil, plogger.LogErr(err)
	}
	plogger.Debugf("response: %s", string(resp))

	var respData getFieldResponse
	err = json.Unmarshal(resp, &respData)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	// 检查响应错误
	if !respData.Success {
		return nil, plogger.LogErr(fmt.Errorf("get fields failed: code=%d, message=%s", respData.Code, respData.Message))
	}

	plogger.Debug("GetCol success, total fields:", len(respData.Data.Fields))
	return respData.Data.Fields, nil
}

// --------------------------------------------------
// 添加指定类型的字段
func (doc *MultiTableDoc) AddCol(fields []*AddField) (ret []*Field, err error) {
	if len(fields) == 0 {
		return []*Field{}, nil
	}

	var results []*Field
	for _, field := range fields {
		url := fmt.Sprintf("%s/fusion/v1/spaces/%s/datasheets/%s/fields",
			g_baseUrl, doc.SpaceId, doc.DatasheetId)

		req, err := putil.NewHttpRequestJson(http.MethodPost, url,
			getTokenHeader(), nil, field)
		if err != nil {
			return nil, plogger.LogErr(err)
		}

		resp, err := putil.HttpDo(req)
		if err != nil {
			return nil, plogger.LogErr(err)
		}

		var respData addFieldResponse
		err = json.Unmarshal(resp, &respData)
		if err != nil {
			return nil, plogger.LogErr(err)
		}
		plogger.Debugf("response: %s", string(resp))
		// 检查响应错误
		if !respData.Success {
			return nil, plogger.LogErr(fmt.Errorf("add field failed: code=%d, message=%s", respData.Code, respData.Message))
		}

		results = append(results, &Field{
			Id:   respData.Data.Id,
			Name: respData.Data.Name,
			Type: field.Type,
		})

		plogger.Debug("AddCol success, field_id:", respData.Data.Id)
	}

	return results, nil
}

// --------------------------------------------------
// api req/resp结构
// --------------------------------------------------

type Field struct {
	Id       string         `json:"id"`
	Name     string         `json:"name"`
	Type     FieldType      `json:"type"`
	Desc     string         `json:"desc,omitempty"`
	Property map[string]any `json:"property"`
}

type AddField struct {
	Type     FieldType      `json:"type"`
	Name     string         `json:"name"`
	Property map[string]any `json:"property"`
}

// 查询字段响应结构
type getFieldResponse struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Fields []*Field `json:"fields"`
	} `json:"data"`
}

// 添加字段响应结构
type addFieldResponse struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"data"`
}

// 删除字段响应结构
type deleteFieldResponse struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	// Data    struct{} `json:"data"` // 可解析，未解析
}

// --------------------------------------------------
// func NewSimpleTextCol(colName string) *AddField {
// 	return &AddField{
// 		Name: colName,
// 		Type: FIELD_TYPE_SINGLE_TEXT,
// 		Property: map[string]interface{}{
// 			"defaultValue": "",
// 		},
// 	}
// }

func NewTextCol(colName string) *AddField {
	return &AddField{
		Name: colName,
		Type: FIELD_TYPE_TEXT,
		Property: map[string]interface{}{
			"defaultValue": "",
		},
	}
}

// --------------------------------------------------
func NewSimpleNumCol(colName string) *AddField {
	return &AddField{
		Name: colName,
		Type: FIELD_TYPE_NUMBER,
		Property: map[string]interface{}{
			"precision": 0,
			// "commaStyle":   "",
			"defaultValue": "",
		},
	}
}

// --------------------------------------------------
func NewSimpleTimeCol(colName string) *AddField {
	return &AddField{
		Name: colName,
		Type: FIELD_TYPE_DATE_TIME,
		Property: map[string]interface{}{
			"dateFormat":  "YYYY-MM-DD",
			"includeTime": false,
			"autoFill":    false,
		},
	}
}

// --------------------------------------------------
func NewSimpleUserCol(colName string, isMultiple bool) *AddField {
	return &AddField{
		Name: colName,
		Type: FIELD_TYPE_MEMBER,
		Property: map[string]interface{}{
			"isMulti":       isMultiple,
			"shouldSendMsg": false,
		},
	}
}

// --------------------------------------------------
type SelectFieldOption struct {
	Id string `json:"id,omitempty"`
	// Name  string                  `json:"name"`
	Text string `json:"name"`
	// Color *SelectFieldOptionColor `json:"color,omitempty"` // 文档有问题，这里只需要string
	Style SelectFieldOptionColor `json:"color,omitempty"`
}

func NewSimpleSingleSelectCol(colName string, options []*SelectFieldOption) *AddField {
	return &AddField{
		Name: colName,
		Type: FIELD_TYPE_SINGLE_SELECT,
		Property: map[string]interface{}{
			"options": options,
		},
	}
}
