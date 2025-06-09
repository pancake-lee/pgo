package app

import (
	"context"
	"pgo/api"
	"pgo/internal/pkg/perr"
	"pgo/internal/userService/conf"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/golang-jwt/jwt/v4"
)

type claims struct {
	UserID int32 `json:"userId"` // 其实标准的sub字段可以用来表达用户ID
	jwt.RegisteredClaims
}

func GenToken(userId int32) (string, error) {
	tNow := time.Now()
	tokenClaims := claims{
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

func ParseToken(tokenString string) (*claims, error) {
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	token, err := jwt.ParseWithClaims(tokenString, &claims{},
		func(token *jwt.Token) (any, error) {
			return []byte(conf.UserSvcConf.TokenSK), nil
		},
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, api.ErrorUnauthorized("token is invalid")
	}

	claims, ok := token.Claims.(*claims)
	if !ok {
		return nil, perr.ErrTokenFormatInvalid
	}
	return claims, nil
}

func authMiddleware(excludePaths ...string) middleware.Middleware {
	excludeMap := make(map[string]bool)
	for _, path := range excludePaths {
		excludeMap[path] = true
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			// https://go-kratos.dev/docs/component/transport/http#middleware-%E4%B8%AD%E5%A4%84%E7%90%86-http-%E8%AF%B7%E6%B1%82
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return nil, api.ErrorUnauthorized("no auth")
			}
			ht, ok := tr.(*http.Transport)
			if !ok {
				return nil, api.ErrorUnauthorized("no auth")
			}

			// 检查是否为排除路径
			if excludeMap[ht.Request().URL.Path] {
				return handler(ctx, req)
			}

			token := ht.Request().Header.Get("Authorization")
			if token == "" {
				return nil, api.ErrorUnauthorized("no auth")
			}

			claims, err := ParseToken(token)
			if err != nil {
				return nil, api.ErrorUnauthorized(err.Error())
			}

			if time.Now().After(claims.ExpiresAt.Time) {
				return nil, api.ErrorUnauthorized("expired")
			}

			// 将用户信息添加到context中
			ctx = CtxSetUserId(ctx, claims.UserID)

			return handler(ctx, req)
		}
	}
}
