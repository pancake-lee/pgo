package papitable

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

type Role string

const (
	RoleAdministrator Role = "administrator"
	RoleManager       Role = "manager"   // 页面出错
	RoleEditor        Role = "editor"    // 不可编辑列配置，可以修改单元格
	RoleReader        Role = "reader"    // 不可编辑列配置，可以修改单元格
	RoleNone          Role = "none"      // 页面出错
	RoleMember        Role = "member"    // 页面出错
	RoleGuest         Role = "guest"     // 页面出错
	RoleForeigner     Role = "foreigner" // 页面出错
)

func (doc *MultiTableDoc) UpdateFieldPermissionRole(fieldId string, unitIds []string, role Role) error {
	if doc == nil {
		return plogger.LogErr(fmt.Errorf("doc is nil"))
	}
	if fieldId == "" {
		return plogger.LogErr(fmt.Errorf("fieldId is required"))
	}
	if len(unitIds) == 0 {
		return plogger.LogErr(fmt.Errorf("unitIds is required"))
	}
	if role == "" {
		return plogger.LogErr(fmt.Errorf("role is required"))
	}

	url := fmt.Sprintf("%s/api/v1/datasheet/%s/field/%s/permission/role/update", g_baseUrl, doc.DatasheetId, fieldId)

	// build request body
	reqBody := &updateFieldPermissionRoleRequest{
		UnitIds: unitIds,
		Role:    string(role),
	}

	req, err := putil.NewHttpRequestJson(http.MethodPost, url, getTokenHeader(), nil, reqBody)
	if err != nil {
		return plogger.LogErr(err)
	}

	resp, err := putil.HttpDo(req)
	if err != nil {
		return plogger.LogErr(err)
	}
	plogger.Debugf("UpdateFieldPermissionRole response: %s", string(resp))

	var respData genericResponse
	err = json.Unmarshal(resp, &respData)
	if err != nil {
		return plogger.LogErr(err)
	}

	if !respData.Success {
		return plogger.LogErr(fmt.Errorf("update field permission role failed: code=%d, message=%s", respData.Code, respData.Message))
	}

	plogger.Debugf("UpdateFieldPermissionRole success, fieldId=%s", fieldId)
	return nil
}

type updateFieldPermissionRoleRequest struct {
	UnitIds []string `json:"unitIds"`
	Role    string   `json:"role"`
}

// genericResponse 与很多 apitable API 响应结构一致
type genericResponse struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// EnableFieldPermission enables field-level permission for a field.
// includeExtend true will include extended permissions if supported by API.
func (doc *MultiTableDoc) EnableFieldPermission(fieldId string, includeExtend bool) error {
	if doc == nil {
		return plogger.LogErr(fmt.Errorf("doc is nil"))
	}
	if fieldId == "" {
		return plogger.LogErr(fmt.Errorf("fieldId is required"))
	}

	url := fmt.Sprintf("%s/api/v1/datasheet/%s/field/%s/permission/enable", g_baseUrl, doc.DatasheetId, fieldId)
	// query param includeExtend
	params := map[string]string{}
	if includeExtend {
		params["includeExtend"] = "true"
	}

	req, err := putil.NewHttpRequestJson(http.MethodPost, url, getTokenHeader(), params, nil)
	if err != nil {
		return plogger.LogErr(err)
	}

	resp, err := putil.HttpDo(req)
	if err != nil {
		return plogger.LogErr(err)
	}

	var respData genericResponse
	err = json.Unmarshal(resp, &respData)
	if err != nil {
		return plogger.LogErr(err)
	}

	if !respData.Success {
		return plogger.LogErr(fmt.Errorf("enable field permission failed: code=%d, message=%s", respData.Code, respData.Message))
	}

	plogger.Debugf("EnableFieldPermission success, fieldId=%s", fieldId)
	return nil
}

// DisableFieldPermission disables field-level permission for a field.
func (doc *MultiTableDoc) DisableFieldPermission(fieldId string) error {
	if doc == nil {
		return plogger.LogErr(fmt.Errorf("doc is nil"))
	}
	if fieldId == "" {
		return plogger.LogErr(fmt.Errorf("fieldId is required"))
	}

	url := fmt.Sprintf("%s/api/v1/datasheet/%s/field/%s/permission/disable", g_baseUrl, doc.DatasheetId, fieldId)

	req, err := putil.NewHttpRequestJson(http.MethodPost, url, getTokenHeader(), nil, nil)
	if err != nil {
		return plogger.LogErr(err)
	}

	resp, err := putil.HttpDo(req)
	if err != nil {
		return plogger.LogErr(err)
	}

	var respData genericResponse
	err = json.Unmarshal(resp, &respData)
	if err != nil {
		return plogger.LogErr(err)
	}

	if !respData.Success {
		return plogger.LogErr(fmt.Errorf("disable field permission failed: code=%d, message=%s", respData.Code, respData.Message))
	}

	plogger.Debugf("DisableFieldPermission success, fieldId=%s", fieldId)
	return nil
}
