package main

import (
	"log"

	"github.com/pancake-lee/pgo/cmd/pgo/internal/prettyCode"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "pgo",
	Short:   "pgo: It's just a tool.",
	Long:    `pgo: It's just a tool.`,
	Version: version,
}

func init() {
	rootCmd.AddCommand(prettyCode.PrettyCode)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
