package main

import (
	"flag"

	_ "github.com/pancake-lee/pgo/internal/canal/service" // Register handlers
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/pdb"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/predis"
)

func main() {
	l := flag.Bool("l", false, "log to console, default is false")
	c := flag.String("c", "./configs/",
		"config folder, should have common.yaml and ${execName}.yaml")
	flag.Parse()

	pconfig.MustInitConfig(*c)
	plogger.InitFromConfig(*l)
	predis.MustInitRedisByConfig()

	var dbConf pdb.MysqlConfig
	err := pconfig.Scan(&dbConf)
	if err != nil {
		panic(err)
	}

	client, err := pdb.NewCanal(dbConf)
	if err != nil {
		plogger.Error(err)
		return
	}

	if err := client.Run(); err != nil {
		plogger.Error(err)
	}
}
