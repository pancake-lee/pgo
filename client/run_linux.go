//go:build !windows

package main

import (
	"fmt"
	"os"
)

func runApp() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
