package ollama

import "net/http"

// Option configures an Ollama provider.
type Option func(*options)

type options struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

func defaultOptions() *options {
	return &options{
		baseURL: DefaultBaseURL,
		model:   DefaultModel,
	}
}

// WithBaseURL sets a custom base URL for the Ollama API.
// Default: http://localhost:11434
func WithBaseURL(url string) Option {
	return func(o *options) {
		o.baseURL = url
	}
}

// WithModel sets the default model for requests.
// Default: qwen2.5:0.5b
func WithModel(model string) Option {
	return func(o *options) {
		o.model = model
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(o *options) {
		o.httpClient = client
	}
}
