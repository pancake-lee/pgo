package pweixin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

// --------------------------------------------------
// 删除记录

func (doc *MultiTableDoc) DelAllRows() error {
	var rowIds []string
	limit := 1000
	offset := 0
	for {
		resp, err := doc.GetRow(&GetRecordRequest{
			Docid:   doc.Docid,
			SheetId: doc.SheetId,
			Offset:  uint32(offset),
			Limit:   uint32(limit),
		})
		if err != nil {
			return plogger.LogErr(err)
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

	err := doc.DelRow(rowIds)
	if err != nil {
		return plogger.LogErr(err)
	}
	return nil
}

func (doc *MultiTableDoc) DelRow(recordIds []string) error {
	if len(recordIds) == 0 {
		return nil
	}
	url := g_baseUrl + "/cgi-bin/wedoc/smartsheet/delete_records"

	// 构建请求体
	reqBody := deleteRecordRequest{
		Docid:     doc.Docid,
		SheetId:   doc.SheetId,
		RecordIds: recordIds,
	}

	req, err := putil.NewHttpRequestJson(http.MethodPost, url, nil,
		getTokenQuerys(),
		reqBody)
	if err != nil {
		return plogger.LogErr(err)
	}

	resp, err := putil.HttpDo(req)
	if err != nil {
		return plogger.LogErr(err)
	}

	var respData deleteRecordResponse
	err = json.Unmarshal(resp, &respData)
	if err != nil {
		return plogger.LogErr(err)
	}

	// 检查响应错误
	if respData.Errcode != 0 {
		return plogger.LogErr(fmt.Errorf("delete records failed: errcode=%d, errmsg=%s", respData.Errcode, respData.Errmsg))
	}

	return nil
}

// 删除记录请求结构
type deleteRecordRequest struct {
	Docid     string   `json:"docid"`
	SheetId   string   `json:"sheet_id"`
	RecordIds []string `json:"record_ids"`
}

// 删除记录响应结构
type deleteRecordResponse struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

// --------------------------------------------------
// 创建排序规则
func NewSortRule(fieldTitle string, desc bool) SortRule {
	return SortRule{
		FieldTitle: fieldTitle,
		Desc:       desc,
	}
}

func (doc *MultiTableDoc) GetRow(req *GetRecordRequest) (*getRecordResponse, error) {
	url := g_baseUrl + "/cgi-bin/wedoc/smartsheet/get_records"

	httpReq, err := putil.NewHttpRequestJson(http.MethodPost, url, nil,
		getTokenQuerys(),
		req)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	resp, err := putil.HttpDo(httpReq)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	var respData getRecordResponse
	err = json.Unmarshal(resp, &respData)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	// 检查响应错误
	if respData.Errcode != 0 {
		return nil, plogger.LogErr(fmt.Errorf("get records failed: errcode=%d, errmsg=%s", respData.Errcode, respData.Errmsg))
	}

	return &respData, nil
}

// --------------------------------------------------
// 创建数字值
func NewNumValue(value float64) float64 {
	return value
}

// 解析数字值
func ParseNumValue(value any) (float64, error) {
	if v, ok := value.(float64); ok {
		return v, nil
	}
	return 0, fmt.Errorf("invalid number value type: %T", value)
}

// 创建文本值
func NewTextValue(text string) []CellTextValue {
	return []CellTextValue{
		{
			Type: "text",
			Text: text,
		},
	}
}

// 解析文本值
func ParseTextValue(value any) (string, error) {
	if values, ok := value.([]interface{}); ok && len(values) > 0 {
		if textValue, ok := values[0].(map[string]interface{}); ok {
			if text, ok := textValue["text"].(string); ok {
				return text, nil
			}
		}
	}
	return "", fmt.Errorf("invalid text value format")
}

// 创建日期时间值（传入毫秒时间戳字符串）
func NewTimeValue(t time.Time) any {
	return putil.Int64ToStr(t.UnixMilli())
}

// 解析日期时间值
func ParseTimeValue(value any) (time.Time, error) {
	if timeStr, ok := value.(string); ok {
		timestamp, err := putil.StrToInt64(timeStr)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid timestamp format: %v", err)
		}
		return time.UnixMilli(timestamp), nil
	}
	return time.Time{}, fmt.Errorf("invalid time value type: %T", value)
}

// 创建链接值
func NewUrlValue(text, link string) []CellTextValue {
	return []CellTextValue{
		{
			Type: "url",
			Text: text,
			Link: link,
		},
	}
}

// 解析链接值
func ParseUrlValue(value any) (text, link string, err error) {
	if values, ok := value.([]interface{}); ok && len(values) > 0 {
		if urlValue, ok := values[0].(map[string]interface{}); ok {
			text, _ = urlValue["text"].(string)
			link, _ = urlValue["link"].(string)
			if text != "" || link != "" {
				return text, link, nil
			}
		}
	}
	return "", "", fmt.Errorf("invalid url value format")
}

// 创建用户值
func NewUserValue(userIds ...string) []CellUserValue {
	values := make([]CellUserValue, len(userIds))
	for i, userId := range userIds {
		values[i] = CellUserValue{UserId: userId}
	}
	return values
}

// 解析用户值
func ParseUserValue(value any) ([]string, error) {
	if values, ok := value.([]interface{}); ok {
		userIds := make([]string, 0, len(values))
		for _, v := range values {
			if userValue, ok := v.(map[string]interface{}); ok {
				if userId, ok := userValue["user_id"].(string); ok {
					userIds = append(userIds, userId)
				}
			}
		}
		return userIds, nil
	}
	return nil, fmt.Errorf("invalid user value format")
}

// 创建选项值
func NewOptionValue(option *SelectFieldOption) *CellOption {
	return &CellOption{
		Id:    option.Id,
		Text:  option.Text,
		Style: uint32(option.Style),
	}
}

// 解析选项值
func ParseSingleOptionValue(value any) (*CellOption, error) {
	if values, ok := value.([]interface{}); ok && len(values) > 0 {
		if optionValue, ok := values[0].(map[string]interface{}); ok {
			option := &CellOption{}
			if id, ok := optionValue["id"].(string); ok {
				option.Id = id
			}
			if text, ok := optionValue["text"].(string); ok {
				option.Text = text
			}
			if style, ok := optionValue["style"].(float64); ok {
				option.Style = uint32(style)
			}
			return option, nil
		}
	}
	return nil, fmt.Errorf("invalid option value format")
}

// 创建地理位置值
func NewLocationValue(id, latitude, longitude, title string) []CellLocationValue {
	return []CellLocationValue{
		{
			SourceType: 1,
			Id:         id,
			Latitude:   latitude,
			Longitude:  longitude,
			Title:      title,
		},
	}
}

// 解析地理位置值
func ParseLocationValue(value any) (*CellLocationValue, error) {
	if values, ok := value.([]interface{}); ok && len(values) > 0 {
		if locationValue, ok := values[0].(map[string]interface{}); ok {
			location := &CellLocationValue{}
			if sourceType, ok := locationValue["source_type"].(float64); ok {
				location.SourceType = uint32(sourceType)
			}
			if id, ok := locationValue["id"].(string); ok {
				location.Id = id
			}
			if latitude, ok := locationValue["latitude"].(string); ok {
				location.Latitude = latitude
			}
			if longitude, ok := locationValue["longitude"].(string); ok {
				location.Longitude = longitude
			}
			if title, ok := locationValue["title"].(string); ok {
				location.Title = title
			}
			return location, nil
		}
	}
	return nil, fmt.Errorf("invalid location value format")
}

func (doc *MultiTableDoc) AddRow(rows []*AddRecord) error {
	url := g_baseUrl + "/cgi-bin/wedoc/smartsheet/add_records"

	// 构建请求体
	reqBody := addRecordRequest{
		Docid:   doc.Docid,
		SheetId: doc.SheetId,
		KeyType: CELL_VALUE_KEY_TYPE_FIELD_TITLE,
		Records: rows,
	}

	req, err := putil.NewHttpRequestJson(http.MethodPost, url, nil,
		getTokenQuerys(),
		reqBody)
	if err != nil {
		return plogger.LogErr(err)
	}

	resp, err := putil.HttpDo(req)
	if err != nil {
		return plogger.LogErr(err)
	}

	var respData addRecordResponse
	err = json.Unmarshal(resp, &respData)
	if err != nil {
		return plogger.LogErr(err)
	}

	// 检查响应错误
	if respData.Errcode != 0 {
		return plogger.LogErr(fmt.Errorf("add records failed: errcode=%d, errmsg=%s", respData.Errcode, respData.Errmsg))
	}

	return nil
}

// --------------------------------------------------
// 添加记录请求结构
type addRecordRequest struct {
	Docid   string       `json:"docid"`
	SheetId string       `json:"sheet_id"`
	KeyType string       `json:"key_type,omitempty"`
	Records []*AddRecord `json:"records"`
}

// 添加记录响应结构
type addRecordResponse struct {
	Errcode int            `json:"errcode"`
	Errmsg  string         `json:"errmsg"`
	Records []CommonRecord `json:"records"`
}

// 添加记录结构
type AddRecord struct {
	Values map[string]any `json:"values"`
}

// 通用记录结构
type CommonRecord struct {
	AddRecord
	RecordId string `json:"record_id"`
}

// 查询记录请求结构
type GetRecordRequest struct {
	Docid       string      `json:"docid"`
	SheetId     string      `json:"sheet_id"`
	ViewId      string      `json:"view_id,omitempty"`
	RecordIds   []string    `json:"record_ids,omitempty"`
	KeyType     string      `json:"key_type,omitempty"`
	FieldTitles []string    `json:"field_titles,omitempty"`
	FieldIds    []string    `json:"field_ids,omitempty"`
	Sort        []SortRule  `json:"sort,omitempty"`
	Offset      uint32      `json:"offset,omitempty"`
	Limit       uint32      `json:"limit,omitempty"` //最大值1000，即使设置为0，超过1000也会被截断
	Ver         uint32      `json:"ver,omitempty"`
	FilterSpec  *filterSpec `json:"filter_spec,omitempty"`
}

// 查询记录响应结构
type getRecordResponse struct {
	Errcode int            `json:"errcode"`
	Errmsg  string         `json:"errmsg"`
	Total   uint32         `json:"total"`
	HasMore bool           `json:"has_more"`
	Next    uint32         `json:"next"`
	Records []RecordDetail `json:"records"`
	Ver     uint32         `json:"ver"`
}

// 记录详情结构
type RecordDetail struct {
	RecordId    string         `json:"record_id"`
	CreateTime  string         `json:"create_time"`
	UpdateTime  string         `json:"update_time"`
	Values      map[string]any `json:"values"`
	CreatorName string         `json:"creator_name"`
	UpdaterName string         `json:"updater_name"`
}

// 排序规则
type SortRule struct {
	FieldTitle string `json:"field_title"`
	Desc       bool   `json:"desc,omitempty"`
}

// 过滤条件
type filterSpec struct {
	Conjunction string      `json:"conjunction"`
	Conditions  []condition `json:"conditions"`
}

type condition struct {
	FieldId     string       `json:"field_id"`
	FieldType   wxFieldType  `json:"field_type"`
	Operator    string       `json:"operator"`
	StringValue *stringValue `json:"string_value,omitempty"`
}

type stringValue struct {
	Value []string `json:"value"`
}

// --------------------------------------------------
const (
	CELL_VALUE_KEY_TYPE_FIELD_TITLE = "CELL_VALUE_KEY_TYPE_FIELD_TITLE" // 以标题索引一个字段
	CELL_VALUE_KEY_TYPE_FIELD_ID    = "CELL_VALUE_KEY_TYPE_FIELD_ID"    // 以ID索引一个字段
)

// 文本类型单元格值
type CellTextValue struct {
	Type string `json:"type"` // "text" 或 "url"
	Text string `json:"text"`
	Link string `json:"link,omitempty"` // 当type为url时使用
}

// 图片类型单元格值
type CellImageValue struct {
	Id       string `json:"id"`
	Title    string `json:"title"`
	ImageUrl string `json:"image_url"`
	Width    int32  `json:"width"`
	Height   int32  `json:"height"`
}

// 文件类型单元格值
type CellAttachmentValue struct {
	Name     string `json:"name"`
	Size     int32  `json:"size"`
	FileExt  string `json:"file_ext"`
	FileUrl  string `json:"file_url"`
	FileType string `json:"file_type"`
	DocType  string `json:"doc_type"`
}

// 用户类型单元格值
type CellUserValue struct {
	UserId            string `json:"user_id"`
	TmpExternalUserId string `json:"tmp_external_userid,omitempty"`
}

// 链接类型单元格值
type CellUrlValue struct {
	Type string `json:"type"` // "url"
	Text string `json:"text"`
	Link string `json:"link"`
}

// 选项类型（用于单选和多选）
type CellOption struct {
	Id string `json:"id"`
	// 你没看错，col配置用int，这里uint，企微官方文档就是这么写的，本来也确实没关系的细节
	Style uint32 `json:"style"`
	Text  string `json:"text"`
}

// 地理位置类型单元格值
type CellLocationValue struct {
	SourceType uint32 `json:"source_type"` // 填1，表示来源为腾讯地图
	Id         string `json:"id"`
	Latitude   string `json:"latitude"`
	Longitude  string `json:"longitude"`
	Title      string `json:"title"`
}

// 自动编号类型单元格值
type CellAutoNumberValue struct {
	Seq  string `json:"seq"`
	Text string `json:"text"`
}

// 数字类型单元格值（实际为float64）
type CellNumValue = float64

// 日期时间类型单元格值（实际为string，毫秒时间戳）
type CellTimeValue = string
