package papitable

import (
	"fmt"
	"time"
)

// --------------------------------------------------
// 行数据中的字段Def/New/Parse
// --------------------------------------------------

func NewNumValue(value float64) float64 {
	return value
}
func ParseNumValue(value any) (float64, error) {
	if v, ok := value.(float64); ok {
		return v, nil
	}
	return 0, fmt.Errorf("invalid number value type: %T", value)
}

// --------------------------------------------------
// func NewSingleTextValue(text string) string {
// 	return text
// }
// func ParseSingleTextValue(value any) (string, error) {
// 	if text, ok := value.(string); ok {
// 		return text, nil
// 	}
// 	return "", fmt.Errorf("invalid single text value type: %T", value)
// }

// --------------------------------------------------
func NewTextValue(text string) string {
	return text
}

func ParseTextValue(value any) (string, error) {
	if text, ok := value.(string); ok {
		return text, nil
	}
	return "", fmt.Errorf("invalid text value type: %T", value)
}

// --------------------------------------------------
func NewSingleSelectValue(option string) string {
	return option
}

func ParseSingleSelectValue(value any) (string, error) {
	if option, ok := value.(string); ok {
		return option, nil
	}
	return "", fmt.Errorf("invalid single select value type: %T", value)
}

// --------------------------------------------------
func NewMultiSelectValue(options ...string) []string {
	return options
}

func ParseMultiSelectValue(value any) ([]string, error) {
	if options, ok := value.([]interface{}); ok {
		result := make([]string, 0, len(options))
		for _, opt := range options {
			if str, ok := opt.(string); ok {
				result = append(result, str)
			}
		}
		return result, nil
	}
	return nil, fmt.Errorf("invalid multi select value type: %T", value)
}

// --------------------------------------------------
func NewTimeValue(t time.Time) float64 {
	return float64(t.UnixMilli())
}

func ParseTimeValue(value any) (time.Time, error) {
	if timestamp, ok := value.(float64); ok && timestamp >= 0 {
		return time.UnixMilli(int64(timestamp)), nil
	}

	return time.UnixMilli(0), fmt.Errorf("invalid time value type: %v", value)
}

// --------------------------------------------------
// 附件类型单元格值
type CellAttachmentValue struct {
	MimeType string `json:"mimeType"`          // 附件的媒体类型
	Name     string `json:"name"`              // 附件的名称
	Size     int32  `json:"size"`              // 附件的大小，单位为字节
	Width    int32  `json:"width,omitempty"`   // 如果附件是图片格式，表示图片的宽度，单位为px
	Height   int32  `json:"height,omitempty"`  // 如果附件是图片格式，表示图片的高度，单位为px
	Token    string `json:"token"`             // 附件的访问路径
	Preview  string `json:"preview,omitempty"` // 如果附件是PDF格式，将会生成一个预览图，用户可以通过此网址访问
}

func NewAttachmentValue(attachments ...CellAttachmentValue) []CellAttachmentValue {
	return attachments
}

func ParseAttachmentValue(value any) ([]CellAttachmentValue, error) {
	if attachments, ok := value.([]interface{}); ok {
		result := make([]CellAttachmentValue, 0, len(attachments))
		for _, att := range attachments {
			if attMap, ok := att.(map[string]interface{}); ok {
				attachment := CellAttachmentValue{}
				attMap = attMap // TODO
				result = append(result, attachment)
			}
		}
		return result, nil
	}
	return nil, fmt.Errorf("invalid attachment value type: %T", value)
}

// --------------------------------------------------
// 成员类型单元格值
type CellUserValue struct {
	// Id     string `json:"id"`               // 组织单元的ID
	UserId string `json:"id"`               // 组织单元的ID
	Type   int32  `json:"type"`             // 组织单元的类型，1是小组，3是成员
	Name   string `json:"name"`             // 小组或成员的名称
	Avatar string `json:"avatar,omitempty"` // 头像URL，只读，不可写入
}

func NewUserValue(userIds ...string) []CellUserValue {
	values := make([]CellUserValue, len(userIds))
	for i, userId := range userIds {
		values[i] = CellUserValue{UserId: userId}
	}
	return values
}

func ParseUserValue(value any) ([]CellUserValue, error) {
	if members, ok := value.([]interface{}); ok {
		result := make([]CellUserValue, 0, len(members))
		for _, mem := range members {
			if memMap, ok := mem.(map[string]interface{}); ok {
				member := CellUserValue{}
				if userId, ok := memMap["id"].(string); ok {
					member.UserId = userId
				}
				if memberType, ok := memMap["type"].(float64); ok {
					member.Type = int32(memberType)
				}
				if name, ok := memMap["name"].(string); ok {
					member.Name = name
				}
				if avatar, ok := memMap["avatar"].(string); ok {
					member.Avatar = avatar
				}
				result = append(result, member)
			}
		}
		return result, nil
	}
	return nil, fmt.Errorf("invalid member value type: %T", value)
}

// --------------------------------------------------
func NewCheckboxValue(checked bool) bool {
	return checked
}

func ParseCheckboxValue(value any) (bool, error) {
	if checked, ok := value.(bool); ok {
		return checked, nil
	}
	return false, fmt.Errorf("invalid checkbox value type: %T", value)
}

// --------------------------------------------------
func NewRatingValue(rating float64) float64 {
	return rating
}

func ParseRatingValue(value any) (float64, error) {
	if rating, ok := value.(float64); ok {
		return rating, nil
	}
	return 0, fmt.Errorf("invalid rating value type: %T", value)
}

// --------------------------------------------------
// 链接类型单元格值
type CellUrlValue struct {
	Title   string `json:"title"`   // 网页标题
	Text    string `json:"text"`    // 网页地址
	Favicon string `json:"favicon"` // 网页 ICON
}

func NewUrlValue(title, text, favicon string) CellUrlValue {
	return CellUrlValue{
		Title:   title,
		Text:    text,
		Favicon: favicon,
	}
}

func ParseUrlValue(value any) (*CellUrlValue, error) {
	if urlMap, ok := value.(map[string]interface{}); ok {
		url := &CellUrlValue{}
		if title, ok := urlMap["title"].(string); ok {
			url.Title = title
		}
		if text, ok := urlMap["text"].(string); ok {
			url.Text = text
		}
		if favicon, ok := urlMap["favicon"].(string); ok {
			url.Favicon = favicon
		}
		return url, nil
	}
	return nil, fmt.Errorf("invalid url value type: %T", value)
}

// --------------------------------------------------
func NewPhoneValue(phone string) string {
	return phone
}

func ParsePhoneValue(value any) (string, error) {
	if phone, ok := value.(string); ok {
		return phone, nil
	}
	return "", fmt.Errorf("invalid phone value type: %T", value)
}

// --------------------------------------------------
func NewEmailValue(email string) string {
	return email
}

func ParseEmailValue(value any) (string, error) {
	if email, ok := value.(string); ok {
		return email, nil
	}
	return "", fmt.Errorf("invalid email value type: %T", value)
}

// --------------------------------------------------
// 工作文档类型单元格值
type CellWorkDocValue struct {
	DocumentId string `json:"documentId"`
	Title      string `json:"title"`
}

func NewWorkDocValue(workDocs ...CellWorkDocValue) []CellWorkDocValue {
	return workDocs
}

func ParseWorkDocValue(value any) ([]CellWorkDocValue, error) {
	if workDocs, ok := value.([]interface{}); ok {
		result := make([]CellWorkDocValue, 0, len(workDocs))
		for _, doc := range workDocs {
			if docMap, ok := doc.(map[string]interface{}); ok {
				workDoc := CellWorkDocValue{}
				if documentId, ok := docMap["documentId"].(string); ok {
					workDoc.DocumentId = documentId
				}
				if title, ok := docMap["title"].(string); ok {
					workDoc.Title = title
				}
				result = append(result, workDoc)
			}
		}
		return result, nil
	}
	return nil, fmt.Errorf("invalid workdoc value type: %T", value)
}

// --------------------------------------------------
func NewOneWayLinkValue(recordIds ...string) []string {
	return recordIds
}

func ParseOneWayLinkValue(value any) ([]string, error) {
	if recordIds, ok := value.([]interface{}); ok {
		result := make([]string, 0, len(recordIds))
		for _, id := range recordIds {
			if str, ok := id.(string); ok {
				result = append(result, str)
			}
		}
		return result, nil
	}
	return nil, fmt.Errorf("invalid one way link value type: %T", value)
}

// --------------------------------------------------
// 双向链接值
func NewTwoWayLinkValue(recordIds ...string) []string {
	return recordIds
}

func ParseTwoWayLinkValue(value any) ([]string, error) {
	if recordIds, ok := value.([]interface{}); ok {
		result := make([]string, 0, len(recordIds))
		for _, id := range recordIds {
			if str, ok := id.(string); ok {
				result = append(result, str)
			}
		}
		return result, nil
	}
	return nil, fmt.Errorf("invalid two way link value type: %T", value)
}

// --------------------------------------------------
// 解析自动编号值（只读字段）
func ParseAutoNumberValue(value any) (float64, error) {
	if num, ok := value.(float64); ok {
		return num, nil
	}
	return 0, fmt.Errorf("invalid auto number value type: %T", value)
}

// 解析公式值（只读字段）
func ParseFormulaValue(value any) (any, error) {
	// 公式字段可以返回string、number或boolean
	return value, nil
}

// 解析引用值（只读字段）
func ParseMagicLookUpValue(value any) ([]interface{}, error) {
	if lookupResult, ok := value.([]interface{}); ok {
		return lookupResult, nil
	}
	return nil, fmt.Errorf("invalid magic lookup value type: %T", value)
}

// --------------------------------------------------
type CellOption string

func (opt *CellOption) GetKey() string {
	return string(*opt)
}

func NewOptionValue(option *SelectFieldOption) *CellOption {
	return (*CellOption)(&option.Text)
}

func NewOptionValueByStr(str string) *CellOption {
	return (*CellOption)(&str)
}
func ParseSingleOptionValue(value any) (*CellOption, error) {
	if str, ok := value.(string); ok {
		option := CellOption(str)
		return &option, nil
	}
	return nil, fmt.Errorf("invalid option value type: %T", value)
}
