package anthropic

// NewWithAPIKey creates a new Anthropic provider with explicit API key authentication.
// This is the recommended constructor for production use with API keys.
//
// Example:
//
//	p, err := anthropic.NewWithAPIKey("sk-ant-...", anthropic.WithModel("claude-sonnet-4-20250514"))
func NewWithAPIKey(apiKey string, opts ...Option) (*Provider, error) {
	return New(append([]Option{WithAPIKey(apiKey)}, opts...)...)
}

// NewWithOAuth creates a new Anthropic provider with local OAuth authentication.
// Uses ~/.claude/.credentials.json for token storage.
// Returns an error if local OAuth credentials are not available.
//
// Example:
//
//	p, err := anthropic.NewWithOAuth(anthropic.WithClaudeHeaders(true))
func NewWithOAuth(opts ...Option) (*Provider, error) {
	return New(append([]Option{WithLocalOAuth()}, opts...)...)
}

// NewWithOAuthAndClaudeHeaders creates a new Anthropic provider with local OAuth
// and Claude CLI compatibility headers enabled.
// This is a convenience function for Claude CLI-style OAuth usage.
//
// Example:
//
//	p, err := anthropic.NewWithOAuthAndClaudeHeaders()
func NewWithOAuthAndClaudeHeaders(opts ...Option) (*Provider, error) {
	return New(append([]Option{WithLocalOAuth(), WithClaudeHeaders(true)}, opts...)...)
}

// MustNew creates a new Anthropic provider, panicking on error.
// Use sparingly; prefer explicit error handling.
//
// Example:
//
//	p := anthropic.MustNew(anthropic.WithAPIKey("sk-ant-..."))
func MustNew(opts ...Option) *Provider {
	p, err := New(opts...)
	if err != nil {
		panic("anthropic.MustNew: " + err.Error())
	}
	return p
}
