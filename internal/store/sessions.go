package store

import (
	"fmt"
	"time"
)

// UpsertSession creates or updates a session.
func (s *Store) UpsertSession(id, cwd, project, branch, model string) error {
	now := time.Now().UnixMilli()
	_, err := s.db.Exec(`
		INSERT INTO sessions (id, cwd, project, branch, model, started_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			model = COALESCE(excluded.model, sessions.model),
			branch = COALESCE(excluded.branch, sessions.branch)
	`, id, cwd, project, branch, model, now)
	if err != nil {
		return fmt.Errorf("store: upsert session: %w", err)
	}
	return nil
}

// EndSession sets ended_at and end_reason on a session.
func (s *Store) EndSession(id, reason string) error {
	now := time.Now().UnixMilli()
	_, err := s.db.Exec(`
		UPDATE sessions SET ended_at = ?, end_reason = ? WHERE id = ?
	`, now, reason, id)
	if err != nil {
		return fmt.Errorf("store: end session: %w", err)
	}
	return nil
}

// IncrementPrompts increments total_prompts for a session.
func (s *Store) IncrementPrompts(sessionID string) error {
	_, err := s.db.Exec(`
		UPDATE sessions SET total_prompts = total_prompts + 1 WHERE id = ?
	`, sessionID)
	if err != nil {
		return fmt.Errorf("store: increment prompts: %w", err)
	}
	return nil
}

// UpdateTokenUsage sets token counts for a session.
func (s *Store) UpdateTokenUsage(sessionID string, inputTokens, outputTokens, cacheReadTokens, cacheCreationTokens int64) error {
	_, err := s.db.Exec(`
		UPDATE sessions SET
			total_input_tokens = ?,
			total_output_tokens = ?,
			total_cache_read_tokens = ?,
			total_cache_creation_tokens = ?
		WHERE id = ?
	`, inputTokens, outputTokens, cacheReadTokens, cacheCreationTokens, sessionID)
	if err != nil {
		return fmt.Errorf("store: update token usage: %w", err)
	}
	return nil
}

// IncrementToolCalls increments total_tool_calls for a session.
func (s *Store) IncrementToolCalls(sessionID string) error {
	_, err := s.db.Exec(`
		UPDATE sessions SET total_tool_calls = total_tool_calls + 1 WHERE id = ?
	`, sessionID)
	if err != nil {
		return fmt.Errorf("store: increment tool calls: %w", err)
	}
	return nil
}
