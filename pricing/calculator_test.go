package pricing

import (
	"testing"

	"github.com/codewandler/agentapis/api/unified"
)

func TestLookupCodexPricing(t *testing.T) {
	if _, ok := Lookup("codex", "gpt-5.4"); !ok {
		t.Fatal("expected codex/gpt-5.4 pricing")
	}
}

func TestEstimateCost(t *testing.T) {
	p, ok := Lookup("codex", "gpt-5.4-mini")
	if !ok {
		t.Fatal("expected pricing")
	}
	cost := EstimateCost(unified.StreamUsage{
		Input:  unified.InputTokens{New: 1000, CacheRead: 500, CacheWrite: 500},
		Output: unified.OutputTokens{Total: 2000},
	}, p)
	if cost <= 0 {
		t.Fatal("expected positive cost")
	}
}
