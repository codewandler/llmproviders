package openrouter

import (
	"net/http"

	"github.com/codewandler/agentapis/httpx"
)

// Option configures the OpenRouter provider.
type Option func(*providerOptions)

type providerOptions struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

func defaultOptions() *providerOptions {
	return &providerOptions{
		baseURL: DefaultBaseURL,
		model:   DefaultModel,
	}
}

// WithBaseURL sets the API base URL.
func WithBaseURL(url string) Option {
	return func(o *providerOptions) {
		o.baseURL = url
	}
}

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) Option {
	return func(o *providerOptions) {
		o.apiKey = key
	}
}

// WithModel sets the default model.
func WithModel(model string) Option {
	return func(o *providerOptions) {
		o.model = model
	}
}

// WithHTTPClient sets a custom HTTP client.
//
// Callers that want to preserve agentapis defaults while extending behavior
// should start from httpx.CloneDefaultClient().
func WithHTTPClient(client *http.Client) Option {
	return func(o *providerOptions) {
		o.httpClient = client
	}
}

// NewDefaultHTTPClient returns a clone of the default agentapis HTTP client.
//
// It supports gzip, deflate, br, and zstd and can be safely customized by callers.
func NewDefaultHTTPClient() *http.Client {
	return httpx.CloneDefaultClient()
}
