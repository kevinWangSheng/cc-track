package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/shenghuikevin/cc-track/internal/config"
	"github.com/shenghuikevin/cc-track/internal/hook"
	"github.com/shenghuikevin/cc-track/internal/store"
	"github.com/spf13/cobra"
)

var hookCmd = &cobra.Command{
	Use:    "hook",
	Short:  "Process hook events from Claude Code (reads JSON from stdin)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}

		dbPath, err := config.DBPath()
		if err != nil {
			return err
		}

		s, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer s.Close()

		return hook.HandleEvent(data, s)
	},
}

func init() {
	rootCmd.AddCommand(hookCmd)
}
