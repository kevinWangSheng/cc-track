package analysis

import "strings"

// TokenPricing holds per-million-token prices in USD.
type TokenPricing struct {
	InputPerMillion        float64 `json:"input_per_million"`
	OutputPerMillion       float64 `json:"output_per_million"`
	CacheReadPerMillion    float64 `json:"cache_read_per_million"`
	CacheCreationPerMillion float64 `json:"cache_creation_per_million"`
}

// Pricing table as of 2025-03. Prices are per million tokens in USD.
// Source: https://docs.anthropic.com/en/docs/about-claude/models
var modelPricing = map[string]TokenPricing{
	"opus": {
		InputPerMillion:         15.0,
		OutputPerMillion:        75.0,
		CacheReadPerMillion:     1.5,
		CacheCreationPerMillion: 18.75,
	},
	"sonnet": {
		InputPerMillion:         3.0,
		OutputPerMillion:        15.0,
		CacheReadPerMillion:     0.3,
		CacheCreationPerMillion: 3.75,
	},
	"haiku": {
		InputPerMillion:         0.8,
		OutputPerMillion:        4.0,
		CacheReadPerMillion:     0.08,
		CacheCreationPerMillion: 1.0,
	},
}

// defaultPricing is used when model cannot be identified. Uses Sonnet pricing as a reasonable middle ground.
var defaultPricing = modelPricing["sonnet"]

// LookupPricing returns pricing for a model string.
// Handles various formats: "opus", "claude-opus-4-6[1m]", "claude-sonnet-4-5-20241022", etc.
func LookupPricing(model string) TokenPricing {
	m := strings.ToLower(model)
	for key, pricing := range modelPricing {
		if strings.Contains(m, key) {
			return pricing
		}
	}
	return defaultPricing
}

// CostBreakdown holds calculated costs in USD.
type CostBreakdown struct {
	InputCost         float64 `json:"input_cost"`
	OutputCost        float64 `json:"output_cost"`
	CacheReadCost     float64 `json:"cache_read_cost"`
	CacheCreationCost float64 `json:"cache_creation_cost"`
	TotalCost         float64 `json:"total_cost"`
}

// CalculateCost computes the cost for given token counts and pricing.
func CalculateCost(inputTokens, outputTokens, cacheReadTokens, cacheCreationTokens int64, pricing TokenPricing) CostBreakdown {
	c := CostBreakdown{
		InputCost:         float64(inputTokens) / 1_000_000 * pricing.InputPerMillion,
		OutputCost:        float64(outputTokens) / 1_000_000 * pricing.OutputPerMillion,
		CacheReadCost:     float64(cacheReadTokens) / 1_000_000 * pricing.CacheReadPerMillion,
		CacheCreationCost: float64(cacheCreationTokens) / 1_000_000 * pricing.CacheCreationPerMillion,
	}
	c.TotalCost = c.InputCost + c.OutputCost + c.CacheReadCost + c.CacheCreationCost
	return c
}
