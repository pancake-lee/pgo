package permissions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// apiClient 复用 HTTP 连接并调用现有业务接口。
type apiClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// newAPIClient 创建权限场景的 HTTP 客户端。
func newAPIClient(baseURL, token string) *apiClient {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = 16
	return &apiClient{
		baseURL: strings.TrimRight(baseURL, "/"), token: token,
		httpClient: &http.Client{Timeout: 5 * time.Second, Transport: transport},
	}
}

// request 校验 HTTP 状态并解析业务响应，错误信息不包含令牌。
func (client *apiClient) request(ctx context.Context, method, path string,
	body any, response any,
) error {
	var reader io.Reader
	if body != nil {
		content, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(content)
	}
	req, err := http.NewRequestWithContext(ctx, method, client.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+client.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: HTTP %d", method, path, resp.StatusCode)
	}
	if response == nil {
		_, err = io.Copy(io.Discard, resp.Body)
		return err
	}
	return json.NewDecoder(resp.Body).Decode(response)
}

// add 通过创建接口返回批次记录主键。
func (client *apiClient) add(ctx context.Context, kind string,
	fields map[string]any,
) (int32, error) {
	var response map[string]json.RawMessage
	err := client.request(ctx, http.MethodPost, "/"+kind,
		map[string]any{envelope(kind): fields}, &response)
	if err != nil {
		return 0, err
	}
	var record struct{ ID int32 }
	content, found := response[envelope(kind)]
	if !found {
		content = response[strings.ReplaceAll(kind, "-", "_")]
	}
	err = json.Unmarshal(content, &record)
	if err != nil {
		return 0, err
	}
	if record.ID <= 0 {
		return 0, fmt.Errorf("%s returned no record ID", kind)
	}
	return record.ID, nil
}

// envelope 将资源名称转换为 Proto JSON 使用的对象名称。
func envelope(kind string) string {
	partList := strings.Split(kind, "-")
	for index := 1; index < len(partList); index++ {
		partList[index] = strings.ToUpper(partList[index][:1]) + partList[index][1:]
	}
	return strings.Join(partList, "")
}

// permissionPath 构造项目、用户组与权限版本可辨认的等长路径。
func permissionPath(project int, group string, action, version int) string {
	return fmt.Sprintf("/project/%02d/%s/action/%03d/v%d/*",
		project, group[:1], action, version)
}

// getPermissions 使用当前用户令牌查询指定项目权限。
func (client *apiClient) getPermissions(ctx context.Context, user User,
	projectID int32,
) (map[string]string, error) {
	query := url.Values{}
	query.Set("userID", strconv.Itoa(int(user.ID)))
	query.Set("projectID", strconv.Itoa(int(projectID)))
	var response struct{ ActionToPathPattern map[string]string }
	userClient := *client
	userClient.token = user.Token
	err := userClient.request(ctx, http.MethodGet,
		"/user/permissions?"+query.Encode(), nil, &response)
	return response.ActionToPathPattern, err
}

// updatePermission 修改路径并确认服务返回的是本次提交值。
func (client *apiClient) updatePermission(ctx context.Context, id int32,
	path string,
) error {
	var response struct {
		Permission struct {
			ID          int32
			PathPattern string
		} `json:"userRolePermissionAssoc"`
	}
	body := map[string]any{"userRolePermissionAssoc": map[string]any{
		"ID": id, "pathPattern": path,
	}}
	err := client.request(ctx, http.MethodPatch,
		"/user-role-permission-assoc", body, &response)
	if err != nil {
		return err
	}
	if response.Permission.ID != id || response.Permission.PathPattern != path {
		return fmt.Errorf("permission update %d returned a different value", id)
	}
	return nil
}

// deleteIDs 使用重复查询参数删除确切主键，避免 SDK 数组拼接问题。
func (client *apiClient) deleteIDs(ctx context.Context, kind string,
	idList []int32,
) error {
	query := url.Values{}
	for _, id := range idList {
		query.Add("IDList", strconv.Itoa(int(id)))
	}
	return client.request(ctx, http.MethodDelete,
		"/"+kind+"?"+query.Encode(), nil, nil)
}
