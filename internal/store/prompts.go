package store

import (
	"fmt"
	"time"
)

// InsertPrompt records a user prompt.
func (s *Store) InsertPrompt(sessionID, promptText string) error {
	now := time.Now().UnixMilli()
	_, err := s.db.Exec(`
		INSERT INTO prompts (session_id, prompt_text, timestamp)
		VALUES (?, ?, ?)
	`, sessionID, truncate(promptText, maxFieldBytes), now)
	if err != nil {
		return fmt.Errorf("store: insert prompt: %w", err)
	}
	return nil
}
