package store

import "fmt"

// DailyStats holds aggregated stats for a single day.
type DailyStats struct {
	Date                     string `json:"date"` // YYYY-MM-DD
	Sessions                 int    `json:"sessions"`
	Prompts                  int    `json:"prompts"`
	ToolCalls                int    `json:"tool_calls"`
	DurationMs               int64  `json:"duration_ms"`
	InputTokens              int64  `json:"input_tokens"`
	OutputTokens             int64  `json:"output_tokens"`
	CacheReadTokens          int64  `json:"cache_read_tokens"`
	CacheCreationTokens      int64  `json:"cache_creation_tokens"`
	FailedToolCalls          int    `json:"failed_tool_calls"`
}

// TotalTokens returns the sum of all token types.
func (d *DailyStats) TotalTokens() int64 {
	return d.InputTokens + d.OutputTokens + d.CacheReadTokens + d.CacheCreationTokens
}

// QueryDailyStats returns per-day aggregated stats for a time range.
func (s *Store) QueryDailyStats(sinceMs, untilMs int64) ([]DailyStats, error) {
	rows, err := s.db.Query(`
		SELECT date(started_at / 1000, 'unixepoch', 'localtime') as day,
		       COUNT(*),
		       COALESCE(SUM(total_prompts), 0),
		       COALESCE(SUM(total_tool_calls), 0),
		       COALESCE(SUM(CASE WHEN ended_at IS NOT NULL THEN ended_at - started_at ELSE 0 END), 0),
		       COALESCE(SUM(total_input_tokens), 0),
		       COALESCE(SUM(total_output_tokens), 0),
		       COALESCE(SUM(total_cache_read_tokens), 0),
		       COALESCE(SUM(total_cache_creation_tokens), 0)
		FROM sessions
		WHERE started_at >= ? AND started_at < ?
		GROUP BY day
		ORDER BY day
	`, sinceMs, untilMs)
	if err != nil {
		return nil, fmt.Errorf("store: query daily stats: %w", err)
	}
	defer rows.Close()

	var result []DailyStats
	for rows.Next() {
		var d DailyStats
		if err := rows.Scan(&d.Date, &d.Sessions, &d.Prompts, &d.ToolCalls,
			&d.DurationMs, &d.InputTokens, &d.OutputTokens,
			&d.CacheReadTokens, &d.CacheCreationTokens); err != nil {
			return nil, fmt.Errorf("store: scan daily stats: %w", err)
		}
		result = append(result, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fill in failed tool calls per day
	failRows, err := s.db.Query(`
		SELECT date(s.started_at / 1000, 'unixepoch', 'localtime') as day,
		       COUNT(*)
		FROM tool_calls tc
		JOIN sessions s ON tc.session_id = s.id
		WHERE s.started_at >= ? AND s.started_at < ? AND tc.succeeded = 0
		GROUP BY day
	`, sinceMs, untilMs)
	if err != nil {
		return result, nil // best-effort
	}
	defer failRows.Close()

	failMap := make(map[string]int)
	for failRows.Next() {
		var day string
		var count int
		if err := failRows.Scan(&day, &count); err != nil {
			continue
		}
		failMap[day] = count
	}
	for i := range result {
		result[i].FailedToolCalls = failMap[result[i].Date]
	}

	return result, nil
}

// QueryWasteCountByDay returns the number of waste findings per day.
// This requires running waste analysis, so we approximate by counting
// failed tool calls and duplicate patterns from the DB directly.
func (s *Store) QueryDailySessionIDs(sinceMs, untilMs int64) (map[string][]string, error) {
	rows, err := s.db.Query(`
		SELECT date(started_at / 1000, 'unixepoch', 'localtime') as day, id
		FROM sessions
		WHERE started_at >= ? AND started_at < ?
		ORDER BY day
	`, sinceMs, untilMs)
	if err != nil {
		return nil, fmt.Errorf("store: query daily session ids: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var day, id string
		if err := rows.Scan(&day, &id); err != nil {
			return nil, fmt.Errorf("store: scan daily session id: %w", err)
		}
		result[day] = append(result[day], id)
	}
	return result, rows.Err()
}
