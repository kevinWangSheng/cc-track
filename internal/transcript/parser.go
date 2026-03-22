package transcript

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// TokenUsage holds aggregated token counts from a transcript.
type TokenUsage struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
}

// transcriptLine is the minimal structure we need from each JSONL line.
type transcriptLine struct {
	Type    string           `json:"type"`
	Message *transcriptMsg   `json:"message,omitempty"`
}

type transcriptMsg struct {
	Role  string     `json:"role"`
	Usage *usageData `json:"usage,omitempty"`
}

type usageData struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
}

// ParseFile reads a transcript JSONL file and sums token usage from assistant messages.
// It deduplicates by message ID to avoid double-counting streaming chunks.
func ParseFile(path string) (*TokenUsage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("transcript: open: %w", err)
	}
	defer f.Close()

	// Track seen message IDs to deduplicate streaming chunks.
	// Assistant messages may appear multiple times (streaming updates) with the same ID.
	type msgKey struct {
		id string
	}
	seen := make(map[msgKey]*usageData)

	scanner := bufio.NewScanner(f)
	// Transcript lines can be large (tool outputs); increase buffer.
	scanner.Buffer(make([]byte, 0, 256*1024), 2*1024*1024)

	for scanner.Scan() {
		var line transcriptLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue // skip unparseable lines
		}
		if line.Type != "assistant" || line.Message == nil || line.Message.Usage == nil {
			continue
		}
		// Extract message ID for dedup
		var raw struct {
			Message struct {
				ID string `json:"id"`
			} `json:"message"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &raw); err != nil {
			continue
		}
		id := raw.Message.ID
		if id == "" {
			continue
		}
		// Keep the latest usage for each message ID (later lines have more complete data)
		seen[msgKey{id}] = line.Message.Usage
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("transcript: scan: %w", err)
	}

	total := &TokenUsage{}
	for _, u := range seen {
		total.InputTokens += u.InputTokens
		total.OutputTokens += u.OutputTokens
		total.CacheCreationInputTokens += u.CacheCreationInputTokens
		total.CacheReadInputTokens += u.CacheReadInputTokens
	}

	return total, nil
}
