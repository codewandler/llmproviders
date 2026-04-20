package openai

// NewWithAPIKey creates an OpenAI provider with an explicit API key.
// This is the recommended way to create a provider in server environments.
func NewWithAPIKey(apiKey string, opts ...Option) (*Provider, error) {
	return New(append([]Option{WithAPIKey(apiKey)}, opts...)...)
}

// MustNew creates an OpenAI provider or panics on error.
// This is useful for initialization in main() or global variables.
func MustNew(opts ...Option) *Provider {
	p, err := New(opts...)
	if err != nil {
		panic("openai: " + err.Error())
	}
	return p
}

// MustNewWithAPIKey creates an OpenAI provider with an explicit API key or panics on error.
func MustNewWithAPIKey(apiKey string, opts ...Option) *Provider {
	p, err := NewWithAPIKey(apiKey, opts...)
	if err != nil {
		panic("openai: " + err.Error())
	}
	return p
}
