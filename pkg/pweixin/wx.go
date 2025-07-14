package pweixin

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

/*
请求之前，需要配置可信IP：
1：要么有一个和公司主题信息一致的域名，则用该公司授权的开发者账号/管理员账号来操作企微后台的应用设置，
2：要么就要配置回调服务，比起找公司的人走流程，还不如整这个回调服务。

首先是文档（可以不看）：

	接收消息与事件
	https://developer.work.weixin.qq.com/document/10514
	回调配置
	https://developer.work.weixin.qq.com/document/path/90930
	加解密方案说明
	https://developer.work.weixin.qq.com/devtool/introduce?id=36388

然后官方提供了这个回调服务的SDK：

	./pkg/pweixin/weworkapi_golang
	用于对接企业微信API的回调SDK
	https://developer.work.weixin.qq.com/devtool/introduce?id=10128
	自己改了参数的输入方式，包路径，屏蔽了sample.go代码，main和结构体都和httpserver.go重复定义了
	参数有token和EncodingAESKey是[企微后台->应用设置->可信IP->设置接收消息服务器URL]页面中的信息
	然后receiver_id不知道是什么，然后通过报错信息，提取了一个值，直接拿来用了
	这个服务运行起来，通过nginx公布到公网，然后把访问地址填写到上面说的页面。

	然后就可以真正配置可信IP了，本地调试根据报错信息，把当前公网IP添加到可信IP列表中。
	如果嫌麻烦，就把程序部署到固定公网IP的云服务器上开发。

最后，回调服务用来接收文档修改事件了
而可信IP，可以通过一个有公网IP的云服务器来跳转

	先配置一个nginx：deploy/nginx/wx_api.conf
	然后在企微后台的应用设置中，设置可信IP为该云服务器的IP
	而局域网/本地程序访问云服务器的nginx提供的服务

关于开发的咨询在https://developer.work.weixin.qq.com/community/question/ask
找企微客服是没用的
*/
var g_corpid string // 企微管理后台-我的企业-企业信息-企业ID
var g_agentid int32 // 企微管理后台-应用管理-对应的应用打开-AgentId

var g_corpSecret string // 企微管理后台-应用管理-对应的应用打开-Secret
var g_token string

var g_userSecret string // 企微管理后台-安全与管理-管理工具-通讯录同步-Secret
var g_userToken string

var g_baseUrl string = "https://qyapi.weixin.qq.com"

// --------------------------------------------------
func InitWxApiByConfig() error {
	corpid, err := pconfig.GetStringE("WX.corpid")
	if err != nil {
		return plogger.LogErr(err)
	}
	corpSecret, err := pconfig.GetStringE("WX.corpsecret")
	if err != nil {
		return plogger.LogErr(err)
	}

	return InitWxApi(corpid, corpSecret,
		pconfig.GetStringD("WX.usersecret", ""),
		int32(pconfig.GetInt64D("WX.agentid", 0)),
		pconfig.GetStringD("WX.baseUrl", ""),
	)
}

func InitWxApi(corpid, corpSecret, userSecret string, agentid int32, baseUrl string) (err error) {
	g_corpid = corpid
	g_corpSecret = corpSecret
	g_userSecret = userSecret
	g_agentid = agentid
	if baseUrl != "" {
		g_baseUrl = baseUrl
	}

	g_token, err = getToken(g_corpid, g_corpSecret)
	if err != nil {
		plogger.Error(err)
		return err
	}
	plogger.Debugf("getToken success, token: %s", g_token)

	if g_userSecret != "" {
		g_userToken, err = getToken(g_corpid, g_userSecret)
		if err != nil {
			plogger.Error(err)
			return err
		}
	}
	return nil
}

func getTokenQuerys() map[string]string {
	if g_token == "" {
		plogger.Error("g_token is empty, please call InitWxApi first")
		return nil
	}
	return map[string]string{
		"access_token": g_token,
		"debug":        "1", // 开发调试时可以开启debug
	}
}
func getUserTokenQuerys() map[string]string {
	if g_userToken == "" {
		plogger.Error("g_userToken is empty, please call InitWxApi first")
		return nil
	}
	return map[string]string{
		"access_token": g_userToken,
		"debug":        "1", // 开发调试时可以开启debug
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

// --------------------------------------------------

func getToken(corpid, corpsecret string) (string, error) {
	url := g_baseUrl + "/cgi-bin/gettoken"
	plogger.Debugf("getToken url: %s", url)

	req, err := putil.NewHttpRequest(http.MethodGet, url,
		nil, map[string]string{"corpid": corpid, "corpsecret": corpsecret}, "")
	if err != nil {
		return "", plogger.LogErr(err)
	}
	resp, err := putil.HttpDo(req)
	if err != nil {
		return "", plogger.LogErr(err)
	}

	var respMap map[string]any
	err = json.Unmarshal(resp, &respMap)
	if err != nil {
		plogger.Errorf("getToken json unmarshal error: %s", string(resp))
		return "", plogger.LogErr(err)
	}

	err = handleRespErrorByMap(respMap)
	if err != nil {
		return "", plogger.LogErr(err)
	}
	return putil.InterfaceToString(respMap["access_token"], ""), nil
}

// --------------------------------------------------
// 只能由通讯录同步助手的access_token来调用。同时需要保证通讯录同步功能是开启的。
func GetUserList() error {
	url := g_baseUrl + "/cgi-bin/user/list_id"

	req, err := putil.NewHttpRequestJson(http.MethodGet, url, nil,
		getUserTokenQuerys(),
		map[string]any{
			"cursor": "",
			"limit":  10000,
		})
	if err != nil {
		return plogger.LogErr(err)
	}
	resp, err := putil.HttpDo(req)
	if err != nil {
		return plogger.LogErr(err)
	}
	err = handleRespError(resp)
	if err != nil {
		return plogger.LogErr(err)
	}
	plogger.Debugf("getUserList resp : %s", string(resp))

	return nil
}

func SendMsg(touserList []string, msg string) error {
	url := g_baseUrl + "/cgi-bin/message/send"

	req, err := putil.NewHttpRequestJson(http.MethodPost, url, nil,
		getTokenQuerys(),
		map[string]any{
			"touser":  putil.StrListToStr(touserList, "|"),
			"msgtype": "text",
			"agentid": g_agentid,
			"text": map[string]any{
				"content": msg,
			},
			"safe": 0,
		})
	if err != nil {
		return plogger.LogErr(err)
	}
	resp, err := putil.HttpDo(req)
	if err != nil {
		return plogger.LogErr(err)
	}
	err = handleRespError(resp)
	if err != nil {
		return plogger.LogErr(err)
	}

	return nil
}
