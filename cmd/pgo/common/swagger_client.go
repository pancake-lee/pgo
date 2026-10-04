package common

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/antihax/optional"
	"github.com/pancake-lee/pgo/cmd/pgo/swagger"
)

const clientTimeout = 5 * time.Second

// Client 封装 Swagger 客户端及各接口的请求构造与响应校验。
type Client struct {
	baseURL   string
	apiClient *swagger.APIClient
}

// NewClient 使用内置 HTTP 策略创建 Swagger 客户端。
func NewClient(baseURL string) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	parsedURL, err := url.Parse(baseURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("invalid base URL %q", baseURL)
	}
	configuration := swagger.NewConfiguration()
	configuration.BasePath = baseURL
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DisableKeepAlives = true
	configuration.HTTPClient = &http.Client{Timeout: clientTimeout, Transport: transport}
	return &Client{baseURL: baseURL, apiClient: swagger.NewAPIClient(configuration)}, nil
}

// BaseURL 返回 Swagger 客户端当前使用的服务根地址。
func (client *Client) BaseURL() string {
	return client.baseURL
}

// Login 构造登录请求并校验响应中的用户与令牌。
func (client *Client) Login(ctx context.Context, userName string) (swagger.ApiUserInfo, string, error) {
	request := swagger.ApiLoginRequest{UserName: userName}
	response, _, err := client.apiClient.UserApi.UserLogin(ctx, request)
	if err != nil {
		return swagger.ApiUserInfo{}, "", fmt.Errorf("login %s: %w", userName, err)
	}
	if response.User == nil || response.User.ID == 0 || response.User.UserName != userName || response.Token == "" {
		return swagger.ApiUserInfo{}, "", fmt.Errorf("login %s returned incomplete identity or token", userName)
	}
	return *response.User, response.Token, nil
}

// GetUserList 构造用户列表请求并返回用户列表与 HTTP 响应。
func (client *Client) GetUserList(ctx context.Context, userID int32, token string) ([]swagger.ApiUserInfo, *http.Response, error) {
	requestContext := ctx
	if token != "" {
		requestContext = context.WithValue(ctx, swagger.ContextAccessToken, token)
	}
	options := &swagger.UserCURDApiUserCURDGetUserListOpts{IDList: optional.NewInterface([]int32{userID})}
	response, httpResponse, err := client.apiClient.UserCURDApi.UserCURDGetUserList(requestContext, options)
	return response.UserList, httpResponse, err
}

// GetAllUserList 通过现有列表接口恢复测试批次中尚未持久化的用户。
func (client *Client) GetAllUserList(ctx context.Context, token string,
) ([]swagger.ApiUserInfo, error) {
	authContext := context.WithValue(ctx, swagger.ContextAccessToken, token)
	response, _, err := client.apiClient.UserCURDApi.UserCURDGetUserList(
		authContext, nil)
	return response.UserList, err
}

// DelUserByIDList 构造带鉴权的用户批量删除请求。
func (client *Client) DelUserByIDList(ctx context.Context, userID int32, token string) error {
	authContext := context.WithValue(ctx, swagger.ContextAccessToken, token)
	options := &swagger.UserCURDApiUserCURDDelUserByIDListOpts{IDList: optional.NewInterface([]int32{userID})}
	_, _, err := client.apiClient.UserCURDApi.UserCURDDelUserByIDList(authContext, options)
	return err
}
