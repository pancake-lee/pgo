package main

import "fmt"

var (
	version = "unknown"
	commit  = "unknown"
	date    = "unknown"
)

func buildInfo() string {
	return fmt.Sprintf("version=%s\ncommit=%s\ndate=%s", version, commit, date)
}
