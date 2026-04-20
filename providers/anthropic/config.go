package anthropic

import "net/http"

const (
	// DefaultBaseURL is the Anthropic API base URL.
	DefaultBaseURL = "https://api.anthropic.com"

	// DefaultModel is the default model when none is specified.
	DefaultModel = ModelSonnet
)

// Config holds the provider configuration.
type Config struct {
	// BaseURL is the API base URL. Defaults to DefaultBaseURL.
	BaseURL string

	// Model is the default model to use. Defaults to DefaultModel.
	Model string

	// HTTPClient is the HTTP client for API requests.
	// If nil, http.DefaultClient is used.
	HTTPClient *http.Client
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL: DefaultBaseURL,
		Model:   DefaultModel,
	}
}

// httpClient returns the configured HTTP client or the default.
func (c Config) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
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
