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
// 成员类型单元格值，因为用户列表API调不通，所以暂时用不上
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
	if members, ok := value.([]any); ok {
		result := make([]CellUserValue, 0, len(members))
		for _, mem := range members {
			if memMap, ok := mem.(map[string]any); ok {
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

// --------------------------------------------------
// Formula（智能公式）字段值处理，不能直接写入值，只能读取计算结果，类型由列配置的valueType决定
// NewFormulaCol直接写入了string类型，所以这里直接解析成string
func ParseFormulaValue(value any) (string, error) {
	if str, ok := value.(string); ok {
		return str, nil
	}
	return "", fmt.Errorf("invalid formula value type: %T", value)
}

// --------------------------------------------------
// OneWayLink（单向关联）字段值处理
func NewOneWayLinkValue(recordIds []string) []string {
	return recordIds
}

func ParseOneWayLinkValue(value any) ([]string, error) {
	if recordIds, ok := value.([]string); ok {
		return recordIds, nil
	}
	return nil, fmt.Errorf("invalid link value type: %T", value)
}

// --------------------------------------------------
// MagicLookUp（神奇引用）字段值处理，不能直接写入值，只能读取计算结果，类型由列配置的RollupFunction决定
func ParseMagicLookUpValue(value any) (any, error) {
	// 暂不处理，解析起来比较麻烦，需要根据该列引用目标列的配置来解析
	return value, nil
}
