package conf

type conf struct {
	TokenExpire int    `default:"24"`                               //hours
	TokenSK     string `default:"12345678123456781234567812345678"` //secret key
}

var UserSvcConf conf
