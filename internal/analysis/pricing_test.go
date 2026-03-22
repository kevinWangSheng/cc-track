package analysis

import (
	"math"
	"testing"
)

func TestLookupPricing_ExactMatch(t *testing.T) {
	p := LookupPricing("opus")
	if p.InputPerMillion != 15.0 {
		t.Fatalf("expected opus input price 15.0, got %f", p.InputPerMillion)
	}
}

func TestLookupPricing_FullModelName(t *testing.T) {
	p := LookupPricing("claude-opus-4-6[1m]")
	if p.InputPerMillion != 15.0 {
		t.Fatalf("expected opus pricing for full name, got input=%f", p.InputPerMillion)
	}
}

func TestLookupPricing_Sonnet(t *testing.T) {
	p := LookupPricing("claude-sonnet-4-5-20241022")
	if p.InputPerMillion != 3.0 {
		t.Fatalf("expected sonnet input price 3.0, got %f", p.InputPerMillion)
	}
}

func TestLookupPricing_Haiku(t *testing.T) {
	p := LookupPricing("claude-haiku-4-5")
	if p.InputPerMillion != 0.8 {
		t.Fatalf("expected haiku input price 0.8, got %f", p.InputPerMillion)
	}
}

func TestLookupPricing_Unknown_DefaultsToSonnet(t *testing.T) {
	p := LookupPricing("unknown-model")
	if p.InputPerMillion != 3.0 {
		t.Fatalf("expected default (sonnet) pricing, got input=%f", p.InputPerMillion)
	}
}

func TestCalculateCost(t *testing.T) {
	pricing := TokenPricing{
		InputPerMillion:         15.0,
		OutputPerMillion:        75.0,
		CacheReadPerMillion:     1.5,
		CacheCreationPerMillion: 18.75,
	}

	c := CalculateCost(1_000_000, 100_000, 500_000, 200_000, pricing)

	assertClose(t, "input", c.InputCost, 15.0)
	assertClose(t, "output", c.OutputCost, 7.5)
	assertClose(t, "cache_read", c.CacheReadCost, 0.75)
	assertClose(t, "cache_creation", c.CacheCreationCost, 3.75)
	assertClose(t, "total", c.TotalCost, 27.0)
}

func TestCalculateCost_ZeroTokens(t *testing.T) {
	c := CalculateCost(0, 0, 0, 0, LookupPricing("opus"))
	if c.TotalCost != 0 {
		t.Fatalf("expected 0 cost for 0 tokens, got %f", c.TotalCost)
	}
}

func assertClose(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.001 {
		t.Errorf("%s: expected %.4f, got %.4f", name, want, got)
	}
}
