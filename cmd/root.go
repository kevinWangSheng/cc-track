package cmd

import (
	"github.com/spf13/cobra"
)

var jsonOutput bool

var rootCmd = &cobra.Command{
	Use:   "cc-track",
	Short: "Local-first CLI tool for monitoring Claude Code usage",
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
}

func Execute() error {
	return rootCmd.Execute()
}
