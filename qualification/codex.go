package qualification

import (
	"github.com/codewandler/llmproviders/pricing"
	"github.com/codewandler/llmproviders/providers/codex"
)

func CodexEntries() []Entry {
	models := codex.CatalogModels()
	entries := make([]Entry, 0, len(models))
	for _, m := range models {
		d := Dimensions{
			Tools:        true,
			Caching:      false,
			Pricing:      hasPricing(m.ID),
			Thinking:     hasCapability(m.Capabilities, "thinking"),
			Conversation: true,
			CodingLoop:   true,
			Usage:        hasCapability(m.Capabilities, "usage"),
		}
		notes := []string{"responses-based codex runtime scaffold implemented"}
		if !d.Caching {
			notes = append(notes, "prompt caching not yet qualified / not declared supported")
		}
		entries = append(entries, Entry{
			Provider:   m.Provider,
			Service:    m.Service,
			Model:      m.ID,
			Status:     DeriveStatus(d),
			Dimensions: d,
			Notes:      notes,
		})
	}
	return entries
}

func hasPricing(model string) bool {
	_, ok := pricing.Lookup("codex", model)
	return ok
}

func hasCapability(caps []string, want string) bool {
	for _, cap := range caps {
		if cap == want {
			return true
		}
	}
	return false
}
