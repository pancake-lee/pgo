package pweixin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

// 设置多表格文档的列列表：
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
		myColNameList = append(myColNameList, col.FieldTitle)
		nameToMyColMap[col.FieldTitle] = col
	}

	// --------------------------------------------------
	colList, err := doc.GetCols(nil, nil)
	if err != nil {
		return isEdited, plogger.LogErr(err)
	}

	colNameList := make([]string, 0, len(colList))
	idToColMap := make(map[string]*Field, len(colList))
	nameToColMap := make(map[string]*Field, len(colList))
	for _, col := range colList {
		colNameList = append(colNameList, col.FieldTitle)
		idToColMap[col.FieldId] = col
		nameToColMap[col.FieldTitle] = col
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
			plogger.Debugf("added col[%s] title[%s] type[%s]",
				newField.FieldId, newField.FieldTitle, newField.FieldType)
		}
	}
	// --------------------------------------------------
	if deleteExcept {
		delColNameList := putil.StrListExcept(colNameList, myColNameList)
		if len(delColNameList) > 0 {
			delColIdList := make([]string, 0, len(delColNameList))
			for _, colName := range delColNameList {
				delColIdList = append(delColIdList, nameToColMap[colName].FieldId)
			}
			err = doc.DelCol(delColIdList)
			if err != nil {
				return isEdited, plogger.LogErr(err)
			}
			isEdited = true
			for _, colId := range delColIdList {
				plogger.Debugf("deleted col[%s] title[%s]", idToColMap[colId].FieldId, idToColMap[colId].FieldTitle)
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
	url := g_baseUrl + "/cgi-bin/wedoc/smartsheet/delete_fields"

	// 构建请求体
	reqBody := deleteFieldRequest{
		Docid:    doc.Docid,
		SheetId:  doc.SheetId,
		FieldIds: fieldIds,
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

	var respData deleteFieldResponse
	err = json.Unmarshal(resp, &respData)
	if err != nil {
		return plogger.LogErr(err)
	}

	// 检查响应错误
	if respData.Errcode != 0 {
		return plogger.LogErr(fmt.Errorf("delete fields failed: errcode=%d, errmsg=%s", respData.Errcode, respData.Errmsg))
	}

	plogger.Debug("DelCol success, deleted field count:", len(fieldIds))
	return nil
}

// 删除字段请求结构
type deleteFieldRequest struct {
	Docid    string   `json:"docid"`
	SheetId  string   `json:"sheet_id"`
	FieldIds []string `json:"field_ids"`
}

// 删除字段响应结构
type deleteFieldResponse struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

// --------------------------------------------------
// 内部方法：查询字段
func (doc *MultiTableDoc) GetCols(fieldIds, fieldTitles []string) ([]*Field, error) {
	url := g_baseUrl + "/cgi-bin/wedoc/smartsheet/get_fields"

	// 构建请求体
	reqBody := getFieldRequest{
		Docid:       doc.Docid,
		SheetId:     doc.SheetId,
		FieldIds:    fieldIds,
		FieldTitles: fieldTitles,
		Offset:      0,
		Limit:       1000,
	}

	req, err := putil.NewHttpRequestJson(http.MethodPost, url, nil,
		getTokenQuerys(),
		reqBody)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	resp, err := putil.HttpDo(req)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	var respData getFieldResponse
	err = json.Unmarshal(resp, &respData)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	// 检查响应错误
	if respData.Errcode != 0 {
		return nil, plogger.LogErr(fmt.Errorf("get fields failed: errcode=%d, errmsg=%s", respData.Errcode, respData.Errmsg))
	}

	plogger.Debug("GetCol success, total fields:", respData.Total)
	return respData.Fields, nil
}

// 查询字段请求结构
type getFieldRequest struct {
	Docid       string   `json:"docid"`
	SheetId     string   `json:"sheet_id"`
	ViewId      string   `json:"view_id,omitempty"`
	FieldIds    []string `json:"field_ids,omitempty"`
	FieldTitles []string `json:"field_titles,omitempty"`
	Offset      int      `json:"offset,omitempty"`
	Limit       int      `json:"limit,omitempty"`
}

// 查询字段响应结构
type getFieldResponse struct {
	Errcode int      `json:"errcode"`
	Errmsg  string   `json:"errmsg"`
	Total   int      `json:"total"`
	Fields  []*Field `json:"fields"`
}

// --------------------------------------------------
// 添加指定类型的字段
func (doc *MultiTableDoc) AddCol(fields []*AddField) (ret []*Field, err error) {
	if len(fields) == 0 {
		return []*Field{}, nil
	}
	url := g_baseUrl + "/cgi-bin/wedoc/smartsheet/add_fields"

	// 构建请求体
	reqBody := addFieldRequest{
		Docid:   doc.Docid,
		SheetId: doc.SheetId, // 如果需要指定sheet_id，可以在结构体中添加该字段
		Fields:  fields,
	}

	req, err := putil.NewHttpRequestJson(http.MethodPost, url, nil,
		getTokenQuerys(),
		reqBody)
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

	// 检查响应错误
	if respData.Errcode != 0 {
		return nil, plogger.LogErr(fmt.Errorf("add field failed: errcode=%d, errmsg=%s", respData.Errcode, respData.Errmsg))
	}

	plogger.Debug("AddCol success, field_id:", respData.Fields[0].FieldId)
	return respData.Fields, nil
}

// --------------------------------------------------
// 添加字段请求结构
type addFieldRequest struct {
	Docid   string      `json:"docid"`
	SheetId string      `json:"sheet_id"`
	Fields  []*AddField `json:"fields"`
}

// 添加字段响应结构
type addFieldResponse struct {
	Errcode int      `json:"errcode"`
	Errmsg  string   `json:"errmsg"`
	Fields  []*Field `json:"fields"`
}

type Field struct {
	AddField
	FieldId string `json:"field_id"`
}

type AddField struct {
	FieldTitle           string                     `json:"field_title"`
	FieldType            wxFieldType                `json:"field_type"`
	PropertyNumber       *numberFieldProperty       `json:"property_number,omitempty"`
	PropertyCheckbox     *checkboxFieldProperty     `json:"property_checkbox,omitempty"`
	PropertyDateTime     *dateTimeFieldProperty     `json:"property_date_time,omitempty"`
	PropertyAttachment   *attachmentFieldProperty   `json:"property_attachment,omitempty"`
	PropertyUser         *userFieldProperty         `json:"property_user,omitempty"`
	PropertyUrl          *urlFieldProperty          `json:"property_url,omitempty"`
	PropertySelect       *selectFieldProperty       `json:"property_select,omitempty"`
	PropertyCreatedTime  *createdTimeFieldProperty  `json:"property_created_time,omitempty"`
	PropertyModifiedTime *modifiedTimeFieldProperty `json:"property_modified_time,omitempty"`
	PropertyProgress     *progressFieldProperty     `json:"property_progress,omitempty"`
	PropertySingleSelect *singleSelectFieldProperty `json:"property_single_select,omitempty"`
	PropertyReference    *referenceFieldProperty    `json:"property_reference,omitempty"`
	PropertyLocation     *locationFieldProperty     `json:"property_location,omitempty"`
	PropertyAutoNumber   *autoNumberFieldProperty   `json:"property_auto_number,omitempty"`
	PropertyCurrency     *currencyFieldProperty     `json:"property_currency,omitempty"`
	PropertyWwGroup      *wwGroupFieldProperty      `json:"property_ww_group,omitempty"`
	PropertyPercentage   *percentageFieldProperty   `json:"property_percentage,omitempty"`
	PropertyBarcode      *barcodeFieldProperty      `json:"property_barcode,omitempty"`
}

// --------------------------------------------------
// 字段类型
// --------------------------------------------------
// 字段类型常量
type wxFieldType string

const (
	FIELD_TYPE_TEXT          wxFieldType = "FIELD_TYPE_TEXT"
	FIELD_TYPE_NUMBER        wxFieldType = "FIELD_TYPE_NUMBER"
	FIELD_TYPE_CHECKBOX      wxFieldType = "FIELD_TYPE_CHECKBOX"
	FIELD_TYPE_DATE_TIME     wxFieldType = "FIELD_TYPE_DATE_TIME"
	FIELD_TYPE_IMAGE         wxFieldType = "FIELD_TYPE_IMAGE"
	FIELD_TYPE_ATTACHMENT    wxFieldType = "FIELD_TYPE_ATTACHMENT"
	FIELD_TYPE_USER          wxFieldType = "FIELD_TYPE_USER"
	FIELD_TYPE_URL           wxFieldType = "FIELD_TYPE_URL"
	FIELD_TYPE_SELECT        wxFieldType = "FIELD_TYPE_SELECT"
	FIELD_TYPE_SINGLE_SELECT wxFieldType = "FIELD_TYPE_SINGLE_SELECT"
	FIELD_TYPE_PROGRESS      wxFieldType = "FIELD_TYPE_PROGRESS"
	FIELD_TYPE_PHONE_NUMBER  wxFieldType = "FIELD_TYPE_PHONE_NUMBER"
	FIELD_TYPE_EMAIL         wxFieldType = "FIELD_TYPE_EMAIL"
	FIELD_TYPE_LOCATION      wxFieldType = "FIELD_TYPE_LOCATION"
	FIELD_TYPE_CURRENCY      wxFieldType = "FIELD_TYPE_CURRENCY"
	FIELD_TYPE_PERCENTAGE    wxFieldType = "FIELD_TYPE_PERCENTAGE"
)

// --------------------------------------------------
func NewSimpleTextCol(colName string) *AddField {
	return &AddField{
		FieldTitle: colName,
		FieldType:  FIELD_TYPE_TEXT,
	}
}

// --------------------------------------------------
type numberFieldProperty struct {
	DecimalPlaces int  `json:"decimal_places"`
	UseSeparate   bool `json:"use_separate"`
}

func NewSimpleNumCol(colName string) *AddField {
	return &AddField{
		FieldTitle: colName,
		FieldType:  FIELD_TYPE_NUMBER,
		PropertyNumber: &numberFieldProperty{
			DecimalPlaces: 0,
			UseSeparate:   false,
		},
	}
}

// --------------------------------------------------
type dateTimeFieldProperty struct {
	Format   string `json:"format"`
	AutoFill bool   `json:"auto_fill"`
}

func NewSimpleTimeCol(colName string) *AddField {
	return &AddField{
		FieldTitle: colName,
		FieldType:  FIELD_TYPE_DATE_TIME,
		PropertyDateTime: &dateTimeFieldProperty{
			// https://developer.work.weixin.qq.com/document/path/99914#format
			// 企微定义要mm而不是MM，确实月和分都是mm，神奇吧
			Format:   "yyyy-mm-dd",
			AutoFill: false,
		},
	}
}

// --------------------------------------------------
type userFieldProperty struct {
	IsMultiple bool `json:"is_multiple"`
	IsNotified bool `json:"is_notified"`
}

func NewSimpleUserCol(colName string, isMultiple bool) *AddField {
	return &AddField{
		FieldTitle: colName,
		FieldType:  FIELD_TYPE_USER,
		PropertyUser: &userFieldProperty{
			IsMultiple: isMultiple,
			IsNotified: false, // 是否通知用户
		},
	}
}

// --------------------------------------------------
type SelectFieldOption struct {
	Id    string `json:"id,omitempty"`
	Text  string `json:"text"`
	Style int    `json:"style,omitempty"`
}
type singleSelectFieldProperty struct {
	IsQuickAdd bool                 `json:"is_quick_add"`
	Options    []*SelectFieldOption `json:"options"`
}

func NewSimpleSingleSelectCol(colName string, options []*SelectFieldOption) *AddField {
	return &AddField{
		FieldTitle: colName,
		FieldType:  FIELD_TYPE_SINGLE_SELECT,
		PropertySingleSelect: &singleSelectFieldProperty{
			IsQuickAdd: true,
			Options:    options,
		},
	}
}

// --------------------------------------------------
type checkboxFieldProperty struct {
	Checked bool `json:"checked"`
}

type attachmentFieldProperty struct {
	DisplayMode string `json:"display_mode"`
}

type urlFieldProperty struct {
	Type string `json:"type"`
}

type selectFieldProperty struct {
	IsQuickAdd bool                 `json:"is_quick_add"`
	Options    []*SelectFieldOption `json:"options"`
}

type createdTimeFieldProperty struct {
	Format string `json:"format"`
}

type modifiedTimeFieldProperty struct {
	Format string `json:"format"`
}

type progressFieldProperty struct {
	DecimalPlaces int `json:"decimal_places"`
}

type referenceFieldProperty struct {
	SubId      string `json:"sub_id"`
	FieldId    string `json:"filed_id"`
	IsMultiple bool   `json:"is_multiple"`
	ViewId     string `json:"view_id"`
}

type locationFieldProperty struct {
	InputType string `json:"input_type"`
}

type autoNumberFieldProperty struct {
	Type                   string       `json:"type"`
	Rules                  []numberRule `json:"rules"`
	ReformatExistingRecord bool         `json:"reformat_existing_record"`
}

type currencyFieldProperty struct {
	CurrencyType  string `json:"currency_type"`
	DecimalPlaces int    `json:"decimal_places"`
	UseSeparate   bool   `json:"use_separate"`
}

type wwGroupFieldProperty struct {
	AllowMultiple bool `json:"allow_multiple"`
}

type percentageFieldProperty struct {
	DecimalPlaces int  `json:"decimal_places"`
	UseSeparate   bool `json:"use_separate"`
}

type barcodeFieldProperty struct {
	MobileScanOnly bool `json:"mobile_scan_only"`
}

type numberRule struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// --------------------------------------------------
// 颜色
type SelectFieldOptionStyle int

const (
	OptStyle_LightRed1     SelectFieldOptionStyle = 1  // 浅红1
	OptStyle_LightOrange1  SelectFieldOptionStyle = 2  // 浅橙1
	OptStyle_LightSkyBlue1 SelectFieldOptionStyle = 3  // 浅天蓝1
	OptStyle_LightGreen1   SelectFieldOptionStyle = 4  // 浅绿1
	OptStyle_LightPurple1  SelectFieldOptionStyle = 5  // 浅紫1
	OptStyle_LightPink1    SelectFieldOptionStyle = 6  // 浅粉红1
	OptStyle_LightGray1    SelectFieldOptionStyle = 7  // 浅灰1
	OptStyle_White         SelectFieldOptionStyle = 8  // 白
	OptStyle_Gray          SelectFieldOptionStyle = 9  // 灰
	OptStyle_LightBlue1    SelectFieldOptionStyle = 10 // 浅蓝1
	OptStyle_LightBlue2    SelectFieldOptionStyle = 11 // 浅蓝2
	OptStyle_Blue          SelectFieldOptionStyle = 12 // 蓝
	OptStyle_LightSkyBlue2 SelectFieldOptionStyle = 13 // 浅天蓝2
	OptStyle_SkyBlue       SelectFieldOptionStyle = 14 // 天蓝
	OptStyle_LightGreen2   SelectFieldOptionStyle = 15 // 浅绿2
	OptStyle_Green         SelectFieldOptionStyle = 16 // 绿
	OptStyle_LightRed2     SelectFieldOptionStyle = 17 // 浅红2
	OptStyle_Red           SelectFieldOptionStyle = 18 // 红
	OptStyle_LightOrange2  SelectFieldOptionStyle = 19 // 浅橙2
	OptStyle_Orange        SelectFieldOptionStyle = 20 // 橙
	OptStyle_LightYellow1  SelectFieldOptionStyle = 21 // 浅黄1
	OptStyle_LightYellow2  SelectFieldOptionStyle = 22 // 浅黄2
	OptStyle_Yellow        SelectFieldOptionStyle = 23 // 黄
	OptStyle_LightPurple2  SelectFieldOptionStyle = 24 // 浅紫2
	OptStyle_Purple        SelectFieldOptionStyle = 25 // 紫
	OptStyle_LightPink2    SelectFieldOptionStyle = 26 // 浅粉红2
	OptStyle_Pink          SelectFieldOptionStyle = 27 // 粉红
)
