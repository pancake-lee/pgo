package app

import (
	"pgo/internal/pkg/perr"
	"pgo/internal/userService/conf"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	UserID int32 `json:"userId"` // 其实标准的sub字段可以用来表达用户ID
	jwt.RegisteredClaims
}

func GenToken(userId int32) (string, error) {
	tNow := time.Now()
	tokenClaims := Claims{
		UserID: userId,
		RegisteredClaims: jwt.RegisteredClaims{
			NotBefore: jwt.NewNumericDate(tNow),
			IssuedAt:  jwt.NewNumericDate(tNow),
			ExpiresAt: jwt.NewNumericDate(
				tNow.Add(time.Duration(conf.UserSvcConf.TokenExpire) * time.Hour)),
			Issuer: "pgo",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims)
	ret, err := token.SignedString([]byte(conf.UserSvcConf.TokenSK))
	if err != nil {
		return ret, perr.ErrTokenSign
	}
	return ret, nil
}

// TODO : 解析/校验token
