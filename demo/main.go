package main

import (
	"flag"
	"net/http"
)

// 浏览器访问 http://127.0.0.1:8080/hello?user=pancake
func helloHandler(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")
	if user == "" {
		user = "Guest"
	}
	w.Write([]byte("Hello, " + user + "!"))
}

func main() {
	port := flag.String("port", "8080", "listening port, default is 8080")
	flag.Parse()

	http.HandleFunc("/hello", helloHandler)
	// 使用 127.0.0.1 作为监听地址，只能本机访问，但不需要经过防火墙，不用弹出防火墙提示，方便调试
	http.ListenAndServe("127.0.0.1:"+*port, nil)
}
