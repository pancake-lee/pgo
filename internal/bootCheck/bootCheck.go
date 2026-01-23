package main

import (
	"flag"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/pdb"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/pmq"
	"github.com/pancake-lee/pgo/pkg/predis"
)

func main() {
	l := flag.Bool("l", false, "log to console, default is false")
	c := flag.String("c", "",
		"config folder, should have common.yaml and ${execName}.yaml")
	flag.Parse()

	pconfig.MustInitConfig(*c)
	plogger.InitFromConfig(*l)
	plogger.Info("BootCheck started...")

	if !pconfig.Has(predis.DefaultConfigGroup) {
		plogger.Info("Redis config not found, skipping check.")
	} else {
		checkRedis()
	}

	if !pconfig.Has(pmq.DefaultConfigGroup) {
		plogger.Info("RabbitMQ config not found, skipping check.")
	} else {
		checkRabbitMQ()
	}

	// Includes: Create DB if not exists, AutoMigrate
	if !pconfig.Has(pdb.DefaultConfigGroup) {
		plogger.Info("Mysql config not found, skipping check.")
	} else {
		checkMysql()
	}

	plogger.Info("BootCheck finished successfully.")
}
