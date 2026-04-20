package minimax

import "net/http"

// Option configures the Provider.
type Option func(*providerOptions)

// providerOptions holds all provider configuration.
type providerOptions struct {
	// Config
	baseURL    string
	model      string
	httpClient *http.Client

	// Auth
	auth Auth
}

func defaultOptions() *providerOptions {
	return &providerOptions{
		baseURL: DefaultBaseURL,
		model:   DefaultModel,
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

// applyOptions applies all options to the default options.
func applyOptions(opts []Option) *providerOptions {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	return o
}
