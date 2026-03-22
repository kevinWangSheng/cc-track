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
	trendMonth bool
	trendSince string
)

var trendCmd = &cobra.Command{
	Use:   "trend",
	Short: "Show usage trends over time",
	RunE:  runTrend,
}

func init() {
	trendCmd.Flags().BoolVar(&trendMonth, "month", false, "last 30 days")
	trendCmd.Flags().StringVar(&trendSince, "since", "", "start date (YYYY-MM-DD)")
	rootCmd.AddCommand(trendCmd)
}

// trendDay holds a single day's display data.
type trendDay struct {
	Date       string                  `json:"date"`
	Sessions   int                     `json:"sessions"`
	Prompts    int                     `json:"prompts"`
	ToolCalls  int                     `json:"tool_calls"`
	DurationMs int64                   `json:"duration_ms"`
	Tokens     int64                   `json:"tokens"`
	Cost       *analysis.CostBreakdown `json:"cost"`
	WasteCount int                     `json:"waste_count"`
}

type trendReport struct {
	Days []trendDay `json:"days"`
}

func runTrend(cmd *cobra.Command, args []string) error {
	sinceMs, untilMs := trendTimeRange()

	dbPath, err := config.DBPath()
	if err != nil {
		return err
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	daily, err := s.QueryDailyStats(sinceMs, untilMs)
	if err != nil {
		return err
	}

	// Get per-day waste counts
	dailySessionIDs, _ := s.QueryDailySessionIDs(sinceMs, untilMs)
	wasteByDay := make(map[string]int)
	for day, ids := range dailySessionIDs {
		report, err := analysis.AnalyzeWaste(s, ids)
		if err == nil {
			wasteByDay[day] = len(report.Findings)
		}
	}

	// Build report
	report := trendReport{}
	for _, d := range daily {
		pricing := calcDayCost(s, d)
		report.Days = append(report.Days, trendDay{
			Date:       d.Date,
			Sessions:   d.Sessions,
			Prompts:    d.Prompts,
			ToolCalls:  d.ToolCalls,
			DurationMs: d.DurationMs,
			Tokens:     d.TotalTokens(),
			Cost:       pricing,
			WasteCount: wasteByDay[d.Date],
		})
	}

	if jsonOutput {
		j, err := output.JSON(report)
		if err != nil {
			return err
		}
		fmt.Println(j)
		return nil
	}

	printTrend(report)
	return nil
}

func calcDayCost(s *store.Store, d store.DailyStats) *analysis.CostBreakdown {
	// Parse day to get time range for that day
	t, err := time.ParseInLocation("2006-01-02", d.Date, time.Local)
	if err != nil {
		return nil
	}
	sinceMs := t.UnixMilli()
	untilMs := t.Add(24 * time.Hour).UnixMilli()

	byModel, err := s.QueryTokensByModel(sinceMs, untilMs)
	if err != nil {
		return nil
	}
	total := &analysis.CostBreakdown{}
	for _, mt := range byModel {
		pricing := analysis.LookupPricing(mt.Model)
		c := analysis.CalculateCost(mt.TotalInputTokens, mt.TotalOutputTokens,
			mt.TotalCacheReadTokens, mt.TotalCacheCreationTokens, pricing)
		total.InputCost += c.InputCost
		total.OutputCost += c.OutputCost
		total.CacheReadCost += c.CacheReadCost
		total.CacheCreationCost += c.CacheCreationCost
		total.TotalCost += c.TotalCost
	}
	return total
}

func trendTimeRange() (int64, int64) {
	now := time.Now()
	untilMs := now.UnixMilli()

	if trendSince != "" {
		t, err := time.ParseInLocation("2006-01-02", trendSince, time.Local)
		if err == nil {
			return t.UnixMilli(), untilMs
		}
	}

	if trendMonth {
		return now.AddDate(0, 0, -30).UnixMilli(), untilMs
	}

	// Default: last 7 days
	return now.AddDate(0, 0, -7).UnixMilli(), untilMs
}

func printTrend(report trendReport) {
	if len(report.Days) == 0 {
		fmt.Println("No data for this period.")
		return
	}

	// Table view
	t := output.NewTable("Date", "Sessions", "Prompts", "Tools", "Duration", "Tokens", "Cost", "Waste")
	for _, d := range report.Days {
		costStr := "-"
		if d.Cost != nil && d.Cost.TotalCost > 0 {
			costStr = fmt.Sprintf("$%.2f", d.Cost.TotalCost)
		}
		wasteStr := "-"
		if d.WasteCount > 0 {
			wasteStr = fmt.Sprintf("%d", d.WasteCount)
		}
		t.AddRow(
			d.Date,
			fmt.Sprintf("%d", d.Sessions),
			fmt.Sprintf("%d", d.Prompts),
			fmt.Sprintf("%d", d.ToolCalls),
			formatDuration(d.DurationMs),
			formatTokens(d.Tokens),
			costStr,
			wasteStr,
		)
	}
	fmt.Print(t.String())

	// Cost bar chart
	fmt.Println("\nDaily Cost:")
	printBarChart(report.Days)
}

func printBarChart(days []trendDay) {
	const maxWidth = 40

	// Find max cost
	var maxCost float64
	for _, d := range days {
		if d.Cost != nil && d.Cost.TotalCost > maxCost {
			maxCost = d.Cost.TotalCost
		}
	}

	if maxCost == 0 {
		fmt.Println("  (no cost data)")
		return
	}

	for _, d := range days {
		cost := 0.0
		if d.Cost != nil {
			cost = d.Cost.TotalCost
		}
		barLen := int(cost / maxCost * maxWidth)
		if cost > 0 && barLen == 0 {
			barLen = 1
		}
		bar := ""
		for i := 0; i < barLen; i++ {
			bar += "█"
		}
		// Show short date (MM-DD)
		dateShort := d.Date[5:] // "03-22"
		fmt.Printf("  %s %s $%.2f\n", dateShort, bar, cost)
	}
}
