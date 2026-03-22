package cmd

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"

	"github.com/shenghuikevin/cc-track/internal/output"
	"github.com/shenghuikevin/cc-track/internal/store"
	"github.com/spf13/cobra"
)

var (
	exportFormat string
	exportSince  string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export session data",
	RunE:  runExport,
}

func init() {
	exportCmd.Flags().StringVar(&exportFormat, "format", "json", "output format: json or csv")
	exportCmd.Flags().StringVar(&exportSince, "since", "", "start date (YYYY-MM-DD)")
	rootCmd.AddCommand(exportCmd)
}

func runExport(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	sinceMs := int64(0)
	if exportSince != "" {
		t, err := time.ParseInLocation("2006-01-02", exportSince, time.Local)
		if err != nil {
			return fmt.Errorf("invalid date: %s", exportSince)
		}
		sinceMs = t.UnixMilli()
	}

	sessions, err := s.ExportSessions(sinceMs)
	if err != nil {
		return err
	}

	switch exportFormat {
	case "json":
		j, err := output.JSON(sessions)
		if err != nil {
			return err
		}
		fmt.Println(j)
	case "csv":
		return writeCSV(sessions)
	default:
		return fmt.Errorf("unknown format: %s (use json or csv)", exportFormat)
	}
	return nil
}

func writeCSV(sessions []store.ExportSession) error {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	header := []string{"session_id", "project", "branch", "model", "started_at", "ended_at", "prompts", "tool_calls"}
	if err := w.Write(header); err != nil {
		return fmt.Errorf("csv: write header: %w", err)
	}

	for _, sess := range sessions {
		started := time.UnixMilli(sess.StartedAt).Format(time.DateTime)
		ended := ""
		if sess.EndedAtVal > 0 {
			ended = time.UnixMilli(sess.EndedAtVal).Format(time.DateTime)
		}
		row := []string{
			sess.ID,
			sess.ProjectStr,
			sess.BranchStr,
			sess.ModelStr,
			started,
			ended,
			fmt.Sprintf("%d", sess.TotalPrompts),
			fmt.Sprintf("%d", sess.TotalToolCalls),
		}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("csv: write row: %w", err)
		}
	}
	return nil
}
