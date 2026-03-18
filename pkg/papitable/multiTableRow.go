package papitable

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

const (
	CELL_VALUE_KEY_TYPE_FIELD_TITLE = "name" // apitable 使用 name 作为字段键
	CELL_VALUE_KEY_TYPE_FIELD_ID    = "id"   // apitable 使用 id 作为字段键
)

// --------------------------------------------------
// 清空记录
func (doc *MultiTableDoc) DelAllRows() error {
	var rowIds []string
	pageNum := 1
	pageSize := 100 // apitable 建议的分页大小=100

	for {
		resp, err := doc.GetRow(&GetRecordRequest{
			PageNum:  pageNum,
			PageSize: pageSize,
		})
		if err != nil {
			return plogger.LogErr(err)
		}

		hasMore := pageNum*pageSize < resp.Data.Total

		plogger.Debugf("get rows pages[%v][%v] cnt [%d] hasMore[%v] total [%v]",
			pageNum, pageSize, resp.Data.PageSize, hasMore, resp.Data.Total)

		for _, row := range resp.Data.Records {
			rowIds = append(rowIds, row.RecordId)
		}

		if !hasMore {
			break
		}
		pageNum++
		time.Sleep(10 * time.Millisecond) // 避免请求过快
	}
	plogger.Debugf("total rows cnt [%d] ", len(rowIds))

	// 批量删除，每次最多10条
	err := putil.WalkSliceByStep(rowIds, 10, func(start, end int) error {
		tmpRowIds := rowIds[start:end]
		err := doc.DelRow(tmpRowIds)
		if err != nil {
			return err
		}
		time.Sleep(25 * time.Millisecond) // 避免请求过快
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// --------------------------------------------------
// 删除记录
func (doc *MultiTableDoc) DelRow(recordIds []string) error {
	if len(recordIds) == 0 {
		return nil
	}
	if len(recordIds) > 10 {
		return fmt.Errorf("recordIds length is %d, should be less than 10", len(recordIds))
	}

	// apitable 支持批量删除，但需要通过查询参数传递recordIds
	url := fmt.Sprintf("%s/fusion/v1/datasheets/%s/records", g_baseUrl, doc.DatasheetId)

	// 构建查询参数
	params := make(map[string]string)
	params["recordIds"] = putil.StrListToStr(recordIds, ",")

	req, err := putil.NewHttpRequestJson(http.MethodDelete, url,
		getTokenHeader(), params, nil)
	if err != nil {
		return plogger.LogErr(err)
	}

	resp, err := safeHttpDo(req)
	if err != nil {
		return plogger.LogErr(err)
	}

	var respData deleteRecordResponse
	err = json.Unmarshal(resp, &respData)
	if err != nil {
		return plogger.LogErr(err)
	}

	// 检查响应错误
	if !respData.Success {
		return plogger.LogErr(fmt.Errorf("delete records failed: code=%d, message=%s", respData.Code, respData.Message))
	}

	return nil
}

// --------------------------------------------------
// 创建记录
func (doc *MultiTableDoc) GetRow(req *GetRecordRequest) (*getRecordResponse, error) {
	url := fmt.Sprintf("%s/fusion/v1/datasheets/%s/records", g_baseUrl, doc.DatasheetId)

	req.FieldKey = CELL_VALUE_KEY_TYPE_FIELD_TITLE

	query := putil.GetUrlQueryString(req)
	httpReq, err := putil.NewHttpRequestJson(http.MethodGet, url,
		getTokenHeader(), query, nil)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	resp, err := safeHttpDo(httpReq)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	var respData getRecordResponse
	err = json.Unmarshal(resp, &respData)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	// 检查响应错误
	if !respData.Success {
		return nil, plogger.LogErr(fmt.Errorf("get records failed: code=%d, message=%s", respData.Code, respData.Message))
	}

	return &respData, nil
}
func NewSortRule(fieldName string, desc bool) SortRule {
	ret := SortRule{
		Field: fieldName,
		Order: "asc",
	}
	if desc {
		ret.Order = "desc"
	}
	return ret
}

// --------------------------------------------------
// 增加记录
func (doc *MultiTableDoc) AddRow(rows []*AddRecord) (records []*CommonRecord, err error) {
	err = putil.WalkSliceByStep(rows, 10, func(start, end int) error {
		tmpRows := rows[start:end]
		r, err := doc.iAddRow(tmpRows)
		if err != nil {
			return err
		}
		records = append(records, r...)

		return nil
	})
	return records, err
}

func (doc *MultiTableDoc) iAddRow(rows []*AddRecord) ([]*CommonRecord, error) {
	if len(rows) == 0 {
		return nil, nil
	}
	if len(rows) > 10 {
		return nil, fmt.Errorf("records length is %d, should be less than 10", len(rows))
	}

	url := fmt.Sprintf("%s/fusion/v1/datasheets/%s/records", g_baseUrl, doc.DatasheetId)

	// 构建请求体
	reqBody := addRecordRequest{
		Records:  rows,
		FieldKey: CELL_VALUE_KEY_TYPE_FIELD_TITLE,
	}

	req, err := putil.NewHttpRequestJson(http.MethodPost, url,
		getTokenHeader(), nil, reqBody)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	resp, err := safeHttpDo(req)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	var respData addRecordResponse
	err = json.Unmarshal(resp, &respData)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	// 检查响应错误
	if !respData.Success {
		return nil, plogger.LogErr(fmt.Errorf("add records failed: code=%d, message=%s", respData.Code, respData.Message))
	}

	return respData.Data.Records, nil
}

// --------------------------------------------------
func (doc *MultiTableDoc) EditRow(rows []*UpdateRecord) error {
	rows, err := doc.filterEditableUpdateRows(rows)
	if err != nil {
		return plogger.LogErr(err)
	}
	if len(rows) == 0 {
		plogger.Debugf("EditRow skip, no editable field to update")
		return nil
	}

	url := fmt.Sprintf("%s/fusion/v1/datasheets/%s/records", g_baseUrl, doc.DatasheetId)

	// 构建请求体
	reqBody := UpdateRecordRequest{
		Records:  rows,
		FieldKey: CELL_VALUE_KEY_TYPE_FIELD_TITLE,
	}

	req, err := putil.NewHttpRequestJson(http.MethodPatch, url,
		getTokenHeader(), nil, reqBody)
	if err != nil {
		return plogger.LogErr(err)
	}

	resp, err := safeHttpDo(req)
	if err != nil {
		return plogger.LogErr(err)
	}

	var respData UpdateRecordResponse
	err = json.Unmarshal(resp, &respData)
	if err != nil {
		return plogger.LogErr(err)
	}

	// 检查响应错误
	if !respData.Success {
		return plogger.LogErr(fmt.Errorf("edit records failed: code=%d, message=%s", respData.Code, respData.Message))
	}

	return nil
}

func (doc *MultiTableDoc) filterEditableUpdateRows(rows []*UpdateRecord) ([]*UpdateRecord, error) {
	if len(rows) == 0 {
		return nil, nil
	}

	colList, err := doc.GetCols()
	if err != nil {
		return nil, err
	}

	nonEditableTypeMap := map[FieldType]struct{}{
		FIELD_TYPE_MAGIC_LOOKUP:       {},
		FIELD_TYPE_FORMULA:            {},
		FIELD_TYPE_AUTO_NUMBER:        {},
		FIELD_TYPE_CREATED_TIME:       {},
		FIELD_TYPE_LAST_MODIFIED_TIME: {},
		FIELD_TYPE_CREATED_BY:         {},
		FIELD_TYPE_LAST_MODIFIED_BY:   {},
		FIELD_TYPE_BUTTON:             {},
	}

	nonEditableFieldNameMap := make(map[string]FieldType)
	nonEditableFieldIDMap := make(map[string]FieldType)
	for _, col := range colList {
		if col == nil {
			continue
		}
		if _, ok := nonEditableTypeMap[col.Type]; !ok {
			continue
		}

		nonEditableFieldNameMap[col.Name] = col.Type
		nonEditableFieldIDMap[col.Id] = col.Type
	}

	ret := make([]*UpdateRecord, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}

		editableFields := make(map[string]any, len(row.Fields))
		for k, v := range row.Fields {
			if fieldType, ok := nonEditableFieldNameMap[k]; ok {
				plogger.Debugf("EditRow skip readonly field[%s], type[%s], recordId[%s]",
					k, fieldType, row.RecordId)
				continue
			}
			if fieldType, ok := nonEditableFieldIDMap[k]; ok {
				plogger.Debugf("EditRow skip readonly field[%s], type[%s], recordId[%s]",
					k, fieldType, row.RecordId)
				continue
			}

			editableFields[k] = v
		}

		if len(editableFields) == 0 {
			plogger.Debugf("EditRow skip recordId[%s], all fields readonly", row.RecordId)
			continue
		}

		ret = append(ret, &UpdateRecord{
			RecordId: row.RecordId,
			Fields:   editableFields,
		})
	}

	return ret, nil
}

// --------------------------------------------------
// api req/resp结构
// --------------------------------------------------
type UpdateRecord struct {
	RecordId string         `json:"recordId"`
	Fields   map[string]any `json:"fields"`
}

type UpdateRecordRequest struct {
	Records  []*UpdateRecord `json:"records"`
	FieldKey string          `json:"fieldKey,omitempty"` // "name" or "id", default "name"
}

type UpdateRecordResponse struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Records []CommonRecord `json:"records"`
	} `json:"data"`
}

// --------------------------------------------------
// 添加记录结构
type AddRecord struct {
	Values map[string]any `json:"fields"`
	// Fields map[string]any `json:"fields"`
}

// 添加记录请求结构
type addRecordRequest struct {
	Records  []*AddRecord `json:"records"`
	FieldKey string       `json:"fieldKey,omitempty"`
}

// 添加记录响应结构
type addRecordResponse struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Records []*CommonRecord `json:"records"`
	} `json:"data"`
}

// --------------------------------------------------
type CommonRecord struct {
	RecordId  string         `json:"recordId"`
	Fields    map[string]any `json:"fields"`
	CreatedAt int64          `json:"createdAt"`
	UpdatedAt int64          `json:"updatedAt"`
}

// 查询记录请求结构
type GetRecordRequest struct {
	PageSize        int        `json:"pageSize,omitempty"`
	MaxRecords      int        `json:"maxRecords,omitempty"`
	PageNum         int        `json:"pageNum,omitempty"`
	Sort            []SortRule `json:"sort,omitempty"`
	RecordIds       []string   `json:"recordIds,omitempty"`
	ViewId          string     `json:"viewId,omitempty"`
	Fields          []string   `json:"fields,omitempty"`
	FilterByFormula string     `json:"filterByFormula,omitempty"`
	CellFormat      string     `json:"cellFormat,omitempty"`
	FieldKey        string     `json:"fieldKey,omitempty"`
}

// 查询记录响应结构
type getRecordResponse struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		PageNum  int             `json:"pageNum"`
		Records  []*CommonRecord `json:"records"`
		PageSize int             `json:"pageSize"`
		Total    int             `json:"total"`
	} `json:"data"`
}

// --------------------------------------------------
// 删除记录响应结构
type deleteRecordResponse struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	// Data    struct{} `json:"data"` // 可解析，未解析
}

// --------------------------------------------------
// 排序规则
type SortRule struct {
	Field string `json:"field"`
	Order string `json:"order"` // "asc" 或 "desc"
}
