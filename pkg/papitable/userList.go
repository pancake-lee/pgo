package papitable

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

// 获取小组列表和成员的接口都404了，可能是开源版本不给用吧。

// User 表示成员信息
type User struct {
	UnitId string `json:"unitId"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Mobile struct {
		Number   string `json:"number"`
		AreaCode string `json:"areaCode"`
	} `json:"mobile"`
	Avatar string `json:"avatar"`
	Status int    `json:"status"` // 0:未加入空间站, 1:已加入空间站
	Type   string `json:"type"`   // PrimaryAdmin, SubAdmin, Member
	Teams  []struct {
		UnitId       string `json:"unitId"`
		Name         string `json:"name"`
		Sequence     int    `json:"sequence"`
		ParentUnitId string `json:"parentUnitId"`
	} `json:"teams"`
	Roles []struct {
		UnitId   string `json:"unitId"`
		Name     string `json:"name"`
		Sequence int    `json:"sequence"`
	} `json:"roles"`
}

// Team 表示小组信息
type Team struct {
	UnitId       string `json:"unitId"`
	Name         string `json:"name"`
	Sequence     int    `json:"sequence"`
	ParentUnitId string `json:"parentUnitId"`
	Roles        []struct {
		UnitId   string `json:"unitId"`
		Name     string `json:"name"`
		Sequence int    `json:"sequence"`
	} `json:"roles"`
}

// GetTeamListResponse 获取小组列表响应
type GetTeamListResponse struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		PageNum  int     `json:"pageNum"`
		PageSize int     `json:"pageSize"`
		Total    int     `json:"total"`
		Teams    []*Team `json:"teams"`
	} `json:"data"`
}

// GetTeamMembersResponse 获取小组成员响应
type GetTeamMembersResponse struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		PageNum  int     `json:"pageNum"`
		PageSize int     `json:"pageSize"`
		Total    int     `json:"total"`
		Members  []*User `json:"members"`
	} `json:"data"`
}

// GetUserResponse 获取单个用户响应
type GetUserResponse struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    User   `json:"data"`
}

// GetUserList 递归获取所有成员
func GetUserList(spaceId string) ([]*User, error) {
	if spaceId == "" {
		return nil, plogger.LogErr(fmt.Errorf("spaceId is required"))
	}

	allUsers := make(map[string]*User) // 使用map去重，key为unitId

	// 从根节点"0"开始递归获取所有小组和成员
	err := getAllUsersRecursive(spaceId, "0", allUsers)
	if err != nil {
		return nil, err
	}

	// 转换map为slice
	var users []*User
	for _, user := range allUsers {
		users = append(users, user)
	}

	plogger.Debugf("获取到总计 %d 个成员", len(users))
	return users, nil
}

// getAllUsersRecursive 递归获取所有用户
func getAllUsersRecursive(spaceId, unitId string, allUsers map[string]*User) error {
	// 1. 获取当前小组的子小组列表
	teams, err := getTeamChildren(spaceId, unitId)
	if err != nil {
		return err
	}

	// 2. 获取当前小组的成员
	unitId = "cd6a59f835704b59997591137295d91a"
	members, err := getTeamMembers(spaceId, unitId)
	if err != nil {
		return err
	}

	// 3. 将成员添加到总列表中（去重）
	for _, member := range members {
		allUsers[member.UnitId] = member
	}

	// 4. 递归处理每个子小组
	for _, team := range teams {
		err = getAllUsersRecursive(spaceId, team.UnitId, allUsers)
		if err != nil {
			return err
		}
	}

	return nil
}

// getTeamChildren 获取小组的子小组列表
func getTeamChildren(spaceId, unitId string) ([]*Team, error) {
	var allTeams []*Team
	pageNum := 1
	pageSize := 100

	for {
		url := fmt.Sprintf("%s/fusion/v1/spaces/%s/teams/%s/children", g_baseUrl, spaceId, unitId)

		params := map[string]string{
			"pageNum":  fmt.Sprintf("%d", pageNum),
			"pageSize": fmt.Sprintf("%d", pageSize),
		}

		req, err := putil.NewHttpRequestJson(http.MethodGet, url,
			getTokenHeader(), params, nil)
		if err != nil {
			return nil, plogger.LogErr(err)
		}

		resp, err := putil.HttpDo(req)
		if err != nil {
			return nil, plogger.LogErr(err)
		}

		var respData GetTeamListResponse
		err = json.Unmarshal(resp, &respData)
		if err != nil {
			return nil, plogger.LogErr(err)
		}

		if !respData.Success {
			return nil, plogger.LogErr(fmt.Errorf("get team children failed: code=%d, message=%s", respData.Code, respData.Message))
		}

		allTeams = append(allTeams, respData.Data.Teams...)

		// 检查是否还有更多页
		if pageNum*pageSize >= respData.Data.Total {
			break
		}
		pageNum++
	}

	plogger.Debugf("小组 %s 有 %d 个子小组", unitId, len(allTeams))
	return allTeams, nil
}

// getTeamMembers 获取小组的成员列表
func getTeamMembers(spaceId, unitId string) ([]*User, error) {
	var allMembers []*User
	pageNum := 1
	pageSize := 100

	for {
		url := fmt.Sprintf("%s/fusion/v1/spaces/%s/teams/%s/members", g_baseUrl, spaceId, unitId)

		params := map[string]string{
			"pageNum":       fmt.Sprintf("%d", pageNum),
			"pageSize":      fmt.Sprintf("%d", pageSize),
			"sensitiveData": "true", // 获取敏感数据
		}

		req, err := putil.NewHttpRequestJson(http.MethodGet, url,
			getTokenHeader(), params, nil)
		if err != nil {
			return nil, plogger.LogErr(err)
		}

		resp, err := putil.HttpDo(req)
		if err != nil {
			return nil, plogger.LogErr(err)
		}

		var respData GetTeamMembersResponse
		err = json.Unmarshal(resp, &respData)
		if err != nil {
			return nil, plogger.LogErr(err)
		}

		if !respData.Success {
			return nil, plogger.LogErr(fmt.Errorf("get team members failed: code=%d, message=%s", respData.Code, respData.Message))
		}

		allMembers = append(allMembers, respData.Data.Members...)

		// 检查是否还有更多页
		if pageNum*pageSize >= respData.Data.Total {
			break
		}
		pageNum++
	}

	plogger.Debugf("小组 %s 有 %d 个成员", unitId, len(allMembers))
	return allMembers, nil
}

// GetUser 根据unitId获取单个用户信息
func GetUser(spaceId, unitId string) (*User, error) {
	if spaceId == "" {
		return nil, plogger.LogErr(fmt.Errorf("spaceId is required"))
	}
	if unitId == "" {
		return nil, plogger.LogErr(fmt.Errorf("unitId is required"))
	}

	url := fmt.Sprintf("%s/fusion/v1/spaces/%s/members/%s", g_baseUrl, spaceId, unitId)

	params := map[string]string{
		"sensitiveData": "true",
	}

	req, err := putil.NewHttpRequestJson(http.MethodGet, url,
		getTokenHeader(), params, nil)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	resp, err := putil.HttpDo(req)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	var respData GetUserResponse
	err = json.Unmarshal(resp, &respData)
	if err != nil {
		return nil, plogger.LogErr(err)
	}

	if !respData.Success {
		return nil, plogger.LogErr(fmt.Errorf("get user failed: code=%d, message=%s", respData.Code, respData.Message))
	}

	return &respData.Data, nil
}
