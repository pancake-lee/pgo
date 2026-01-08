package papitable

import (
	"context"
	"net/http"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
	"golang.org/x/time/rate"
)

// APITable API 接口
// https://developers.aitable.ai/zh-CN/api/cn/reference/

var g_token string

var g_baseUrl string = "https://aitable.ai"

var g_limiter = rate.NewLimiter(40, 1)

func safeHttpDo(req *http.Request) ([]byte, error) {
	err := g_limiter.Wait(context.Background())
	if err != nil {
		return nil, err
	}
	return putil.HttpDo(req)
}

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
