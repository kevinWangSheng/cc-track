package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "0.1.0-dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("cc-track %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
