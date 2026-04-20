package minimax

// NewWithAPIKey creates a new MiniMax provider with explicit API key authentication.
//
// Example:
//
//	p, err := minimax.NewWithAPIKey("your-api-key", minimax.WithModel("MiniMax-M2.7"))
func NewWithAPIKey(apiKey string, opts ...Option) (*Provider, error) {
	return New(append([]Option{WithAPIKey(apiKey)}, opts...)...)
}

// MustNew creates a new MiniMax provider, panicking on error.
// Use sparingly; prefer explicit error handling.
//
// Example:
//
//	p := minimax.MustNew(minimax.WithAPIKey("your-api-key"))
func MustNew(opts ...Option) *Provider {
	p, err := New(opts...)
	if err != nil {
		panic("minimax.MustNew: " + err.Error())
	}
	return p
}
