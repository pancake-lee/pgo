//go:build windows

package main

import (
	"github.com/pancake-lee/pgo/cmd/pgo/courseSwap"
	"github.com/pancake-lee/pgo/pkg/pclient"
)

func runUI() {
	app := pclient.NewApp("PGO Client")
	app.RegisterPage("调课", courseSwap.BuildPage)
	app.ShowAndRun()
}
