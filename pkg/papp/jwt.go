package papp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pancake-lee/pgo/internal/pkg/perr"
	"github.com/pancake-lee/pgo/pkg/putil"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
	jwt5 "github.com/golang-jwt/jwt/v5"
)

type claims struct {
	// 其实标准的sub字段可以用来表达用户ID，这里只是示例，方便后续加入更多字段
	UserID int32 `json:"userId"`
	jwt5.RegisteredClaims
}

// Valid implements the jwt.v4 Claims interface
// Checks expiration and not-before times
func (c claims) Valid() error {
	now := time.Now()
	if c.ExpiresAt != nil && now.After(c.ExpiresAt.Time) {
		return fmt.Errorf("token expired")
	}
	if c.NotBefore != nil && now.Before(c.NotBefore.Time) {
		return fmt.Errorf("token not yet valid")
	}
	return nil
}

func GenToken(userId int32) (string, error) {
	tNow := time.Now()
	tokenClaims := claims{
		UserID: userId,
		RegisteredClaims: jwt5.RegisteredClaims{
			NotBefore: jwt5.NewNumericDate(tNow),
			IssuedAt:  jwt5.NewNumericDate(tNow),
			ExpiresAt: jwt5.NewNumericDate(tNow.Add(httpAuthExpire)),
			Issuer:    "pgo",
			Subject:   putil.Int32ToStr(userId),
		},
	}
	token := jwt5.NewWithClaims(jwt5.SigningMethodHS256, tokenClaims)
	ret, err := token.SignedString([]byte(httpAuthKey))
	if err != nil {
		return ret, perr.ErrTokenSign
	}
	return ret, nil
}

// --------------------------------------------------
// contextKey is the key for storing claims in context
type contextKey string

const claimsContextKey contextKey = "claims"

func GetTokenFromCtx(ctx context.Context) (*claims, error) {
	c, ok := ctx.Value(claimsContextKey).(*claims)
	if !ok {
		return nil, fmt.Errorf("auth failed")
	}
	return c, nil
}

func ParseToken(tokenString string) (*claims, error) {
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	token, err := jwt5.ParseWithClaims(tokenString, &claims{},
		func(token *jwt5.Token) (any, error) {
			return []byte(httpAuthKey), nil
		},
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
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
// 利用kratos的selector和jwt组件实现（已废弃，改用authMiddleware2）
/*
func authMiddleware() middleware.Middleware {
	return selector.
		Server(jwt.Server(
			func(token *jwt5.Token) (interface{}, error) {
				return []byte(httpAuthKey), nil
			},
			jwt.WithSigningMethod(jwt5.SigningMethodHS256),
			jwt.WithClaims(func() jwt5.Claims {
				return &jwt5.MapClaims{}
			}),
		)).
		Match(func(ctx context.Context, operation string) bool {
			return whiteList[operation]
		}).
		Build()
}
*/

// --------------------------------------------------
// 自定义中间件的方式实现
func authMiddleware2() middleware.Middleware {
	return func(nextHandler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			// https://go-kratos.dev/docs/component/transport/http#middleware-%E4%B8%AD%E5%A4%84%E7%90%86-http-%E8%AF%B7%E6%B1%82
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return nil, fmt.Errorf("no auth")
			}
			ht, ok := tr.(*http.Transport)
			if !ok {
				return nil, fmt.Errorf("no auth")
			}

			// 检查是否为排除路径
			if whiteList[ht.Request().URL.Path] {
				return nextHandler(ctx, req)
			}

			token := ht.Request().Header.Get("Authorization")
			if token == "" {
				return nil, fmt.Errorf("no auth")
			}

			claims, err := ParseToken(token)
			if err != nil {
				return nil, fmt.Errorf("unauthorized: %v", err)
			}

			// ParseToken里的ParseWithClaims已经做了过期检查
			// if time.Now().After(claims.ExpiresAt.Time) {
			// 	return nil, fmt.Errorf("expired")
			// }

			ctx = context.WithValue(ctx, claimsContextKey, claims)

			return nextHandler(ctx, req)
		}
	}
}
