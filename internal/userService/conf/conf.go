package conf

type conf struct {
	TokenExpire int    `default:"24"` //hours
	TokenSK     string `default:"10"` //secret key
}

var UserSvcConf conf
