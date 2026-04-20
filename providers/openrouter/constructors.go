package openrouter

// NewWithAPIKey creates an OpenRouter provider with an explicit API key.
func NewWithAPIKey(apiKey string, opts ...Option) (*Provider, error) {
	return New(append([]Option{WithAPIKey(apiKey)}, opts...)...)
}

// MustNew creates an OpenRouter provider or panics on error.
func MustNew(opts ...Option) *Provider {
	p, err := New(opts...)
	if err != nil {
		panic("openrouter: " + err.Error())
	}
	return p
}

// MustNewWithAPIKey creates an OpenRouter provider with an explicit API key or panics on error.
func MustNewWithAPIKey(apiKey string, opts ...Option) *Provider {
	p, err := NewWithAPIKey(apiKey, opts...)
	if err != nil {
		panic("openrouter: " + err.Error())
	}
	return p
}
