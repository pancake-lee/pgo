package common

import "testing"

func TestNewClient(t *testing.T) {
	client, err := NewClient("http://127.0.0.1:8080/")
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("client is nil")
	}
	if client.BaseURL() != "http://127.0.0.1:8080" {
		t.Fatalf("base URL = %q", client.BaseURL())
	}
	if _, err = NewClient("missing-scheme"); err == nil {
		t.Fatal("expected invalid URL error")
	}
}
