package papp

import (
	"context"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/pancake-lee/pgo/pkg/plogger"
)

func TestJwt(t *testing.T) {

	SetHTTPAuthKey("testkey")
	SetHTTPAuthExpire(2 * time.Hour)
	var userId int32 = 1

	tokenStr, err := GenToken(userId)
	if err != nil {
		t.Fatal(err)
	}
	// plogger.Debugf("token: %s", tokenStr)

	token, err := ParseToken(tokenStr)
	if err != nil {
		t.Fatal(err)
	}

	if token.UserID != userId {
		plogger.Fatalf("expected userId 1, got %d", token.UserID)
		t.FailNow()
	}

	ctx := context.Background()

	// --------------------------------------------------
	ctx = jwt.NewContext(ctx, token)
	token2, err := GetTokenFromCtx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if token2.UserID != userId {
		plogger.Fatalf("expected userId 1, got %d", token2.UserID)
		t.FailNow()
	}

	// --------------------------------------------------
	ctx = SetUserIdToCtx(ctx, token.UserID)
	uid, ok := GetUserIdFromCtx(ctx)
	if !ok || uid != userId {
		plogger.Fatalf("expected userId 1, got %d", uid)
		t.FailNow()
	}
}
