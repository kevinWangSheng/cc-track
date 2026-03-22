package store

import (
	"database/sql"
	"fmt"
)

// WasteToolCall holds a tool call with input for waste analysis.
type WasteToolCall struct {
	ID            int64  `json:"id"`
	SessionID     string `json:"session_id"`
	ToolUseID     string `json:"tool_use_id"`
	ToolName      string `json:"tool_name"`
	ToolInputJSON string `json:"tool_input_json"`
	Succeeded     int    `json:"succeeded"`
	StartedAt     int64  `json:"started_at"`
}

// WasteSession holds session-level data for zombie detection.
type WasteSession struct {
	ID             string `json:"id"`
	Project        string `json:"project"`
	StartedAt      int64  `json:"started_at"`
	EndedAt        int64  `json:"ended_at"`
	DurationMs     int64  `json:"duration_ms"`
	TotalPrompts   int    `json:"total_prompts"`
	TotalToolCalls int    `json:"total_tool_calls"`
}

// GetToolCallsForSession returns tool calls with input JSON for a session, ordered by started_at.
func (s *Store) GetToolCallsForSession(sessionID string) ([]WasteToolCall, error) {
	rows, err := s.db.Query(`
		SELECT id, session_id, COALESCE(tool_use_id,''), tool_name,
		       COALESCE(tool_input_json,''), succeeded, started_at
		FROM tool_calls
		WHERE session_id = ?
		ORDER BY started_at
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("store: get tool calls for waste: %w", err)
	}
	defer rows.Close()

	var result []WasteToolCall
	for rows.Next() {
		var tc WasteToolCall
		if err := rows.Scan(&tc.ID, &tc.SessionID, &tc.ToolUseID, &tc.ToolName,
			&tc.ToolInputJSON, &tc.Succeeded, &tc.StartedAt); err != nil {
			return nil, fmt.Errorf("store: scan waste tool call: %w", err)
		}
		result = append(result, tc)
	}
	return result, rows.Err()
}

// GetRecentSessionIDs returns session IDs for the last N sessions.
func (s *Store) GetRecentSessionIDs(limit int) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT id FROM sessions ORDER BY started_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("store: get recent session ids: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("store: scan session id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// GetZombieCandidates returns sessions that ended and lasted >30min.
func (s *Store) GetZombieCandidates(sessionIDs []string) ([]WasteSession, error) {
	if len(sessionIDs) == 0 {
		return nil, nil
	}

	query := `
		SELECT id, COALESCE(project,''), started_at, COALESCE(ended_at, started_at),
		       COALESCE(ended_at - started_at, 0), total_prompts, total_tool_calls
		FROM sessions
		WHERE id IN (`
	args := make([]interface{}, len(sessionIDs))
	for i, id := range sessionIDs {
		if i > 0 {
			query += ","
		}
		query += "?"
		args[i] = id
	}
	query += `) AND ended_at IS NOT NULL AND (ended_at - started_at) > 1800000`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: get zombie candidates: %w", err)
	}
	defer rows.Close()

	var result []WasteSession
	for rows.Next() {
		var ws WasteSession
		var endedAt sql.NullInt64
		if err := rows.Scan(&ws.ID, &ws.Project, &ws.StartedAt, &endedAt,
			&ws.DurationMs, &ws.TotalPrompts, &ws.TotalToolCalls); err != nil {
			return nil, fmt.Errorf("store: scan zombie session: %w", err)
		}
		if endedAt.Valid {
			ws.EndedAt = endedAt.Int64
		}
		result = append(result, ws)
	}
	return result, rows.Err()
}
