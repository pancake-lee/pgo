package main

import (
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/pmq"
)

func checkRabbitMQ() {
	plogger.Info("Checking RabbitMQ...")
	if err := pmq.InitMQByConfig(); err != nil {
		// Treat as warning/skip if config is missing or invalid,
		// assuming not all services need RabbitMQ.
		// If it's critical, user should see the log.
		plogger.Warnf("RabbitMQ check skipped or failed: %v", err)
		return
	}

	if pmq.DefaultClient == nil {
		plogger.Error("RabbitMQ DefaultClient is nil after init")
		return
	}

	plogger.Info("RabbitMQ connected.")
}
