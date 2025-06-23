package app

import (
	"testing"

	"github.com/pancake-lee/pgo/pkg/logger"
)

func TestJwt(t *testing.T) {
	tokenStr, err := GenToken(1)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("token: %s", tokenStr)

	token, err := ParseToken(tokenStr)
	if err != nil {
		t.Fatal(err)
	}
	logger.Debugf("userId: %d", token.UserID)

	if token.UserID != 1 {
		logger.Fatalf("expected userId 1, got %d", token.UserID)
	}
}
