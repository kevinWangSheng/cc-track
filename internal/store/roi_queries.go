package store

import (
	"fmt"
)

// ROISessionData holds session-level data for ROI calculation.
type ROISessionData struct {
	TotalSessions  int    `json:"total_sessions"`
	TotalPrompts   int    `json:"total_prompts"`
	TotalToolCalls int    `json:"total_tool_calls"`
	TotalDurationMs int64 `json:"total_duration_ms"`
	Repos          []string `json:"repos"`
}

// QueryROISessions returns aggregated session data and distinct repo paths for a time range.
func (s *Store) QueryROISessions(sinceMs, untilMs int64) (*ROISessionData, error) {
	data := &ROISessionData{}

	err := s.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(total_prompts),0), COALESCE(SUM(total_tool_calls),0),
		       COALESCE(SUM(CASE WHEN ended_at IS NOT NULL THEN ended_at - started_at ELSE 0 END),0)
		FROM sessions WHERE started_at >= ? AND started_at < ?
	`, sinceMs, untilMs).Scan(&data.TotalSessions, &data.TotalPrompts, &data.TotalToolCalls, &data.TotalDurationMs)
	if err != nil {
		return nil, fmt.Errorf("store: query roi sessions: %w", err)
	}

	rows, err := s.db.Query(`
		SELECT DISTINCT cwd FROM sessions
		WHERE started_at >= ? AND started_at < ? AND cwd != ''
	`, sinceMs, untilMs)
	if err != nil {
		return nil, fmt.Errorf("store: query roi repos: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cwd string
		if err := rows.Scan(&cwd); err != nil {
			return nil, fmt.Errorf("store: scan roi repo: %w", err)
		}
		data.Repos = append(data.Repos, cwd)
	}
	return data, rows.Err()
}
