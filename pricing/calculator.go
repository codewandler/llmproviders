package pricing

import (
	"fmt"

	"github.com/codewandler/agentapis/api/unified"
)

// EstimateCost computes a simple estimated cost from unified usage.
func EstimateCost(u unified.StreamUsage, p Price) float64 {
	input := float64(u.Input.New) / 1_000_000 * p.InputPerMTok
	cacheRead := float64(u.Input.CacheRead) / 1_000_000 * p.CacheReadPerMTok
	cacheWrite := float64(u.Input.CacheWrite) / 1_000_000 * p.CacheWritePerMTok
	output := float64(u.Output.Total) / 1_000_000 * p.OutputPerMTok
	return input + cacheRead + cacheWrite + output
}

func Lookup(provider, model string) (Price, bool) {
	p, ok := Catalog[fmt.Sprintf("%s/%s", provider, model)]
	return p, ok
}
