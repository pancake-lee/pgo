package papitable

import (
	"encoding/json"
	"fmt"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

var g_token string

// var g_baseUrl string = "https://aitable.ai"
var g_baseUrl string = "http://192.168.17.163" //本地部署

// --------------------------------------------------
func InitAPITableByConfig() error {
	token, err := pconfig.GetStringE("APITable.token")
	if err != nil {
		return plogger.LogErr(err)
	}
	baseUrl := pconfig.GetStringD("APITable.baseUrl", "")

	return InitAPITable(token, baseUrl)
}

func InitAPITable(token string, baseUrl string) (err error) {
	g_token = token
	if baseUrl != "" {
		g_baseUrl = baseUrl
	}
	return nil
}

func getTokenHeader() map[string]string {
	if g_token == "" {
		plogger.Error("g_token is empty, please call InitAPITable first")
		return nil
	}
	return map[string]string{
		"Authorization": "Bearer " + g_token,
	}
}

// --------------------------------------------------
func handleRespError(resp []byte) error {
	var respMap map[string]any
	err := json.Unmarshal(resp, &respMap)
	if err != nil {
		plogger.Error(err)
		return err
	}
	return handleRespErrorByMap(respMap)
}

func handleRespErrorByMap(resp map[string]any) error {
	if resp["errcode"] != nil {
		e := putil.InterfaceToInt32(resp["errcode"], 0)
		if e != 0 {
			errMsg := putil.InterfaceToString(resp["errmsg"], "")
			return fmt.Errorf("wx api error[%d] : %s", e, errMsg)
		}
	}
	return nil
}
