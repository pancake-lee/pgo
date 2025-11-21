package papitable

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

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
// 查询字段
func (doc *MultiTableDoc) GetCols() ([]*Field, error) {
	url := fmt.Sprintf("%s/fusion/v1/datasheets/%s/fields", g_baseUrl, doc.DatasheetId)

	req, err := putil.NewHttpRequestJson(http.MethodGet, url,
		getTokenHeader(), map[string]string{}, nil)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	resp, err := putil.HttpDo(req)
	if err != nil {
		return nil, plogger.LogErr(err)
	}
	// plogger.Debugf("response: %s", string(resp))

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
			if respData.Code == 400 && strings.Contains(respData.Message, "必须唯一") {
				plogger.Warnf("AddCol failed: %s, field name[%s] already exists",
					respData.Message, field.Name)
				continue
			}

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
// 修改列并没有专门API
// 如果要做其实是删除列然后重新加列，并且需要重新写入该列数据
// 但是这样的性能还不如全表重建，因为都要遍历所有行
// func (doc *MultiTableDoc) EditCol(fields []*AddField) (ret []*Field, err error) {
// }

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
// 成员类型，因为用户列表API调不通，所以暂时用不上
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
// Attachment (文件/附件) 字段
func NewAttachmentCol(colName string) *AddField {
	return &AddField{
		Name:     colName,
		Type:     FIELD_TYPE_ATTACHMENT,
		Property: nil,
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
	if options == nil {
		options = []*SelectFieldOption{}
	}
	return &AddField{
		Name: colName,
		Type: FIELD_TYPE_SINGLE_SELECT,
		Property: map[string]interface{}{
			"options": options,
		},
	}
}

func NewMultiSingleSelectCol(colName string, options []*SelectFieldOption) *AddField {
	return &AddField{
		Name: colName,
		Type: FIELD_TYPE_MULTI_SELECT,
		Property: map[string]interface{}{
			"options": options,
		},
	}
}

type SelectFieldOptionHandler struct {
	optionMap map[string]*SelectFieldOption
}

func (h *SelectFieldOptionHandler) Reset() {
	h.optionMap = make(map[string]*SelectFieldOption)
}
func (h *SelectFieldOptionHandler) RegOptionI(id int32, text string, style SelectFieldOptionColor) {
	h.RegOptionS(putil.Int32ToStr(id), text, style)
}
func (h *SelectFieldOptionHandler) RegOptionS(id, text string, style SelectFieldOptionColor) {
	if h.optionMap == nil {
		h.optionMap = make(map[string]*SelectFieldOption)
	}
	h.optionMap[id] = &SelectFieldOption{
		Id:    id,
		Text:  text,
		Style: style,
	}
}
func (h *SelectFieldOptionHandler) GetOptionList() []*SelectFieldOption {
	if h.optionMap == nil {
		return []*SelectFieldOption{}
	}
	var options []*SelectFieldOption
	for _, option := range h.optionMap {
		options = append(options, option)
	}
	return options
}
func (h *SelectFieldOptionHandler) GetOptionByText(text string) *SelectFieldOption {
	if h.optionMap == nil {
		return nil
	}
	for _, option := range h.optionMap {
		if option.Text == text {
			return option
		}
	}
	return nil
}

func (h *SelectFieldOptionHandler) GetCellOptionById_I(id int32) *CellOption {
	return h.GetCellOptionById_S(putil.Int32ToStr(id))
}
func (h *SelectFieldOptionHandler) GetCellOptionById_S(id string) *CellOption {
	if h.optionMap == nil {
		return NewOptionValueByStr("")
	}
	opt, ok := h.optionMap[id]
	if !ok {
		return NewOptionValueByStr("")
	}
	return NewOptionValue(opt)
}

func (h *SelectFieldOptionHandler) GetCellOptionByText(text string) *CellOption {
	if h.optionMap == nil {
		return NewOptionValueByStr("")
	}
	for _, opt := range h.optionMap {
		if opt.Text == text {
			return NewOptionValue(opt)
		}
	}
	return NewOptionValueByStr("")
}

// --------------------------------------------------
// Formula（智能公式）字段，暂时只处理string类型
func NewFormulaCol(colName string, expression string) *AddField {
	return &AddField{
		Name: colName,
		Type: FIELD_TYPE_FORMULA,
		Property: map[string]any{
			"expression": expression,
			"valueType":  "String",
		},
	}
}

// --------------------------------------------------
// OneWayLink（单向关联）字段
func NewOneWayLinkCol(colName string, foreignDatasheetId string, multiSelect bool) *AddField {
	return &AddField{
		Name: colName,
		Type: FIELD_TYPE_ONE_WAY_LINK,
		Property: map[string]any{
			"foreignDatasheetId": foreignDatasheetId,
			"limitSingleRecord":  !multiSelect,
			// "limitToViewId":      limitToViewId, // 没有创建视图的API，就不能完成整个程序闭环
		},
	}
}

// --------------------------------------------------
// MagicLookUp（神奇引用）字段
type MagicLookUpConfig struct {
	RelatedLinkFieldId string                 // 引用的当前表的关联字段 ID（注意这是本表）
	TargetFieldId      string                 // 关联表中查询的字段 ID（注意这是关联的表）
	RollupFunction     string                 // 汇总函数，如 "VALUES", "AVERAGE", "COUNT" 等
	EnableFilterSort   bool                   // 是否开启筛选和排序
	SortInfo           *MagicLookUpSortInfo   // 排序设置
	FilterInfo         *MagicLookUpFilterInfo // 筛选设置
	LookUpLimit        string                 // 限制展示的记录数量 "ALL" 或 "FIRST"
}

type MagicLookUpSortInfo struct {
	Rules []MagicLookUpSortRule `json:"rules"`
}

type MagicLookUpSortRule struct {
	FieldId string `json:"fieldId"` // 用于排序的字段ID
	Desc    bool   `json:"desc"`    // 是否按降序排序
}

type MagicLookUpFilterInfo struct {
	Conjunction string                       `json:"conjunction"` // "and" 或 "or"
	Conditions  []MagicLookUpFilterCondition `json:"conditions"`
}

type MagicLookUpFilterCondition struct {
	FieldId   string        `json:"fieldId"`   // 筛选字段的字段ID
	FieldType string        `json:"fieldType"` // 筛选字段的字段类型
	Operator  string        `json:"operator"`  // 筛选条件的运算符
	Value     []interface{} `json:"value"`     // 筛选条件的基准值
}

func NewMagicLookUpCol(colName string, config *MagicLookUpConfig) *AddField {
	property := map[string]interface{}{
		"relatedLinkFieldId": config.RelatedLinkFieldId,
		"targetFieldId":      config.TargetFieldId,
		"rollupFunction":     config.RollupFunction,
	}

	if config.LookUpLimit != "" {
		property["lookUpLimit"] = config.LookUpLimit
	} else {
		property["lookUpLimit"] = "ALL"
	}

	if config.EnableFilterSort {
		property["enableFilterSort"] = true
		if config.SortInfo != nil {
			property["sortInfo"] = config.SortInfo
		}
		if config.FilterInfo != nil {
			property["filterInfo"] = config.FilterInfo
		}
	}

	return &AddField{
		Name:     colName,
		Type:     FIELD_TYPE_MAGIC_LOOKUP,
		Property: property,
	}
}

// NewSimpleMagicLookUpCol 创建简单的神奇引用字段（不带筛选和排序）
func NewSimpleMagicLookUpCol(colName string, relatedLinkFieldId string, targetFieldId string) *AddField {
	return NewMagicLookUpCol(colName, &MagicLookUpConfig{
		RelatedLinkFieldId: relatedLinkFieldId,
		TargetFieldId:      targetFieldId,
		RollupFunction:     "VALUES",
		LookUpLimit:        "ALL",
	})
}
