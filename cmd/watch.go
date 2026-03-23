package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shenghuikevin/cc-track/internal/analysis"
	"github.com/shenghuikevin/cc-track/internal/config"
	"github.com/shenghuikevin/cc-track/internal/store"
	"github.com/spf13/cobra"
)

var watchInterval int

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Live dashboard of current session activity",
	RunE:  runWatch,
}

func init() {
	watchCmd.Flags().IntVar(&watchInterval, "interval", 3, "refresh interval in seconds")
	rootCmd.AddCommand(watchCmd)
}

type watchState struct {
	sessions   int
	prompts    int
	toolCalls  int
	tokens     int64
	cost       float64
	lastTool   string
	lastPrompt string
}

func runWatch(cmd *cobra.Command, args []string) error {
	dbPath, err := config.DBPath()
	if err != nil {
		return err
	}

	// Handle Ctrl+C gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(time.Duration(watchInterval) * time.Second)
	defer ticker.Stop()

	fmt.Println("cc-track watch — press Ctrl+C to stop")
	fmt.Println()

	// Render immediately, then on each tick
	renderWatch(dbPath)
	for {
		select {
		case <-ticker.C:
			renderWatch(dbPath)
		case <-sigCh:
			fmt.Println("\nStopped.")
			return nil
		}
	}
}

func renderWatch(dbPath string) {
	s, err := store.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  DB error: %v\n", err)
		return
	}
	defer s.Close()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	sinceMs := today.UnixMilli()
	untilMs := now.UnixMilli()

	sum, err := s.QuerySummary(sinceMs, untilMs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Query error: %v\n", err)
		return
	}

	cost := calcWatchCost(s, sinceMs, untilMs)

	// Get latest session info
	sessions, _ := s.ListSessions(1)
	var latestProject, latestModel, latestID string
	if len(sessions) > 0 {
		latestProject = sessions[0].ProjectStr
		latestModel = sessions[0].ModelStr
		latestID = sessions[0].ID
		if len(latestID) > 12 {
			latestID = latestID[:12]
		}
	}

	// Get latest tool call
	var lastToolName string
	if len(sessions) > 0 {
		tl, err := s.GetSessionTimeline(sessions[0].ID)
		if err == nil && len(tl.ToolCalls) > 0 {
			last := tl.ToolCalls[len(tl.ToolCalls)-1]
			lastToolName = last.ToolName
		}
	}

	// Clear screen and render
	fmt.Print("\033[2J\033[H")
	fmt.Printf("  cc-track watch  |  %s\n", now.Format("15:04:05"))
	fmt.Println("  ─────────────────────────────────────")

	if latestID != "" {
		fmt.Printf("  Latest:   %s  %s  (%s)\n", latestID, latestProject, latestModel)
	}

	fmt.Println()
	fmt.Printf("  Sessions:    %d\n", sum.TotalSessions)
	fmt.Printf("  Prompts:     %d\n", sum.TotalPrompts)
	fmt.Printf("  Tool Calls:  %d\n", sum.TotalToolCalls)
	fmt.Printf("  Error Rate:  %.1f%%\n", sum.ErrorRate)

	totalTokens := sum.TotalInputTokens + sum.TotalOutputTokens + sum.TotalCacheReadTokens + sum.TotalCacheCreationTokens
	fmt.Printf("  Tokens:      %s\n", formatTokens(totalTokens))

	if cost > 0 {
		fmt.Printf("  Cost:        $%.2f\n", cost)
	}

	if lastToolName != "" {
		fmt.Printf("\n  Last Tool:   %s\n", lastToolName)
	}

	// Tool breakdown (top 5)
	if len(sum.ToolBreakdown) > 0 {
		fmt.Println("\n  Top Tools:")
		limit := len(sum.ToolBreakdown)
		if limit > 5 {
			limit = 5
		}
		for _, tc := range sum.ToolBreakdown[:limit] {
			bar := ""
			barLen := int(tc.Percent / 100 * 20)
			for i := 0; i < barLen; i++ {
				bar += "█"
			}
			fmt.Printf("    %-12s %s %.0f%%\n", tc.Name, bar, tc.Percent)
		}
	}
}

func calcWatchCost(s *store.Store, sinceMs, untilMs int64) float64 {
	byModel, err := s.QueryTokensByModel(sinceMs, untilMs)
	if err != nil {
		return 0
	}
	var total float64
	for _, mt := range byModel {
		pricing := analysis.LookupPricing(mt.Model)
		c := analysis.CalculateCost(mt.TotalInputTokens, mt.TotalOutputTokens,
			mt.TotalCacheReadTokens, mt.TotalCacheCreationTokens, pricing)
		total += c.TotalCost
	}
	return total
}
