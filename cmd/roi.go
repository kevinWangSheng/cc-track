package cmd

import (
	"fmt"
	"time"

	"github.com/shenghuikevin/cc-track/internal/analysis"
	"github.com/shenghuikevin/cc-track/internal/config"
	"github.com/shenghuikevin/cc-track/internal/output"
	"github.com/shenghuikevin/cc-track/internal/store"
	"github.com/spf13/cobra"
)

var (
	roiSince string
	roiRepo  string
)

var roiCmd = &cobra.Command{
	Use:   "roi",
	Short: "Show return on investment analysis",
	RunE:  runROI,
}

func init() {
	roiCmd.Flags().StringVar(&roiSince, "since", "", "start date (YYYY-MM-DD)")
	roiCmd.Flags().StringVar(&roiRepo, "repo", "", "override repo path")
	rootCmd.AddCommand(roiCmd)
}

func runROI(cmd *cobra.Command, args []string) error {
	sinceMs, untilMs := roiTimeRange()

	dbPath, err := config.DBPath()
	if err != nil {
		return err
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	report, err := analysis.AnalyzeROI(s, sinceMs, untilMs, roiRepo)
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

	printROIReport(report)
	return nil
}

func roiTimeRange() (int64, int64) {
	now := time.Now()
	untilMs := now.UnixMilli()

	if roiSince != "" {
		t, err := time.ParseInLocation("2006-01-02", roiSince, time.Local)
		if err == nil {
			return t.UnixMilli(), untilMs
		}
	}

	// Default: today
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	return today.UnixMilli(), untilMs
}

func printROIReport(r *analysis.ROIReport) {
	since := time.UnixMilli(r.SinceMs).Format("2006-01-02")
	fmt.Printf("ROI Report since %s\n\n", since)

	fmt.Printf("  Sessions:       %d\n", r.TotalSessions)
	fmt.Printf("  Prompts:        %d\n", r.TotalPrompts)
	fmt.Printf("  Tool Calls:     %d\n", r.TotalToolCalls)
	fmt.Printf("  Duration:       %s\n", formatDuration(r.TotalDurationMs))

	fmt.Printf("\n  Git Output (%d repo(s)):\n", r.ReposAnalyzed)
	fmt.Printf("    Commits:      %d\n", r.Commits)
	fmt.Printf("    Lines Added:  %d\n", r.LinesAdded)
	fmt.Printf("    Lines Removed:%d\n", r.LinesRemoved)
	net := r.LinesAdded - r.LinesRemoved
	fmt.Printf("    Net Change:   %+d\n", net)
}
