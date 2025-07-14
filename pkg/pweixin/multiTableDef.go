package pweixin

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

// 预定义的颜色选项
var (
	ColorRed    = OptStyle_Red
	ColorOrange = OptStyle_Orange
	ColorYellow = OptStyle_Yellow
	ColorBlue   = OptStyle_Blue
	ColorGreen  = OptStyle_Green
	ColorPurple = OptStyle_Purple
	ColorGray   = OptStyle_Gray
	ColorPink   = OptStyle_Pink
)
