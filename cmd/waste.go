package cmd

import (
	"fmt"

	"github.com/shenghuikevin/cc-track/internal/analysis"
	"github.com/shenghuikevin/cc-track/internal/config"
	"github.com/shenghuikevin/cc-track/internal/output"
	"github.com/shenghuikevin/cc-track/internal/store"
	"github.com/spf13/cobra"
)

var wasteSessionID string

var wasteCmd = &cobra.Command{
	Use:   "waste",
	Short: "Detect wasteful patterns in sessions",
	RunE:  runWaste,
}

func init() {
	wasteCmd.Flags().StringVar(&wasteSessionID, "session", "", "analyze a specific session (ID or prefix)")
	rootCmd.AddCommand(wasteCmd)
}

func runWaste(cmd *cobra.Command, args []string) error {
	dbPath, err := config.DBPath()
	if err != nil {
		return err
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	var sessionIDs []string
	if wasteSessionID != "" {
		sess, err := s.FindSessionByPrefix(wasteSessionID)
		if err != nil {
			return err
		}
		sessionIDs = []string{sess.ID}
	} else {
		ids, err := s.GetRecentSessionIDs(10)
		if err != nil {
			return err
		}
		sessionIDs = ids
	}

	if len(sessionIDs) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	report, err := analysis.AnalyzeWaste(s, sessionIDs)
	if err != nil {
		return err
	}

	if jsonOutput {
		j, err := output.JSON(report)
		if err != nil {
			return err
		}
		fmt.Println(j)
		return nil
	}

	printWasteReport(report)
	return nil
}

func printWasteReport(report *analysis.WasteReport) {
	fmt.Printf("Waste Analysis (%d sessions analyzed)\n\n", report.SessionsAnalyzed)

	if len(report.Findings) == 0 {
		fmt.Println("  No waste patterns detected. Nice!")
		return
	}

	fmt.Printf("  Found %d issue(s):\n\n", len(report.Findings))

	typeLabels := map[analysis.WasteType]string{
		analysis.WasteDuplicateCalls: "Duplicate Calls",
		analysis.WasteExcessiveReads: "Excessive Reads",
		analysis.WasteFailedRetries:  "Failed Retries",
		analysis.WasteEditRevert:     "Edit Revert",
		analysis.WasteZombieSession:  "Zombie Session",
	}

	for i, f := range report.Findings {
		label := typeLabels[f.Type]
		if label == "" {
			label = string(f.Type)
		}
		fmt.Printf("  %d. [%s] %s\n", i+1, label, f.Summary)
		if f.SessionID != "" {
			fmt.Printf("     Session: %s\n", f.SessionID[:min(12, len(f.SessionID))])
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
