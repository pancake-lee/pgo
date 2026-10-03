package conf

// conf 保存用户服务的鉴权、练习开关与多维表格配置。
type conf struct {
	PermissionExercise bool   `default:"false"`
	TokenExpire        int    `default:"24"`                               //hours
	TokenSK            string `default:"12345678123456781234567812345678"` //secret key

	APITable struct {
		UserSheetID    string
		ProjectSheetID string
	}
}

// UserSvcConf 保存启动时读取的用户服务配置。
var UserSvcConf conf
