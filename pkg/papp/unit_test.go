package papp

import (
	"testing"

	"github.com/pancake-lee/pgo/pkg/plogger"
)

func TestJwt(t *testing.T) {
	tokenStr, err := GenToken(1)
	if err != nil {
		t.Fatal(err)
	}
	plogger.Debugf("token: %s", tokenStr)

	token, err := ParseToken(tokenStr)
	if err != nil {
		t.Fatal(err)
	}
	plogger.Debugf("userId: %d", token.UserID)

	if token.UserID != 1 {
		plogger.Fatalf("expected userId 1, got %d", token.UserID)
	}
}
