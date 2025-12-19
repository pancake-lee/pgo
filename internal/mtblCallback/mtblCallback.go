package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"time"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/pmq"
	"github.com/pancake-lee/pgo/pkg/putil"
)

// mtblCallback 多维表格数据变更回调服务
// mtbl = multi table 多维表格
// 是相对于ltbl = local table 则本地数据库而言的

func main() {
	var isLogConsole = flag.Bool("l", false, "true: log in file and console; false: only log in file")
	var configFile = flag.String("c", "", "The specified config file")
	flag.Parse()

	pconfig.MustInitConfig(*configFile)
	plogger.InitFromConfig(*isLogConsole)
	pmq.MustInitMQByConfig()

	// 设置路由
	http.HandleFunc("/webhook", webhookHandler)
	http.HandleFunc("/", healthHandler)

	// 启动服务器
	addr := "0.0.0.0:" + putil.Int64ToStr(pconfig.GetInt64M("APITable.cbServerPort"))
	plogger.Debugf("Starting webhook server on %s", addr)
	plogger.Debugf("Webhook endpoint: http://%s/webhook", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		plogger.Errorf("Server failed to start: %v", err)
		return
	}
}

// webhook处理函数
func webhookHandler(w http.ResponseWriter, r *http.Request) {
	// 记录请求时间
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	plogger.Debugf("--------------------------------------------------")

	// 只接受POST方法
	if r.Method != http.MethodPost {
		plogger.Debugf("Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 打印请求头
	plogger.Debugf("Headers : %s", r.Header)
	plogger.Debugf("Query   : %s", r.URL.RawQuery)

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		plogger.Errorf("Error reading request body: %v", err)
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 打印原始请求体
	plogger.Debugf("Raw Body: %s", string(body))

	// 尝试解析JSON
	if len(body) > 0 {
		// {"datasheetId":"111","recordId":"123"}
		var jsonStrReq string = string(body)
		err = pmq.DefaultClient.SendServerEventStr(context.Background(),
			"apitable_event", "apitable_change", &jsonStrReq,
		)
		if err != nil {
			plogger.LogErr(err)
			return
		}
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"success":   true,
		"message":   "Webhook received successfully",
		"timestamp": timestamp,
	}

	json.NewEncoder(w).Encode(response)
}

// 健康检查处理函数
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "apitable-webhook-server",
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
	}

	json.NewEncoder(w).Encode(response)
}
