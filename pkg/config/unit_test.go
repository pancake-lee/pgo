package config

import (
	"log"
	"os"
	"testing"
)

func TestConf(t *testing.T) {
	d, _ := os.Getwd()
	log.Println("test ", d)
	var mConf myConfig
	c, err := LoadConfFromFile("../../configs/config.yaml", &mConf)
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}
	t.Logf("load config struct : %v", mConf)

	s, _ := c.Value("Grpc.Addr").String()
	t.Logf("load config value : %v", s)
}
