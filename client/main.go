package main

import (
	"log"

	"github.com/pancake-lee/pgo/client/courseSwap"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lshortfile)

	i := 0
	switch i {
	case 0:
		courseSwap.CourseSwap()
	}
}
