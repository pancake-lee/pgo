package main

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/pancake-lee/pgo/pkg/plogger"
)

func main() {
	plogger.InitConsoleLogger()

	// 设置路由
	http.HandleFunc("/webhook", webhookHandler)
	http.HandleFunc("/", healthHandler)

	// 启动服务器
	addr := "0.0.0.0:7050"
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
	plogger.Debugf("\n=== Webhook Request Received at %s ===", timestamp)

	// 只接受POST方法
	if r.Method != http.MethodPost {
		plogger.Debugf("Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 打印请求头
	plogger.Debugf("Headers:")
	for name, values := range r.Header {
		for _, value := range values {
			plogger.Debugf("  %s: %s", name, value)
		}
	}

	// 打印URL参数
	if len(r.URL.RawQuery) > 0 {
		plogger.Debugf("Query Parameters: %s", r.URL.RawQuery)
		for key, values := range r.URL.Query() {
			for _, value := range values {
				plogger.Debugf("  %s: %s", key, value)
			}
		}
	}

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		plogger.Errorf("Error reading request body: %v", err)
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 打印原始请求体
	plogger.Debugf("Raw Body Length: %d bytes", len(body))
	plogger.Debugf("Raw Body: %s", string(body))

	// 尝试解析JSON
	if len(body) > 0 {
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err != nil {
			plogger.Debugf("Body is not valid JSON: %v", err)
		} else {
			// 格式化打印JSON
			prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
			if err != nil {
				plogger.Errorf("Error formatting JSON: %v", err)
			} else {
				plogger.Debugf("Formatted JSON Body:\n%s", string(prettyJSON))
			}
		}
	}

	// 打印客户端信息
	plogger.Debugf("Remote Address: %s", r.RemoteAddr)
	plogger.Debugf("User Agent: %s", r.UserAgent())
	plogger.Debugf("Content Length: %d", r.ContentLength)
	plogger.Debugf("Content Type: %s", r.Header.Get("Content-Type"))

	plogger.Debugf("=== End of Request ===")

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
