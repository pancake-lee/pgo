package papp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

func TestAuthMiddlewareRejectsInvalidTokens(t *testing.T) {
	server := kratoshttp.NewServer(
		kratoshttp.Middleware(authMiddleware2()),
	)
	called := false
	server.Route("/").GET("/user", func(ctx kratoshttp.Context) error {
		handler := ctx.Middleware(
			func(context.Context, any) (any, error) {
				called = true
				return nil, nil
			},
		)
		_, err := handler(ctx, nil)
		return err
	})

	for _, token := range []string{"", "invalid-token"} {
		t.Run("token="+token, func(t *testing.T) {
			called = false
			request := httptest.NewRequest(http.MethodGet, "/user", nil)
			request.Header.Set("Authorization", token)
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if called || response.Code < http.StatusBadRequest {
				t.Fatalf("invalid token accepted: status %d", response.Code)
			}
		})
	}
}
