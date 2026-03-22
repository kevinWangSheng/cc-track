package store

import "fmt"

// ModelTokens holds token counts grouped by model.
type ModelTokens struct {
	Model                    string `json:"model"`
	TotalInputTokens         int64  `json:"total_input_tokens"`
	TotalOutputTokens        int64  `json:"total_output_tokens"`
	TotalCacheReadTokens     int64  `json:"total_cache_read_tokens"`
	TotalCacheCreationTokens int64  `json:"total_cache_creation_tokens"`
}

// QueryTokensByModel returns token counts grouped by model for a time range.
func (s *Store) QueryTokensByModel(sinceMs, untilMs int64) ([]ModelTokens, error) {
	rows, err := s.db.Query(`
		SELECT COALESCE(model,'unknown'),
		       COALESCE(SUM(total_input_tokens),0),
		       COALESCE(SUM(total_output_tokens),0),
		       COALESCE(SUM(total_cache_read_tokens),0),
		       COALESCE(SUM(total_cache_creation_tokens),0)
		FROM sessions
		WHERE started_at >= ? AND started_at < ?
		GROUP BY model
	`, sinceMs, untilMs)
	if err != nil {
		return nil, fmt.Errorf("store: query tokens by model: %w", err)
	}
	defer rows.Close()

	var result []ModelTokens
	for rows.Next() {
		var mt ModelTokens
		if err := rows.Scan(&mt.Model, &mt.TotalInputTokens, &mt.TotalOutputTokens,
			&mt.TotalCacheReadTokens, &mt.TotalCacheCreationTokens); err != nil {
			return nil, fmt.Errorf("store: scan model tokens: %w", err)
		}
		result = append(result, mt)
	}
	return result, rows.Err()
}
