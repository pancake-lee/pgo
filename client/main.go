package main

import (
	"log"
	"pgo/client/courseSwap"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lshortfile)

	i := 1
	switch i {
	case 0:
		courseSwap.CourseSwap()
	}
}
