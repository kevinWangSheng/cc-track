package store

import (
	"fmt"
	"time"
)

// InsertToolCall records a PreToolUse event (partial row).
func (s *Store) InsertToolCall(sessionID, toolUseID, toolName, toolInput string) error {
	now := time.Now().UnixMilli()
	_, err := s.db.Exec(`
		INSERT INTO tool_calls (session_id, tool_use_id, tool_name, tool_input_json, started_at)
		VALUES (?, ?, ?, ?, ?)
	`, sessionID, toolUseID, toolName, truncate(toolInput, maxFieldBytes), now)
	if err != nil {
		return fmt.Errorf("store: insert tool call: %w", err)
	}
	return nil
}

// CompleteToolCall updates an existing tool call with output, or inserts a full row if not found.
func (s *Store) CompleteToolCall(sessionID, toolUseID, toolName, toolInput, toolOutput string) error {
	now := time.Now().UnixMilli()
	result, err := s.db.Exec(`
		UPDATE tool_calls
		SET tool_output_json = ?, completed_at = ?,
		    duration_ms = ? - started_at, succeeded = 1
		WHERE tool_use_id = ?
	`, truncate(toolOutput, maxFieldBytes), now, now, toolUseID)
	if err != nil {
		return fmt.Errorf("store: complete tool call: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		// PreToolUse hasn't arrived yet — insert full row
		_, err := s.db.Exec(`
			INSERT INTO tool_calls (session_id, tool_use_id, tool_name, tool_input_json, tool_output_json, started_at, completed_at, succeeded)
			VALUES (?, ?, ?, ?, ?, ?, ?, 1)
		`, sessionID, toolUseID, toolName,
			truncate(toolInput, maxFieldBytes),
			truncate(toolOutput, maxFieldBytes),
			now, now)
		if err != nil {
			return fmt.Errorf("store: insert full tool call: %w", err)
		}
	}
	return nil
}

// FailToolCall marks a tool call as failed.
func (s *Store) FailToolCall(toolUseID, errorMsg string) error {
	now := time.Now().UnixMilli()
	result, err := s.db.Exec(`
		UPDATE tool_calls
		SET succeeded = 0, error_message = ?, completed_at = ?,
		    duration_ms = ? - started_at
		WHERE tool_use_id = ?
	`, errorMsg, now, now, toolUseID)
	if err != nil {
		return fmt.Errorf("store: fail tool call: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		// PreToolUse hasn't arrived yet
		_, err := s.db.Exec(`
			INSERT INTO tool_calls (session_id, tool_use_id, tool_name, error_message, succeeded, started_at, completed_at)
			VALUES ('', ?, 'unknown', ?, 0, ?, ?)
		`, toolUseID, errorMsg, now, now)
		if err != nil {
			return fmt.Errorf("store: insert failed tool call: %w", err)
		}
	}
	return nil
}

// InsertStopEvent records a stop event.
func (s *Store) InsertStopEvent(sessionID, eventType, errorType, errorDetails string) error {
	now := time.Now().UnixMilli()
	_, err := s.db.Exec(`
		INSERT INTO stop_events (session_id, event_type, error_type, error_details, timestamp)
		VALUES (?, ?, ?, ?, ?)
	`, sessionID, eventType, errorType, errorDetails, now)
	if err != nil {
		return fmt.Errorf("store: insert stop event: %w", err)
	}
	return nil
}
