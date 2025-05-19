package config

import (
	"testing"
)

type testConf struct {
	// panic: using value obtained using unexported field
	// 必须是公开的字段，才能被反射写入
	// t0 int `default:"0"`

	T1 int
	T2 int `default:"2"`
	T3 int `default:"3"`
}

func TestConf(t *testing.T) {
	InitConfig(`D:\nycko\code\pgo\configs`)

	var err error

	var c testConf
	SetDefaults(&c)
	// t.Logf("load default : %v", c)

	err = Scan(&c)
	if err != nil {
		t.Fatalf("scan config failed: %v", err)
	}

	// t.Logf("load config : %v", c)

	if c.T1 != 0 { // 无默认值，无配置值
		t.Fatalf("T1 should be 0, but got %d", c.T1)
	}

	if c.T2 != 2 { // 有默认值，无配置值
		t.Fatalf("T2 should be 2, but got %d", c.T2)
	}

	if c.T3 != 33 { // 有默认值，有配置值
		t.Fatalf("T3 should be 33, but got %d", c.T3)
	}

	T4 := GetStringD("T4", "")
	if T4 != "44" { //无配置结构体，有配置值
		t.Fatalf("T4 should be 44, but got %s", T4)
	}

	T5 := GetStringD("T5", "55")
	if T5 != "55" { //无配置结构体，无配置值，代码配置默认值
		t.Fatalf("T5 should be 55, but got %s", T5)
	}

	_, err = GetStringE("T6")
	// t.Log(err)
	if err == nil {
		t.Fatalf("should get error")
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("should panic")
		}
	}()
	GetStringM("T7")
}
