package main

import (
	"github.com/pancake-lee/pgo/client/courseSwap"
	"github.com/pancake-lee/pgo/pkg/plogger"
)

func main() {
	plogger.SetJsonLog(false)
	plogger.InitConsoleLogger()

	i := 0
	switch i {
	case 0:
		courseSwap.CourseSwap()
	}
}
