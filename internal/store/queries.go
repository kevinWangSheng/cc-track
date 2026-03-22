package store

import (
	"database/sql"
	"fmt"
)

// SessionSummary holds aggregated session data.
type SessionSummary struct {
	TotalSessions           int         `json:"total_sessions"`
	TotalPrompts            int         `json:"total_prompts"`
	TotalToolCalls          int         `json:"total_tool_calls"`
	TotalDurationMs         int64       `json:"total_duration_ms"`
	TotalInputTokens        int64       `json:"total_input_tokens"`
	TotalOutputTokens       int64       `json:"total_output_tokens"`
	TotalCacheReadTokens    int64       `json:"total_cache_read_tokens"`
	TotalCacheCreationTokens int64      `json:"total_cache_creation_tokens"`
	ToolBreakdown           []ToolCount `json:"tool_breakdown"`
	ErrorRate               float64     `json:"error_rate"`
}

// ToolCount holds a tool name and its call count.
type ToolCount struct {
	Name    string  `json:"name"`
	Count   int     `json:"count"`
	Percent float64 `json:"percent"`
}

// QuerySummary returns aggregated stats for sessions in the given time range.
func (s *Store) QuerySummary(sinceMs, untilMs int64) (*SessionSummary, error) {
	sum := &SessionSummary{}

	err := s.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(total_prompts),0), COALESCE(SUM(total_tool_calls),0),
		       COALESCE(SUM(CASE WHEN ended_at IS NOT NULL THEN ended_at - started_at ELSE 0 END),0),
		       COALESCE(SUM(total_input_tokens),0), COALESCE(SUM(total_output_tokens),0),
		       COALESCE(SUM(total_cache_read_tokens),0), COALESCE(SUM(total_cache_creation_tokens),0)
		FROM sessions WHERE started_at >= ? AND started_at < ?
	`, sinceMs, untilMs).Scan(&sum.TotalSessions, &sum.TotalPrompts, &sum.TotalToolCalls, &sum.TotalDurationMs,
		&sum.TotalInputTokens, &sum.TotalOutputTokens, &sum.TotalCacheReadTokens, &sum.TotalCacheCreationTokens)
	if err != nil {
		return nil, fmt.Errorf("store: query summary: %w", err)
	}

	// Tool breakdown
	rows, err := s.db.Query(`
		SELECT tool_name, COUNT(*) as cnt
		FROM tool_calls tc
		JOIN sessions s ON tc.session_id = s.id
		WHERE s.started_at >= ? AND s.started_at < ?
		GROUP BY tool_name
		ORDER BY cnt DESC
	`, sinceMs, untilMs)
	if err != nil {
		return nil, fmt.Errorf("store: query tool breakdown: %w", err)
	}
	defer rows.Close()

	var total int
	for rows.Next() {
		var tc ToolCount
		if err := rows.Scan(&tc.Name, &tc.Count); err != nil {
			return nil, fmt.Errorf("store: scan tool count: %w", err)
		}
		total += tc.Count
		sum.ToolBreakdown = append(sum.ToolBreakdown, tc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: tool breakdown rows: %w", err)
	}

	for i := range sum.ToolBreakdown {
		if total > 0 {
			sum.ToolBreakdown[i].Percent = float64(sum.ToolBreakdown[i].Count) / float64(total) * 100
		}
	}

	// Error rate
	var failedCount int
	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM tool_calls tc
		JOIN sessions s ON tc.session_id = s.id
		WHERE s.started_at >= ? AND s.started_at < ? AND tc.succeeded = 0
	`, sinceMs, untilMs).Scan(&failedCount)
	if err != nil {
		return nil, fmt.Errorf("store: query error rate: %w", err)
	}
	if total > 0 {
		sum.ErrorRate = float64(failedCount) / float64(total) * 100
	}

	return sum, nil
}

// SessionRow holds a single session for listing.
type SessionRow struct {
	ID                       string         `json:"id"`
	CWD                      string         `json:"cwd"`
	Project                  sql.NullString `json:"-"`
	ProjectStr               string         `json:"project"`
	Branch                   sql.NullString `json:"-"`
	BranchStr                string         `json:"branch"`
	Model                    sql.NullString `json:"-"`
	ModelStr                 string         `json:"model"`
	StartedAt                int64          `json:"started_at"`
	EndedAt                  sql.NullInt64  `json:"-"`
	EndedAtVal               int64          `json:"ended_at"`
	TotalPrompts             int            `json:"total_prompts"`
	TotalToolCalls           int            `json:"total_tool_calls"`
	TotalInputTokens         int64          `json:"total_input_tokens"`
	TotalOutputTokens        int64          `json:"total_output_tokens"`
	TotalCacheReadTokens     int64          `json:"total_cache_read_tokens"`
	TotalCacheCreationTokens int64          `json:"total_cache_creation_tokens"`
}

func (r *SessionRow) fill() {
	r.ProjectStr = r.Project.String
	r.BranchStr = r.Branch.String
	r.ModelStr = r.Model.String
	if r.EndedAt.Valid {
		r.EndedAtVal = r.EndedAt.Int64
	}
}

// TotalTokens returns the sum of all token types.
func (r *SessionRow) TotalTokens() int64 {
	return r.TotalInputTokens + r.TotalOutputTokens + r.TotalCacheReadTokens + r.TotalCacheCreationTokens
}

// ListSessions returns recent sessions.
func (s *Store) ListSessions(limit int) ([]SessionRow, error) {
	rows, err := s.db.Query(`
		SELECT id, cwd, project, branch, model, started_at, ended_at, total_prompts, total_tool_calls,
		       COALESCE(total_input_tokens,0), COALESCE(total_output_tokens,0),
		       COALESCE(total_cache_read_tokens,0), COALESCE(total_cache_creation_tokens,0)
		FROM sessions ORDER BY started_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("store: list sessions: %w", err)
	}
	defer rows.Close()

	var result []SessionRow
	for rows.Next() {
		var r SessionRow
		if err := rows.Scan(&r.ID, &r.CWD, &r.Project, &r.Branch, &r.Model,
			&r.StartedAt, &r.EndedAt, &r.TotalPrompts, &r.TotalToolCalls,
			&r.TotalInputTokens, &r.TotalOutputTokens,
			&r.TotalCacheReadTokens, &r.TotalCacheCreationTokens); err != nil {
			return nil, fmt.Errorf("store: scan session: %w", err)
		}
		r.fill()
		result = append(result, r)
	}
	return result, rows.Err()
}

// FindSessionByPrefix finds a session by ID prefix.
func (s *Store) FindSessionByPrefix(prefix string) (*SessionRow, error) {
	rows, err := s.db.Query(`
		SELECT id, cwd, project, branch, model, started_at, ended_at, total_prompts, total_tool_calls,
		       COALESCE(total_input_tokens,0), COALESCE(total_output_tokens,0),
		       COALESCE(total_cache_read_tokens,0), COALESCE(total_cache_creation_tokens,0)
		FROM sessions WHERE id LIKE ? || '%' LIMIT 2
	`, prefix)
	if err != nil {
		return nil, fmt.Errorf("store: find session: %w", err)
	}
	defer rows.Close()

	var results []SessionRow
	for rows.Next() {
		var r SessionRow
		if err := rows.Scan(&r.ID, &r.CWD, &r.Project, &r.Branch, &r.Model,
			&r.StartedAt, &r.EndedAt, &r.TotalPrompts, &r.TotalToolCalls,
			&r.TotalInputTokens, &r.TotalOutputTokens,
			&r.TotalCacheReadTokens, &r.TotalCacheCreationTokens); err != nil {
			return nil, fmt.Errorf("store: scan session: %w", err)
		}
		r.fill()
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	switch len(results) {
	case 0:
		return nil, fmt.Errorf("store: session not found: %s", prefix)
	case 1:
		return &results[0], nil
	default:
		return nil, fmt.Errorf("store: ambiguous session prefix: %s", prefix)
	}
}

// ToolCallRow holds a single tool call for display.
type ToolCallRow struct {
	ToolName   string `json:"tool_name"`
	ToolUseID  string `json:"tool_use_id"`
	Succeeded  int    `json:"succeeded"`
	StartedAt  int64  `json:"started_at"`
	CompletedAt sql.NullInt64 `json:"-"`
	CompletedAtVal int64      `json:"completed_at"`
	DurationMs sql.NullInt64  `json:"-"`
	DurationMsVal int64       `json:"duration_ms"`
}

// PromptRow holds a single prompt for display.
type PromptRow struct {
	PromptText string `json:"prompt_text"`
	Timestamp  int64  `json:"timestamp"`
}

// StopEventRow holds a single stop event for display.
type StopEventRow struct {
	EventType    string `json:"event_type"`
	ErrorType    string `json:"error_type,omitempty"`
	ErrorDetails string `json:"error_details,omitempty"`
	Timestamp    int64  `json:"timestamp"`
}

// SessionTimeline holds all events for a session.
type SessionTimeline struct {
	Session    SessionRow     `json:"session"`
	Prompts    []PromptRow    `json:"prompts"`
	ToolCalls  []ToolCallRow  `json:"tool_calls"`
	StopEvents []StopEventRow `json:"stop_events"`
}

// GetSessionTimeline returns the full timeline for a session.
func (s *Store) GetSessionTimeline(sessionID string) (*SessionTimeline, error) {
	sess, err := s.FindSessionByPrefix(sessionID)
	if err != nil {
		return nil, err
	}

	tl := &SessionTimeline{Session: *sess}

	// Prompts
	pRows, err := s.db.Query(`
		SELECT prompt_text, timestamp FROM prompts
		WHERE session_id = ? ORDER BY timestamp
	`, sess.ID)
	if err != nil {
		return nil, fmt.Errorf("store: query prompts: %w", err)
	}
	defer pRows.Close()
	for pRows.Next() {
		var p PromptRow
		if err := pRows.Scan(&p.PromptText, &p.Timestamp); err != nil {
			return nil, fmt.Errorf("store: scan prompt: %w", err)
		}
		tl.Prompts = append(tl.Prompts, p)
	}

	// Tool calls
	tRows, err := s.db.Query(`
		SELECT tool_name, COALESCE(tool_use_id,''), succeeded, started_at, completed_at, duration_ms
		FROM tool_calls WHERE session_id = ? ORDER BY started_at
	`, sess.ID)
	if err != nil {
		return nil, fmt.Errorf("store: query tool calls: %w", err)
	}
	defer tRows.Close()
	for tRows.Next() {
		var tc ToolCallRow
		if err := tRows.Scan(&tc.ToolName, &tc.ToolUseID, &tc.Succeeded,
			&tc.StartedAt, &tc.CompletedAt, &tc.DurationMs); err != nil {
			return nil, fmt.Errorf("store: scan tool call: %w", err)
		}
		if tc.CompletedAt.Valid {
			tc.CompletedAtVal = tc.CompletedAt.Int64
		}
		if tc.DurationMs.Valid {
			tc.DurationMsVal = tc.DurationMs.Int64
		}
		tl.ToolCalls = append(tl.ToolCalls, tc)
	}

	// Stop events
	sRows, err := s.db.Query(`
		SELECT event_type, COALESCE(error_type,''), COALESCE(error_details,''), timestamp
		FROM stop_events WHERE session_id = ? ORDER BY timestamp
	`, sess.ID)
	if err != nil {
		return nil, fmt.Errorf("store: query stop events: %w", err)
	}
	defer sRows.Close()
	for sRows.Next() {
		var se StopEventRow
		if err := sRows.Scan(&se.EventType, &se.ErrorType, &se.ErrorDetails, &se.Timestamp); err != nil {
			return nil, fmt.Errorf("store: scan stop event: %w", err)
		}
		tl.StopEvents = append(tl.StopEvents, se)
	}

	return tl, nil
}

// ExportSession holds all data for export.
type ExportSession struct {
	SessionRow
	Prompts    []PromptRow    `json:"prompts"`
	ToolCalls  []ToolCallRow  `json:"tool_calls"`
	StopEvents []StopEventRow `json:"stop_events"`
}

// ExportSessions returns all sessions with their events since the given time.
func (s *Store) ExportSessions(sinceMs int64) ([]ExportSession, error) {
	sessions, err := s.db.Query(`
		SELECT id, cwd, project, branch, model, started_at, ended_at, total_prompts, total_tool_calls,
		       COALESCE(total_input_tokens,0), COALESCE(total_output_tokens,0),
		       COALESCE(total_cache_read_tokens,0), COALESCE(total_cache_creation_tokens,0)
		FROM sessions WHERE started_at >= ? ORDER BY started_at
	`, sinceMs)
	if err != nil {
		return nil, fmt.Errorf("store: export sessions: %w", err)
	}
	defer sessions.Close()

	var result []ExportSession
	for sessions.Next() {
		var r SessionRow
		if err := sessions.Scan(&r.ID, &r.CWD, &r.Project, &r.Branch, &r.Model,
			&r.StartedAt, &r.EndedAt, &r.TotalPrompts, &r.TotalToolCalls); err != nil {
			return nil, fmt.Errorf("store: scan export session: %w", err)
		}
		r.fill()
		result = append(result, ExportSession{SessionRow: r})
	}

	for i := range result {
		tl, err := s.GetSessionTimeline(result[i].ID)
		if err != nil {
			return nil, err
		}
		result[i].Prompts = tl.Prompts
		result[i].ToolCalls = tl.ToolCalls
		result[i].StopEvents = tl.StopEvents
	}

	return result, nil
}
