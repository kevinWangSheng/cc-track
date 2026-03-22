package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/shenghuikevin/cc-track/internal/analysis"
	"github.com/shenghuikevin/cc-track/internal/config"
	"github.com/shenghuikevin/cc-track/internal/store"
	"github.com/spf13/cobra"
)

var (
	reportFormat string
	reportSince  string
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate usage report",
	RunE:  runReport,
}

func init() {
	reportCmd.Flags().StringVar(&reportFormat, "format", "md", "output format: md, html")
	reportCmd.Flags().StringVar(&reportSince, "since", "", "start date (YYYY-MM-DD), default last 7 days")
	rootCmd.AddCommand(reportCmd)
}

func runReport(cmd *cobra.Command, args []string) error {
	sinceMs, untilMs := reportTimeRange()

	dbPath, err := config.DBPath()
	if err != nil {
		return err
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	// Gather all data
	summary, err := s.QuerySummary(sinceMs, untilMs)
	if err != nil {
		return err
	}

	daily, err := s.QueryDailyStats(sinceMs, untilMs)
	if err != nil {
		return err
	}

	sessions, err := s.ListSessions(100)
	if err != nil {
		return err
	}
	// Filter sessions to time range
	var rangeSessions []store.SessionRow
	for _, sess := range sessions {
		if sess.StartedAt >= sinceMs && sess.StartedAt < untilMs {
			rangeSessions = append(rangeSessions, sess)
		}
	}

	// Cost
	cost := calcCostFromStore(s, sinceMs, untilMs)

	// Waste
	var sessionIDs []string
	for _, sess := range rangeSessions {
		sessionIDs = append(sessionIDs, sess.ID)
	}
	var wasteReport *analysis.WasteReport
	if len(sessionIDs) > 0 {
		wasteReport, _ = analysis.AnalyzeWaste(s, sessionIDs)
	}

	// ROI
	roiReport, _ := analysis.AnalyzeROI(s, sinceMs, untilMs, "")

	sinceDate := time.UnixMilli(sinceMs).Format("2006-01-02")
	untilDate := time.UnixMilli(untilMs).Format("2006-01-02")

	switch reportFormat {
	case "html":
		fmt.Println(generateHTML(sinceDate, untilDate, summary, daily, rangeSessions, cost, wasteReport, roiReport))
	default:
		fmt.Println(generateMarkdown(sinceDate, untilDate, summary, daily, rangeSessions, cost, wasteReport, roiReport))
	}
	return nil
}

func reportTimeRange() (int64, int64) {
	now := time.Now()
	untilMs := now.UnixMilli()
	if reportSince != "" {
		t, err := time.ParseInLocation("2006-01-02", reportSince, time.Local)
		if err == nil {
			return t.UnixMilli(), untilMs
		}
	}
	return now.AddDate(0, 0, -7).UnixMilli(), untilMs
}

func generateMarkdown(since, until string, sum *store.SessionSummary, daily []store.DailyStats,
	sessions []store.SessionRow, cost *analysis.CostBreakdown,
	waste *analysis.WasteReport, roi *analysis.ROIReport) string {

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# cc-track Report: %s ~ %s\n\n", since, until))

	// Summary
	sb.WriteString("## Overview\n\n")
	sb.WriteString(fmt.Sprintf("| Metric | Value |\n"))
	sb.WriteString(fmt.Sprintf("|--------|-------|\n"))
	sb.WriteString(fmt.Sprintf("| Sessions | %d |\n", sum.TotalSessions))
	sb.WriteString(fmt.Sprintf("| Prompts | %d |\n", sum.TotalPrompts))
	sb.WriteString(fmt.Sprintf("| Tool Calls | %d |\n", sum.TotalToolCalls))
	sb.WriteString(fmt.Sprintf("| Duration | %s |\n", formatDuration(sum.TotalDurationMs)))
	sb.WriteString(fmt.Sprintf("| Error Rate | %.1f%% |\n", sum.ErrorRate))
	if cost != nil && cost.TotalCost > 0 {
		sb.WriteString(fmt.Sprintf("| Estimated Cost | $%.2f |\n", cost.TotalCost))
	}
	sb.WriteString("\n")

	// Tokens
	totalTokens := sum.TotalInputTokens + sum.TotalOutputTokens + sum.TotalCacheReadTokens + sum.TotalCacheCreationTokens
	if totalTokens > 0 {
		sb.WriteString("## Tokens\n\n")
		sb.WriteString("| Type | Count |\n")
		sb.WriteString("|------|-------|\n")
		sb.WriteString(fmt.Sprintf("| Input | %s |\n", formatTokens(sum.TotalInputTokens)))
		sb.WriteString(fmt.Sprintf("| Output | %s |\n", formatTokens(sum.TotalOutputTokens)))
		sb.WriteString(fmt.Sprintf("| Cache Read | %s |\n", formatTokens(sum.TotalCacheReadTokens)))
		sb.WriteString(fmt.Sprintf("| Cache Creation | %s |\n", formatTokens(sum.TotalCacheCreationTokens)))
		sb.WriteString(fmt.Sprintf("| **Total** | **%s** |\n", formatTokens(totalTokens)))
		sb.WriteString("\n")
	}

	// Cost breakdown
	if cost != nil && cost.TotalCost > 0 {
		sb.WriteString("## Cost Breakdown\n\n")
		sb.WriteString("| Category | Cost |\n")
		sb.WriteString("|----------|------|\n")
		sb.WriteString(fmt.Sprintf("| Input | $%.4f |\n", cost.InputCost))
		sb.WriteString(fmt.Sprintf("| Output | $%.4f |\n", cost.OutputCost))
		sb.WriteString(fmt.Sprintf("| Cache Read | $%.4f |\n", cost.CacheReadCost))
		sb.WriteString(fmt.Sprintf("| Cache Creation | $%.4f |\n", cost.CacheCreationCost))
		sb.WriteString(fmt.Sprintf("| **Total** | **$%.2f** |\n", cost.TotalCost))
		sb.WriteString("\n")
	}

	// Daily trend
	if len(daily) > 0 {
		sb.WriteString("## Daily Trend\n\n")
		sb.WriteString("| Date | Sessions | Prompts | Tools | Duration |\n")
		sb.WriteString("|------|----------|---------|-------|----------|\n")
		for _, d := range daily {
			sb.WriteString(fmt.Sprintf("| %s | %d | %d | %d | %s |\n",
				d.Date, d.Sessions, d.Prompts, d.ToolCalls, formatDuration(d.DurationMs)))
		}
		sb.WriteString("\n")
	}

	// Tool breakdown
	if len(sum.ToolBreakdown) > 0 {
		sb.WriteString("## Tool Usage\n\n")
		sb.WriteString("| Tool | Count | % |\n")
		sb.WriteString("|------|-------|---|\n")
		for _, tc := range sum.ToolBreakdown {
			sb.WriteString(fmt.Sprintf("| %s | %d | %.1f%% |\n", tc.Name, tc.Count, tc.Percent))
		}
		sb.WriteString("\n")
	}

	// Top sessions
	if len(sessions) > 0 {
		sb.WriteString("## Top Sessions\n\n")
		sb.WriteString("| ID | Project | Model | Prompts | Tools |\n")
		sb.WriteString("|----|---------|-------|---------|-------|\n")
		limit := len(sessions)
		if limit > 10 {
			limit = 10
		}
		for _, sess := range sessions[:limit] {
			id := sess.ID
			if len(id) > 12 {
				id = id[:12]
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %d |\n",
				id, sess.ProjectStr, sess.ModelStr, sess.TotalPrompts, sess.TotalToolCalls))
		}
		sb.WriteString("\n")
	}

	// Waste
	if waste != nil && len(waste.Findings) > 0 {
		sb.WriteString("## Waste Detection\n\n")
		sb.WriteString(fmt.Sprintf("Found **%d** issue(s):\n\n", len(waste.Findings)))
		for i, f := range waste.Findings {
			sid := f.SessionID
			if len(sid) > 12 {
				sid = sid[:12]
			}
			sb.WriteString(fmt.Sprintf("%d. **[%s]** %s (session: `%s`)\n", i+1, f.Type, f.Summary, sid))
		}
		sb.WriteString("\n")
	}

	// ROI
	if roi != nil && roi.Commits > 0 {
		sb.WriteString("## ROI\n\n")
		sb.WriteString("| Metric | Value |\n")
		sb.WriteString("|--------|-------|\n")
		sb.WriteString(fmt.Sprintf("| Repos Analyzed | %d |\n", roi.ReposAnalyzed))
		sb.WriteString(fmt.Sprintf("| Commits | %d |\n", roi.Commits))
		sb.WriteString(fmt.Sprintf("| Lines Added | +%d |\n", roi.LinesAdded))
		sb.WriteString(fmt.Sprintf("| Lines Removed | -%d |\n", roi.LinesRemoved))
		sb.WriteString(fmt.Sprintf("| Net Change | %+d |\n", roi.LinesAdded-roi.LinesRemoved))
		sb.WriteString("\n")
	}

	sb.WriteString("---\n*Generated by [cc-track](https://github.com/kevinWangSheng/cc-track)*\n")
	return sb.String()
}

func generateHTML(since, until string, sum *store.SessionSummary, daily []store.DailyStats,
	sessions []store.SessionRow, cost *analysis.CostBreakdown,
	waste *analysis.WasteReport, roi *analysis.ROIReport) string {

	md := generateMarkdown(since, until, sum, daily, sessions, cost, waste, roi)

	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>cc-track Report</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 900px; margin: 40px auto; padding: 0 20px; color: #333; }
  h1 { border-bottom: 2px solid #333; padding-bottom: 10px; }
  h2 { color: #555; margin-top: 30px; }
  table { border-collapse: collapse; width: 100%; margin: 10px 0; }
  th, td { border: 1px solid #ddd; padding: 8px 12px; text-align: left; }
  th { background: #f5f5f5; font-weight: 600; }
  tr:nth-child(even) { background: #fafafa; }
  code { background: #f0f0f0; padding: 2px 6px; border-radius: 3px; font-size: 0.9em; }
  strong { color: #c0392b; }
  hr { margin-top: 30px; border: none; border-top: 1px solid #ddd; }
</style>
</head>
<body>
`)
	// Simple markdown-to-html conversion
	inTable := false
	isFirstRow := false
	lines := strings.Split(md, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		isTableLine := strings.HasPrefix(line, "|")
		isSeparator := isTableLine && strings.Contains(line, "---")

		// Handle table open/close
		if isTableLine && !inTable {
			sb.WriteString("<table>\n")
			inTable = true
			isFirstRow = true
		}
		if !isTableLine && inTable {
			sb.WriteString("</table>\n")
			inTable = false
		}

		if isSeparator {
			// Keep in table but skip rendering the separator row
			continue
		} else if isTableLine {
			cells := strings.Split(line, "|")
			tag := "td"
			if isFirstRow {
				tag = "th"
			}
			sb.WriteString("<tr>")
			for _, cell := range cells {
				cell = strings.TrimSpace(cell)
				if cell == "" {
					continue
				}
				// Strip markdown bold
				cell = strings.ReplaceAll(cell, "**", "")
				sb.WriteString(fmt.Sprintf("<%s>%s</%s>", tag, cell, tag))
			}
			sb.WriteString("</tr>\n")
			if isFirstRow {
				isFirstRow = false
			}
		} else if strings.HasPrefix(line, "# ") {
			sb.WriteString(fmt.Sprintf("<h1>%s</h1>\n", line[2:]))
		} else if strings.HasPrefix(line, "## ") {
			sb.WriteString(fmt.Sprintf("<h2>%s</h2>\n", line[3:]))
		} else if strings.HasPrefix(line, "---") {
			sb.WriteString("<hr>\n")
		} else if strings.HasPrefix(line, "*Generated") {
			sb.WriteString(fmt.Sprintf("<p><em>%s</em></p>\n", strings.Trim(line, "*")))
		} else if line != "" {
			sb.WriteString(fmt.Sprintf("<p>%s</p>\n", line))
		}
		_ = i
	}
	if inTable {
		sb.WriteString("</table>\n")
	}
	sb.WriteString("</body>\n</html>\n")
	return sb.String()
}
