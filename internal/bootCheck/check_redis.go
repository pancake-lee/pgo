package main

import (
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/predis"
)

func checkRedis() {
	plogger.Info("Checking Redis...")

	// Initialize Redis from config
	err := predis.InitRedisByConfig()
	if err != nil {
		plogger.Fatalf("Failed to init redis: %v", err)
	}

	// Ping
	if predis.DefaultClient == nil {
		plogger.Fatalf("Redis client is nil")
	}

	pong, err := predis.DefaultClient.Ping().Result()
	if err != nil {
		plogger.Fatalf("Redis ping failed: %v", err)
	}
	plogger.Infof("Redis ping success: %s", pong)
}
