package openrouter

import "net/http"

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
func WithHTTPClient(client *http.Client) Option {
	return func(o *providerOptions) {
		o.httpClient = client
	}
}
