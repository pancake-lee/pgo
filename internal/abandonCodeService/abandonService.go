package main

import (
	"flag"

	"github.com/pancake-lee/pgo/internal/abandonCodeService/service"
	"github.com/pancake-lee/pgo/pkg/papp"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
)

func main() {
	l := flag.Bool("l", false, "log to console, default is false")
	c := flag.String("c", "./configs/",
		"config folder, should have common.yaml and ${execName}.yaml")
	flag.Parse()

	pconfig.MustInitConfig(*c)
	plogger.InitServiceLogger(*l)

	var s service.AbandonCodeCURDServer

	papp.RunKratosApp(&s)
}
