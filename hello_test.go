package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHelloHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "http://127.0.0.1:8080/hello?user=pancake", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(helloHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := "pancake"
	if !strings.Contains(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestHelloService(t *testing.T) {

	// 手动启动服务，或者使用以下代码通过协程启动服务
	// go main()
	// time.Sleep(1 * time.Second)

	resp, err := http.Get("http://127.0.0.1:8080/hello?user=pancake")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	expected := "pancake"
	if !strings.Contains(string(body), expected) {
		t.Errorf("handler returned unexpected body: got %v want %v", string(body), expected)
	}
}
