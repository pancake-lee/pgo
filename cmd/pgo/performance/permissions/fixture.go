package permissions

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/antihax/optional"
	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/pancake-lee/pgo/cmd/pgo/common"
	"github.com/pancake-lee/pgo/cmd/pgo/swagger"
	"github.com/pancake-lee/pgo/pkg/papp"
)

// manifest 记录权限练习拥有的数据及中断恢复所需元数据。
type manifest struct {
	Version          int        `json:"version"`
	BatchID          string     `json:"batchID"`
	BaseURL          string     `json:"baseURL"`
	Options          options    `json:"options"`
	CreatedAt        time.Time  `json:"createdAt"`
	CleanedAt        *time.Time `json:"cleanedAt,omitempty"`
	UserName         string     `json:"userName"`
	UserID           int32      `json:"userID"`
	Token            string     `json:"token"`
	ProjectIDList    []int32    `json:"projectIDs"`
	RoleIDList       []int32    `json:"roleIDs"`
	AssocIDList      []int32    `json:"assocIDs"`
	PermissionIDList []int32    `json:"permissionIDs"`
	Incomplete       bool       `json:"incomplete"`
	path             string
}

// newManifest 在创建任何数据前持久化唯一批次身份。
func newManifest(baseURL string, options options, path string,
) (*manifest, error) {
	random := make([]byte, 8)
	_, err := rand.Read(random)
	if err != nil {
		return nil, err
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if !strings.Contains(baseURL, "://") {
		baseURL = "http://" + baseURL
	}
	batchID := hex.EncodeToString(random)
	manifest := &manifest{
		Version: 1, BatchID: batchID, BaseURL: baseURL, Options: options,
		CreatedAt: time.Now().UTC(), UserName: "perm_" + batchID,
		Incomplete: true, path: path,
	}
	return manifest, manifest.write()
}

// write 原子保存批次清单并限制文件访问权限。
func (manifest *manifest) write() error {
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	tempPath := manifest.path + ".tmp"
	err = os.WriteFile(tempPath, append(content, '\n'), 0o600)
	if err != nil {
		return err
	}
	return os.Rename(tempPath, manifest.path)
}

// readManifest 校验清单身份并读取中断后的批次。
func readManifest(path string) (*manifest, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	manifest := &manifest{path: path}
	err = json.Unmarshal(content, manifest)
	if err != nil {
		return nil, err
	}
	_, batchErr := hex.DecodeString(manifest.BatchID)
	if manifest.Version != 1 || len(manifest.BatchID) != 16 ||
		batchErr != nil || manifest.UserName != "perm_"+manifest.BatchID {
		return nil, errors.New("invalid permissions manifest identity")
	}
	err = manifest.Options.validate()
	if err != nil {
		return nil, err
	}
	_, err = common.NewClient(manifest.BaseURL)
	return manifest, err
}

// newAPIClient 使用固定请求超时和练习请求头构造既有 Swagger SDK。
func (manifest *manifest) newAPIClient(aggregation string,
) *swagger.APIClient {
	config := swagger.NewConfiguration()
	config.BasePath = manifest.BaseURL
	config.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	config.DefaultHeader["Authorization"] = manifest.Token
	config.DefaultHeader[aggregationHeader] = aggregation
	return swagger.NewAPIClient(config)
}

// login 获取批次用户身份及当前有效令牌。
func (manifest *manifest) login(ctx context.Context) error {
	client, err := common.NewClient(manifest.BaseURL)
	if err != nil {
		return err
	}
	user, token, err := client.Login(ctx, manifest.UserName)
	if err != nil {
		return err
	}
	if manifest.UserID != 0 && manifest.UserID != user.ID {
		return errors.New("batch user identity changed")
	}
	manifest.UserID = user.ID
	manifest.Token = token
	return manifest.write()
}

// prepare 通过既有 CRUD 接口准备两个隔离项目及重复角色权限。
func (manifest *manifest) prepare(ctx context.Context, logger klog.Logger,
) error {
	err := manifest.login(ctx)
	if err != nil {
		return err
	}
	client := manifest.newAPIClient("map")
	_, err = manifest.getPermissions(ctx, 0, manifest.Options.Aggregation)
	if err != nil {
		return err
	}
	for index := range 2 {
		request := swagger.ApiAddProjectRequest{
			Project: &swagger.ApiProjectInfo{
				ProjName:   fmt.Sprintf("perm_%s_%d", manifest.BatchID, index),
				CreateUser: manifest.UserID, UpdateUser: manifest.UserID,
			},
		}
		response, _, err := client.UserCURDApi.UserCURDAddProject(ctx, request)
		if err != nil {
			return err
		}
		if response.Project == nil || response.Project.ID <= 0 {
			return errors.New("project create returned no ID")
		}
		manifest.ProjectIDList = append(
			manifest.ProjectIDList, response.Project.ID,
		)
		err = manifest.write()
		if err != nil {
			return err
		}
	}
	for index := range manifest.Options.Roles + 1 {
		projectID := manifest.ProjectIDList[0]
		if index == manifest.Options.Roles {
			projectID = manifest.ProjectIDList[1]
		}
		request := swagger.ApiAddUserRoleRequest{
			UserRole: &swagger.ApiUserRoleInfo{
				ProjID:     projectID,
				RoleName:   fmt.Sprintf("perm_%s_%02d", manifest.BatchID, index),
				CreateUser: manifest.UserID, UpdateUser: manifest.UserID,
			},
		}
		response, _, err := client.UserCURDApi.UserCURDAddUserRole(ctx, request)
		if err != nil {
			return err
		}
		if response.UserRole == nil || response.UserRole.ID <= 0 {
			return errors.New("role create returned no ID")
		}
		roleID := response.UserRole.ID
		manifest.RoleIDList = append(manifest.RoleIDList, roleID)
		err = manifest.write()
		if err != nil {
			return err
		}
		assocRequest := swagger.ApiAddUserRoleAssocRequest{
			UserRoleAssoc: &swagger.ApiUserRoleAssocInfo{
				UserID: manifest.UserID, RoleID: roleID,
				CreateUser: manifest.UserID,
			},
		}
		assoc, _, err := client.UserCURDApi.UserCURDAddUserRoleAssoc(
			ctx, assocRequest,
		)
		if err != nil {
			return err
		}
		if assoc.UserRoleAssoc == nil || assoc.UserRoleAssoc.ID <= 0 {
			return errors.New("role association create returned no ID")
		}
		manifest.AssocIDList = append(
			manifest.AssocIDList, assoc.UserRoleAssoc.ID,
		)
		err = manifest.write()
		if err != nil {
			return err
		}
		permissionCount := manifest.Options.Permissions
		if index == manifest.Options.Roles {
			permissionCount = 1
		}
		for start := 0; start < permissionCount; start += 100 {
			end := min(start+100, permissionCount)
			jobList := make([]int, end-start)
			for job := range jobList {
				jobList[job] = start + job
			}
			resultList, runErr := papp.RunConcurrent(
				ctx, jobList,
				func(ctx context.Context, actionIndex int) (int32, error) {
					action := getAction(actionIndex)
					path := getPath(actionIndex)
					if index == manifest.Options.Roles {
						action, path = "foreign_project", "/foreign/project"
					}
					request := swagger.ApiAddUserRolePermissionAssocRequest{
						UserRolePermissionAssoc: &swagger.ApiUserRolePermissionAssocInfo{
							RoleID: roleID, Action: action, PathPattern: path,
							CreateUser: manifest.UserID,
						},
					}
					response, _, err := client.UserCURDApi.
						UserCURDAddUserRolePermissionAssoc(ctx, request)
					if err != nil {
						return 0, err
					}
					if response.UserRolePermissionAssoc == nil ||
						response.UserRolePermissionAssoc.ID <= 0 {
						return 0, errors.New("permission create returned no ID")
					}
					return response.UserRolePermissionAssoc.ID, nil
				},
			)
			errorList := []error{runErr}
			for _, result := range resultList {
				if result.Err == nil {
					manifest.PermissionIDList = append(
						manifest.PermissionIDList, result.Value,
					)
				} else {
					errorList = append(errorList, result.Err)
				}
			}
			errorList = append(errorList, manifest.write())
			err = errors.Join(errorList...)
			if err != nil {
				return err
			}
		}
		_ = logger.Log(klog.LevelInfo,
			"msg", "permission role prepared",
			"role", index+1, "total", manifest.Options.Roles+1,
			"permissionRows", len(manifest.PermissionIDList),
		)
	}
	manifest.Incomplete = false
	return manifest.write()
}

// getAction 返回固定长度的权限动作名。
func getAction(index int) string {
	return fmt.Sprintf("permission_%06d", index)
}

// getPath 返回固定长度的权限路径模式。
func getPath(index int) string {
	return fmt.Sprintf("/exercise/resources/%06d/*", index)
}

// getPermissions 请求指定项目并核对服务器实际执行的算法。
func (manifest *manifest) getPermissions(
	ctx context.Context, projectID int32, aggregation string,
) (map[string]string, error) {
	client := manifest.newAPIClient(aggregation)
	options := &swagger.UserApiUserGetUserPermissionsOpts{
		UserID:    optional.NewInt32(manifest.UserID),
		ProjectID: optional.NewInt32(projectID),
	}
	response, httpResponse, err := client.UserApi.UserGetUserPermissions(
		ctx, options,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"verify %s permissions (list requires PermissionExercise: true): %w",
			aggregation, err,
		)
	}
	if httpResponse == nil ||
		httpResponse.Header.Get(aggregationHeader) != aggregation {
		return nil, errors.New(
			"server did not confirm aggregation; deploy the new userService",
		)
	}
	return response.ActionToPathPattern, nil
}

// verify 检查完整结果、跨项目隔离及两个聚合算法的逐键一致性。
func (manifest *manifest) verify(ctx context.Context) error {
	modeList := []string{"map"}
	if manifest.Options.Aggregation == "list" {
		modeList = append(modeList, "list")
	}
	for _, mode := range modeList {
		permissionMap, err := manifest.getPermissions(
			ctx, manifest.ProjectIDList[0], mode,
		)
		if err != nil {
			return err
		}
		if len(permissionMap) != manifest.Options.Permissions {
			return fmt.Errorf("expected %d permissions, got %d",
				manifest.Options.Permissions, len(permissionMap))
		}
		for index := range manifest.Options.Permissions {
			if permissionMap[getAction(index)] != getPath(index) {
				return fmt.Errorf("permission content mismatch at %d", index)
			}
		}
		foreignMap, err := manifest.getPermissions(
			ctx, manifest.ProjectIDList[1], mode,
		)
		if err != nil {
			return err
		}
		if len(foreignMap) != 1 ||
			foreignMap["foreign_project"] != "/foreign/project" {
			return errors.New("project isolation failed")
		}
	}
	return nil
}

// writeTargets 输出带鉴权及算法选择的 Vegeta JSON 目标。
func (manifest *manifest) writeTargets(path string) error {
	query := url.Values{}
	query.Set("userID", fmt.Sprint(manifest.UserID))
	query.Set("projectID", fmt.Sprint(manifest.ProjectIDList[0]))
	target := struct {
		Method string              `json:"method"`
		URL    string              `json:"url"`
		Header map[string][]string `json:"header"`
	}{
		Method: http.MethodGet,
		URL:    manifest.BaseURL + "/user/permissions?" + query.Encode(),
		Header: map[string][]string{
			"Authorization":   {manifest.Token},
			aggregationHeader: {manifest.Options.Aggregation},
		},
	}
	content, err := json.Marshal(target)
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(content, '\n'), 0o600)
}

// recoverIDs 从公开列表接口找回中断或响应丢失的本批次数据。
func (manifest *manifest) recoverIDs(ctx context.Context,
	client *swagger.APIClient,
) error {
	projects, _, err := client.UserCURDApi.UserCURDGetProjectList(ctx, nil)
	if err != nil {
		return err
	}
	for _, project := range projects.ProjectList {
		if strings.HasPrefix(project.ProjName, "perm_"+manifest.BatchID+"_") &&
			project.CreateUser == manifest.UserID &&
			!slices.Contains(manifest.ProjectIDList, project.ID) {
			manifest.ProjectIDList = append(manifest.ProjectIDList, project.ID)
		}
	}
	roles, _, err := client.UserCURDApi.UserCURDGetUserRoleList(ctx, nil)
	if err != nil {
		return err
	}
	for _, role := range roles.UserRoleList {
		if slices.Contains(manifest.ProjectIDList, role.ProjID) &&
			strings.HasPrefix(role.RoleName, "perm_"+manifest.BatchID+"_") &&
			role.CreateUser == manifest.UserID &&
			!slices.Contains(manifest.RoleIDList, role.ID) {
			manifest.RoleIDList = append(manifest.RoleIDList, role.ID)
		}
	}
	if len(manifest.RoleIDList) == 0 {
		return manifest.write()
	}
	assocOptions := &swagger.UserCURDApiUserCURDGetUserRoleAssocListOpts{
		UserIDList: optional.NewInterface([]int32{manifest.UserID}),
	}
	assocs, _, err := client.UserCURDApi.UserCURDGetUserRoleAssocList(
		ctx, assocOptions,
	)
	if err != nil {
		return err
	}
	for _, assoc := range assocs.UserRoleAssocList {
		if assoc.UserID == manifest.UserID &&
			slices.Contains(manifest.RoleIDList, assoc.RoleID) &&
			!slices.Contains(manifest.AssocIDList, assoc.ID) {
			manifest.AssocIDList = append(manifest.AssocIDList, assoc.ID)
		}
	}
	permissions, _, err := client.UserCURDApi.
		UserCURDGetUserRolePermissionAssocList(ctx, nil)
	if err != nil {
		return err
	}
	for _, permission := range permissions.UserRolePermissionAssocList {
		if slices.Contains(manifest.RoleIDList, permission.RoleID) &&
			permission.CreateUser == manifest.UserID &&
			!slices.Contains(manifest.PermissionIDList, permission.ID) {
			manifest.PermissionIDList = append(
				manifest.PermissionIDList, permission.ID,
			)
		}
	}
	return manifest.write()
}

// cleanup 按子项到父项顺序删除批次数据并支持重复清理。
func (manifest *manifest) cleanup(ctx context.Context) error {
	if manifest.CleanedAt != nil {
		return nil
	}
	stored, err := readManifest(manifest.path)
	if err != nil {
		return err
	}
	if stored.CleanedAt != nil {
		manifest.CleanedAt = stored.CleanedAt
		return nil
	}
	err = manifest.login(ctx)
	if err != nil {
		return err
	}
	client := manifest.newAPIClient("map")
	if manifest.Incomplete {
		err = manifest.recoverIDs(ctx, client)
		if err != nil {
			return fmt.Errorf("recover batch IDs: %w", err)
		}
	}
	err = manifest.deleteRows(ctx)
	if err != nil {
		return fmt.Errorf("cleanup failed; rerun permissions cleanup %s: %w",
			manifest.path, err)
	}
	commonClient, err := common.NewClient(manifest.BaseURL)
	if err != nil {
		return err
	}
	err = commonClient.DelUserByIDList(ctx, manifest.UserID, manifest.Token)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	manifest.CleanedAt = &now
	return manifest.write()
}

// deleteRows 分块删除已记录的权限、角色关联、角色及项目。
func (manifest *manifest) deleteRows(ctx context.Context) error {
	// deletion 保存一种批次资源的路径与 ID。
	type deletion struct {
		path   string
		idList []int32
	}
	deletionList := []deletion{
		{"user-role-permission-assoc", manifest.PermissionIDList},
		{"user-role-assoc", manifest.AssocIDList},
		{"user-role", manifest.RoleIDList},
		{"project", manifest.ProjectIDList},
	}
	client := &http.Client{Timeout: 5 * time.Second}
	for _, deletion := range deletionList {
		for start := 0; start < len(deletion.idList); start += 100 {
			end := min(start+100, len(deletion.idList))
			query := url.Values{}
			for _, id := range deletion.idList[start:end] {
				query.Add("IDList", fmt.Sprint(id))
			}
			address := manifest.BaseURL + "/" + deletion.path +
				"?" + query.Encode()
			request, err := http.NewRequestWithContext(
				ctx, http.MethodDelete, address, nil,
			)
			if err != nil {
				return err
			}
			request.Header.Set("Authorization", manifest.Token)
			response, err := client.Do(request)
			if err != nil {
				return err
			}
			_, readErr := io.Copy(io.Discard, response.Body)
			closeErr := response.Body.Close()
			if response.StatusCode < 200 || response.StatusCode >= 300 {
				return fmt.Errorf("delete %s returned HTTP %d",
					deletion.path, response.StatusCode)
			}
			err = errors.Join(readErr, closeErr)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
