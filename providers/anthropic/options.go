package anthropic

import (
	"context"
	"net/http"

	messagesapi "github.com/codewandler/agentapis/api/messages"
)

// RateLimits contains parsed rate limit information from Anthropic API response headers.
// Re-exported from agentapis/api/messages for convenience.
type RateLimits = messagesapi.RateLimits

// Option configures the Provider.
type Option func(*providerOptions)

// RateLimitCallback is called with rate limit information after each API response.
type RateLimitCallback func(RateLimits)

// providerOptions holds all provider configuration.
type providerOptions struct {
	// Config
	baseURL    string
	model      string
	httpClient *http.Client

	// Auth
	auth          Auth
	claudeHeaders bool

	// Behavior
	autoSystemCacheControl bool
	autoSystemCacheTTL     string
	rateLimitCallback      RateLimitCallback
}

func defaultOptions() *providerOptions {
	return &providerOptions{
		baseURL:            DefaultBaseURL,
		model:              DefaultModel,
		autoSystemCacheTTL: "1h",
	}
}

// --- Config options ---

// WithBaseURL sets the API base URL.
func WithBaseURL(url string) Option {
	return func(o *providerOptions) {
		o.baseURL = url
	}
}

// WithModel sets the default model.
func WithModel(model string) Option {
	return func(o *providerOptions) {
		o.model = model
	}
}

// WithHTTPClient sets the HTTP client for API requests.
func WithHTTPClient(c *http.Client) Option {
	return func(o *providerOptions) {
		o.httpClient = c
	}
}

// --- Auth options ---

// WithAPIKey sets authentication using a static API key.
func WithAPIKey(key string) Option {
	return func(o *providerOptions) {
		o.auth = NewAPIKeyAuth(key)
	}
}

// WithAuth sets a custom Auth implementation.
func WithAuth(auth Auth) Option {
	return func(o *providerOptions) {
		o.auth = auth
	}
}

// WithOAuth sets authentication using an OAuth token provider.
func WithOAuth(tp TokenProvider) Option {
	return func(o *providerOptions) {
		o.auth = NewOAuthAuth(tp)
	}
}

// localOAuthMarker is a marker type to signal that local OAuth should be used.
type localOAuthMarker struct{}

func (localOAuthMarker) ApplyAuth(ctx context.Context, h http.Header) error {
	return nil // Never called; just a marker
}

// WithLocalOAuth sets authentication using ~/.claude/.credentials.json.
// Returns an error during New() if the credentials file doesn't exist.
func WithLocalOAuth() Option {
	return func(o *providerOptions) {
		o.auth = localOAuthMarker{}
	}
}

// WithClaudeHeaders enables Claude CLI compatibility headers.
// Only applies when using OAuth authentication.
func WithClaudeHeaders(enabled bool) Option {
	return func(o *providerOptions) {
		o.claudeHeaders = enabled
	}
}

// --- Behavior options ---

// WithAutoSystemCacheControl enables automatic cache_control on the last system block.
// The ttl parameter specifies the cache TTL (e.g., "1h", "30m"). Empty defaults to "1h".
func WithAutoSystemCacheControl(ttl string) Option {
	return func(o *providerOptions) {
		o.autoSystemCacheControl = true
		if ttl != "" {
			o.autoSystemCacheTTL = ttl
		}
	}
}

// WithRateLimitCallback sets a callback function that will be invoked with rate limit
// information after each API response. This allows monitoring API usage and
// implementing custom rate limiting strategies.
//
// Example:
//
//	p, err := anthropic.New(
//	    anthropic.WithAPIKey("sk-ant-..."),
//	    anthropic.WithRateLimitCallback(func(rl anthropic.RateLimits) {
//	        if rl.TokenUtilization() > 0.8 {
//	            log.Println("Warning: High token usage")
//	        }
//	    }),
//	)
func WithRateLimitCallback(cb RateLimitCallback) Option {
	return func(o *providerOptions) {
		o.rateLimitCallback = cb
	}
}

// applyOptions applies all options to the default options.
func applyOptions(opts []Option) *providerOptions {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	return o
}
