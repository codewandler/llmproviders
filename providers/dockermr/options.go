package dockermr

import "net/http"

// Option configures a Docker Model Runner provider.
type Option func(*options)

type options struct {
	baseURL    string
	engine     string
	model      string
	httpClient *http.Client
}

func defaultOptions() *options {
	return &options{
		baseURL: DefaultBaseURL,
		engine:  DefaultEngine,
		model:   DefaultModel,
	}
}

// WithBaseURL sets a custom base URL for the Docker Model Runner API.
// Default: http://localhost:12434
//
// Use ContainerBaseURL ("http://model-runner.docker.internal") when
// running inside a Docker container.
func WithBaseURL(url string) Option {
	return func(o *options) {
		o.baseURL = url
	}
}

// WithEngine sets the inference engine to use.
// Default: llama.cpp
//
// The engine is appended to the base URL: {baseURL}/engines/{engine}
func WithEngine(engine string) Option {
	return func(o *options) {
		o.engine = engine
	}
}

// WithModel sets the default model for requests.
// Default: ai/smollm2
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
