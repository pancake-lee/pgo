package service

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-kratos/kratos/v2/transport"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/pancake-lee/pgo/internal/pkg/db/model"
)

// TestPermissionAggregation 保证重复权限、覆盖路径与空集合的语义一致。
func TestPermissionAggregation(t *testing.T) {
	permissionList := []*model.UserRolePermissionAssoc{
		{Action: "read", PathPattern: "/first"},
		{Action: "write", PathPattern: "/write"},
		{Action: "read", PathPattern: "/last"},
		{Action: "write", PathPattern: "/write"},
	}
	wantMap := map[string]string{"read": "/last", "write": "/write"}
	mergeList := []func([]*model.UserRolePermissionAssoc) map[string]string{
		mergePermissionsByList, mergePermissionsByMap,
	}
	for _, merge := range mergeList {
		if !maps.Equal(merge(permissionList), wantMap) {
			t.Fatal("duplicate action overwrite differs")
		}
		if len(merge(nil)) != 0 || merge(nil) == nil {
			t.Fatal("empty permissions must return an empty map")
		}
	}
}

// permissionTestTransport 为算法选择测试提供真实 HTTP 头语义。
type permissionTestTransport struct {
	requestHeader transport.Header
	replyHeader   transport.Header
}

// Kind 返回测试传输类型。
func (*permissionTestTransport) Kind() transport.Kind {
	return transport.KindHTTP
}

// Endpoint 返回测试端点。
func (*permissionTestTransport) Endpoint() string { return "http://test" }

// Operation 返回测试操作名。
func (*permissionTestTransport) Operation() string { return "permissions" }

// RequestHeader 返回请求头。
func (tr *permissionTestTransport) RequestHeader() transport.Header {
	return tr.requestHeader
}

// ReplyHeader 返回响应头。
func (tr *permissionTestTransport) ReplyHeader() transport.Header {
	return tr.replyHeader
}

// permissionTestHeader 将 HTTP 头适配到 Kratos 传输接口。
type permissionTestHeader struct{ http.Header }

// Keys 返回测试头名称列表。
func (header permissionTestHeader) Keys() []string {
	keyList := make([]string, 0, len(header.Header))
	for key := range header.Header {
		keyList = append(keyList, key)
	}
	return keyList
}

// TestPermissionExerciseSelection 验证默认、开关与服务回传模式。
func TestPermissionExerciseSelection(t *testing.T) {
	testList := []struct {
		mode    string
		enabled bool
		want    string
	}{
		{"", false, "map"}, {"map", false, "map"},
		{"list", false, ""}, {"list", true, "list"},
		{"unknown", true, ""},
	}
	for _, test := range testList {
		requestHeader := permissionTestHeader{http.Header{}}
		requestHeader.Set(permissionAggregationHeader, test.mode)
		replyHeader := permissionTestHeader{http.Header{}}
		tr := &permissionTestTransport{requestHeader, replyHeader}
		ctx := transport.NewServerContext(t.Context(), tr)
		server := &UserServer{PermissionExercise: test.enabled}
		actual, err := server.getPermissionAggregation(ctx)
		if test.want == "" {
			if err == nil {
				t.Fatalf("mode %q should be rejected", test.mode)
			}
			continue
		}
		if err != nil || actual != test.want ||
			replyHeader.Get(permissionAggregationHeader) != test.want {
			t.Fatalf("selection = %q, %v", actual, err)
		}
	}
	server := &UserServer{}
	mode, err := server.getPermissionAggregation(context.Background())
	if err != nil || mode != "map" {
		t.Fatal("direct calls must retain map behavior")
	}
}

// TestPermissionExerciseHTTPRejectsDisabled 验证旧路径会在数据库访问前拒绝练习。
func TestPermissionExerciseHTTPRejectsDisabled(t *testing.T) {
	server := kratoshttp.NewServer()
	userServer := &UserServer{}
	userServer.Reg(nil, server)
	request := httptest.NewRequest(http.MethodGet,
		"/user/permissions?userID=1&projectID=1", nil)
	request.Header.Set(permissionAggregationHeader, "list")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
}

// BenchmarkPermissionAggregation 对比相同权限输入下两种合并算法。
func BenchmarkPermissionAggregation(b *testing.B) {
	for _, roles := range []int{1, 10, 50} {
		permissionList := make([]*model.UserRolePermissionAssoc, 0, roles*500)
		for range roles {
			for index := range 500 {
				permissionList = append(permissionList,
					&model.UserRolePermissionAssoc{
						Action:      fmt.Sprintf("permission_%06d", index),
						PathPattern: fmt.Sprintf("/resources/%06d/*", index),
					})
			}
		}
		// merger 描述基准中接受相同权限输入的算法。
		type merger func([]*model.UserRolePermissionAssoc) map[string]string
		mergeMap := map[string]merger{
			"list": mergePermissionsByList,
			"map":  mergePermissionsByMap,
		}
		for name, merge := range mergeMap {
			b.Run(fmt.Sprintf("roles%d/%s", roles, name), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if len(merge(permissionList)) != 500 {
						b.Fatal("incorrect result")
					}
				}
			})
		}
	}
}
