package predis

import (
	"testing"

	"github.com/pancake-lee/pgo/pkg/pconfig"
)

func TestRedis(t *testing.T) {
	pconfig.MustInitConfig("../../configs/pancake.yaml")
	err := InitRedisByConfig()
	if err != nil {
		t.Fatalf("Failed to initialize Redis client: %v", err)
	}
	defer DefaultClient.Close()

	err = DefaultClient.Set("test_key", "test_value", 0).Err()
	if err != nil {
		t.Fatalf("Failed to set Redis key: %v", err)
	}

	val, err := DefaultClient.Get("test_key").Result()
	if err != nil {
		t.Fatalf("Failed to get Redis key: %v", err)
	}
	if val != "test_value" {
		t.Fatalf("Unexpected Redis value: %v", val)
	}

	err = DefaultClient.Del("test_key").Err()
	if err != nil {
		t.Fatalf("Failed to delete Redis key: %v", err)
	}
}
