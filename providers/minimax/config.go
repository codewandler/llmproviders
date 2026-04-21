package minimax

import "net/http"

const (
	// DefaultBaseURL is the default MiniMax API endpoint.
	// MiniMax uses an Anthropic-compatible API at this endpoint.
	DefaultBaseURL = "https://api.minimax.io/anthropic"

	// ProviderName is the canonical name for this provider instance.
	ProviderName = "minimax"

	// ServiceID is the modeldb service identifier for this provider.
	ServiceID = "minimax"

	// DefaultModel is the default model when none is specified.
	DefaultModel = ModelM27

	// EnvAPIKey is the environment variable for the MiniMax API key.
	EnvAPIKey = "MINIMAX_API_KEY"

	// AnthropicVersion is the Anthropic API version header value.
	// MiniMax uses an Anthropic-compatible API.
	AnthropicVersion = "2023-06-01"

	// BetaInterleavedThinking is the beta feature flag for thinking support.
	BetaInterleavedThinking = "interleaved-thinking-2025-05-14"
)

// Config holds provider configuration.
type Config struct {
	// BaseURL is the API base URL.
	BaseURL string

	// Model is the default model to use.
	Model string

	// HTTPClient is the HTTP client for requests.
	HTTPClient *http.Client
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL: DefaultBaseURL,
		Model:   DefaultModel,
	}
}

// baseURL returns the configured base URL or the default.
func (c Config) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return DefaultBaseURL
}

// model returns the configured model or the default.
func (c Config) model() string {
	if c.Model != "" {
		return c.Model
	}
	return DefaultModel
}

// httpClient returns the configured HTTP client or the default.
func (c Config) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}
