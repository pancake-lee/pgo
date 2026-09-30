package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

func TestUserServiceRoot(t *testing.T) {
	server := kratoshttp.NewServer()
	var userServer UserServer
	userServer.Reg(nil, server)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if body := response.Body.String(); body != "hello, this is userService" {
		t.Fatalf("body = %q, want hello message", body)
	}
}
