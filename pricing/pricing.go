package pricing

// Price stores provider/model-specific per-million-token pricing.
type Price struct {
	InputPerMTok      float64
	OutputPerMTok     float64
	CacheReadPerMTok  float64
	CacheWritePerMTok float64
	ReasoningIncluded bool
}
