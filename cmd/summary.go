package cmd

import (
	"fmt"
	"time"

	"github.com/shenghuikevin/cc-track/internal/config"
	"github.com/shenghuikevin/cc-track/internal/output"
	"github.com/shenghuikevin/cc-track/internal/store"
	"github.com/spf13/cobra"
)

var (
	summaryWeek  bool
	summaryMonth bool
	summarySince string
)

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show usage summary",
	RunE:  runSummary,
}

func init() {
	summaryCmd.Flags().BoolVar(&summaryWeek, "week", false, "last 7 days")
	summaryCmd.Flags().BoolVar(&summaryMonth, "month", false, "last 30 days")
	summaryCmd.Flags().StringVar(&summarySince, "since", "", "start date (YYYY-MM-DD)")
	rootCmd.AddCommand(summaryCmd)
}

func runSummary(cmd *cobra.Command, args []string) error {
	sinceMs, untilMs := summaryTimeRange()

	dbPath, err := config.DBPath()
	if err != nil {
		return err
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	sum, err := s.QuerySummary(sinceMs, untilMs)
	if err != nil {
		return err
	}

	if jsonOutput {
		j, err := output.JSON(sum)
		if err != nil {
			return err
		}
		fmt.Println(j)
		return nil
	}

	printSummary(sum, sinceMs)
	return nil
}

func summaryTimeRange() (int64, int64) {
	now := time.Now()
	untilMs := now.UnixMilli()

	if summarySince != "" {
		t, err := time.ParseInLocation("2006-01-02", summarySince, time.Local)
		if err == nil {
			return t.UnixMilli(), untilMs
		}
	}

	if summaryMonth {
		return now.AddDate(0, 0, -30).UnixMilli(), untilMs
	}
	if summaryWeek {
		return now.AddDate(0, 0, -7).UnixMilli(), untilMs
	}

	// Default: today
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	return today.UnixMilli(), untilMs
}

func printSummary(sum *store.SessionSummary, sinceMs int64) {
	since := time.UnixMilli(sinceMs).Format("2006-01-02 15:04")
	fmt.Printf("Summary since %s\n\n", since)

	fmt.Printf("  Sessions:    %d\n", sum.TotalSessions)
	fmt.Printf("  Prompts:     %d\n", sum.TotalPrompts)
	fmt.Printf("  Tool Calls:  %d\n", sum.TotalToolCalls)
	fmt.Printf("  Duration:    %s\n", formatDuration(sum.TotalDurationMs))
	fmt.Printf("  Error Rate:  %.1f%%\n", sum.ErrorRate)

	totalTokens := sum.TotalInputTokens + sum.TotalOutputTokens + sum.TotalCacheReadTokens + sum.TotalCacheCreationTokens
	if totalTokens > 0 {
		fmt.Printf("\n  Tokens:\n")
		fmt.Printf("    Input:          %s\n", formatTokens(sum.TotalInputTokens))
		fmt.Printf("    Output:         %s\n", formatTokens(sum.TotalOutputTokens))
		fmt.Printf("    Cache Read:     %s\n", formatTokens(sum.TotalCacheReadTokens))
		fmt.Printf("    Cache Creation: %s\n", formatTokens(sum.TotalCacheCreationTokens))
		fmt.Printf("    Total:          %s\n", formatTokens(totalTokens))
	}

	if len(sum.ToolBreakdown) > 0 {
		fmt.Println("\n  Tool Breakdown:")
		t := output.NewTable("Tool", "Count", "%")
		for _, tc := range sum.ToolBreakdown {
			t.AddRow(tc.Name, fmt.Sprintf("%d", tc.Count), fmt.Sprintf("%.1f%%", tc.Percent))
		}
		fmt.Print("  ")
		for i, line := range splitLines(t.String()) {
			if i > 0 {
				fmt.Print("  ")
			}
			fmt.Println(line)
		}
	}
}

func formatDuration(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func formatTokens(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
