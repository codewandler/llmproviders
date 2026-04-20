package pricing

// Catalog is a placeholder provider/model -> price registry.
var Catalog = map[string]Price{
	"codex/gpt-5.4": {
		InputPerMTok:      1.25,
		OutputPerMTok:     10.00,
		CacheReadPerMTok:  0.125,
		CacheWritePerMTok: 1.25,
		ReasoningIncluded: true,
	},
	"codex/gpt-5.4-mini": {
		InputPerMTok:      0.25,
		OutputPerMTok:     2.00,
		CacheReadPerMTok:  0.025,
		CacheWritePerMTok: 0.25,
		ReasoningIncluded: true,
	},
}
