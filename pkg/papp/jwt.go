package papp

import (
	"context"
	"strings"
	"time"

	"github.com/pancake-lee/pgo/api"
	"github.com/pancake-lee/pgo/internal/pkg/perr"
	"github.com/pancake-lee/pgo/pkg/putil"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
	jwt4 "github.com/golang-jwt/jwt/v4"
)

type claims struct {
	// 其实标准的sub字段可以用来表达用户ID，这里只是示例，方便后续加入更多字段
	UserID int32 `json:"userId"`
	jwt4.StandardClaims
}

func GenToken(userId int32) (string, error) {
	tNow := time.Now()
	tokenClaims := claims{
		UserID: userId,
		StandardClaims: jwt4.StandardClaims{
			NotBefore: tNow.Unix(),
			IssuedAt:  tNow.Unix(),
			ExpiresAt: tNow.Add(httpAuthExpire).Unix(),
			Issuer:    "pgo",
			Subject:   putil.Int32ToStr(userId),
		},
	}
	token := jwt4.NewWithClaims(jwt4.SigningMethodHS256, tokenClaims)
	ret, err := token.SignedString([]byte(httpAuthKey))
	if err != nil {
		return ret, perr.ErrTokenSign
	}
	return ret, nil
}

// --------------------------------------------------
func GetTokenFromCtx(ctx context.Context) (*claims, error) {
	token, ok := jwt.FromContext(ctx)
	if !ok {
		return nil, api.ErrorUnauthorized("auth failed")
	}
	t, ok := token.(*claims)
	if !ok {
		return nil, api.ErrorUnauthorized("token format invalid")
	}
	return t, nil
}

func ParseToken(tokenString string) (*claims, error) {
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	token, err := jwt4.ParseWithClaims(tokenString, &claims{},
		func(token *jwt4.Token) (any, error) {
			return []byte(httpAuthKey), nil
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

// --------------------------------------------------
var httpAuthKey string = ""

func SetHTTPAuthKey(key string) {
	httpAuthKey = key
}

var httpAuthExpire time.Duration = 24 * time.Hour

func SetHTTPAuthExpire(expire time.Duration) {
	httpAuthExpire = expire
}

var whiteList = make(map[string]bool)

func AddWhiteList(paths ...string) {
	for _, p := range paths {
		whiteList[p] = true
	}
}

// --------------------------------------------------
// 利用kratos的selector和jwt组件实现
func authMiddleware() middleware.Middleware {
	return selector.
		Server(jwt.Server(
			func(token *jwt4.Token) (interface{}, error) {
				return []byte(httpAuthKey), nil
			},
			jwt.WithSigningMethod(jwt4.SigningMethodHS256),
			jwt.WithClaims(func() jwt4.Claims {
				return &jwt4.MapClaims{}
			}),
		)).
		Match(func(ctx context.Context, operation string) bool {
			return whiteList[operation]
		}).
		Build()
}

// --------------------------------------------------
// 自定义中间件的方式实现
func authMiddleware2() middleware.Middleware {
	return func(nextHandler middleware.Handler) middleware.Handler {
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
			if whiteList[ht.Request().URL.Path] {
				return nextHandler(ctx, req)
			}

			token := ht.Request().Header.Get("Authorization")
			if token == "" {
				return nil, api.ErrorUnauthorized("no auth")
			}

			claims, err := ParseToken(token)
			if err != nil {
				return nil, api.ErrorUnauthorized(err.Error())
			}

			// ParseToken里的ParseWithClaims已经做了过期检查
			// if time.Now().After(claims.ExpiresAt.Time) {
			// 	return nil, api.ErrorUnauthorized("expired")
			// }

			ctx = jwt.NewContext(ctx, claims)

			return nextHandler(ctx, req)
		}
	}
}
