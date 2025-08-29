package papitable

// --------------------------------------------------
// 字段类型
// --------------------------------------------------
// 字段类型常量
type FieldType string

const (
	// FIELD_TYPE_SINGLE_TEXT FieldType = "SingleText" //单行文本
	// FIELD_TYPE_TEXT               FieldType = "Text"             //多行文本
	FIELD_TYPE_TEXT FieldType = "SingleText" // 用一个就好了，文档有两个类型

	FIELD_TYPE_SINGLE_SELECT      FieldType = "SingleSelect"     //单选
	FIELD_TYPE_MULTI_SELECT       FieldType = "MultiSelect"      //多选
	FIELD_TYPE_NUMBER             FieldType = "Number"           //数字
	FIELD_TYPE_CURRENCY           FieldType = "Currency"         //货币
	FIELD_TYPE_PERCENT            FieldType = "Percent"          //百分比
	FIELD_TYPE_DATE_TIME          FieldType = "DateTime"         //日期
	FIELD_TYPE_ATTACHMENT         FieldType = "Attachment"       //附件
	FIELD_TYPE_MEMBER             FieldType = "Member"           //成员
	FIELD_TYPE_CHECKBOX           FieldType = "Checkbox"         //勾选
	FIELD_TYPE_RATING             FieldType = "Rating"           //评分
	FIELD_TYPE_URL                FieldType = "URL"              //网址
	FIELD_TYPE_PHONE              FieldType = "Phone"            //电话
	FIELD_TYPE_EMAIL              FieldType = "Email"            //邮箱
	FIELD_TYPE_WORK_DOC           FieldType = "WorkDoc"          //轻文档
	FIELD_TYPE_ONE_WAY_LINK       FieldType = "OneWayLink"       //单向关联
	FIELD_TYPE_TWO_WAY_LINK       FieldType = "TwoWayLink"       //双向关联
	FIELD_TYPE_MAGIC_LOOKUP       FieldType = "MagicLookUp"      //神奇引用
	FIELD_TYPE_FORMULA            FieldType = "Formula"          //智能公式
	FIELD_TYPE_AUTO_NUMBER        FieldType = "AutoNumber"       //自增数字
	FIELD_TYPE_CREATED_TIME       FieldType = "CreatedTime"      //创建时间
	FIELD_TYPE_LAST_MODIFIED_TIME FieldType = "LastModifiedTime" //修改时间
	FIELD_TYPE_CREATED_BY         FieldType = "CreatedBy"        //创建人
	FIELD_TYPE_LAST_MODIFIED_BY   FieldType = "LastModifiedBy"   //更新人
	FIELD_TYPE_BUTTON             FieldType = "Button"           //按钮
)

// --------------------------------------------------
// 颜色
// 文档有问题，这里只需要string

// type SelectFieldOptionColor struct {
// 	Name  string `json:"name"`
// 	Value string `json:"value"`
// }

// // 预定义的颜色选项
// var (
// 	ColorRed    = &SelectFieldOptionColor{Name: "red_4", Value: "#F19F9C"}
// 	ColorOrange = &SelectFieldOptionColor{Name: "tangerine_4", Value: "#FFBD80"}
// 	ColorYellow = &SelectFieldOptionColor{Name: "yellow_4", Value: "#FFEA9D"}
// 	ColorBlue   = &SelectFieldOptionColor{Name: "blue_4", Value: "#AAE6FF"}
// 	ColorGreen  = &SelectFieldOptionColor{Name: "green_4", Value: "#A9E28D"}
// 	ColorPurple = &SelectFieldOptionColor{Name: "deepPurple_4", Value: "#BDB3F7"}
// 	ColorGray   = &SelectFieldOptionColor{Name: "blue_0", Value: "#EEFAFF"}
// 	ColorPink   = &SelectFieldOptionColor{Name: "pink_4", Value: "#FFB8C5"}
// )

type SelectFieldOptionColor string

// 预定义的颜色选项
var (
	ColorRed    SelectFieldOptionColor = "red_4"
	ColorOrange SelectFieldOptionColor = "tangerine_4"
	ColorYellow SelectFieldOptionColor = "yellow_4"
	ColorBlue   SelectFieldOptionColor = "blue_4"
	ColorGreen  SelectFieldOptionColor = "green_4"
	ColorPurple SelectFieldOptionColor = "deepPurple_4"
	ColorGray   SelectFieldOptionColor = "blue_0"
	ColorPink   SelectFieldOptionColor = "pink_4"
)

var AllColors = []SelectFieldOptionColor{
	ColorRed,
	ColorOrange,
	ColorYellow,
	ColorBlue,
	ColorGreen,
	ColorPurple,
	ColorGray,
	ColorPink,
}
