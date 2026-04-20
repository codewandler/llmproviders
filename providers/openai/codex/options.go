package codex

import "net/http"

// Option configures a Codex provider.
type Option func(*options)

type options struct {
	baseURL        string
	model          string
	httpClient     *http.Client
	installationID string
	betaFeatures   string
}

func defaultOptions() *options {
	return &options{
		baseURL: DefaultBaseURL,
		model:   DefaultModel,
	}
}

// WithBaseURL sets a custom base URL for the Codex API.
// Default: https://chatgpt.com/backend-api/codex/responses
func WithBaseURL(url string) Option {
	return func(o *options) {
		o.baseURL = url
	}
}

// WithModel sets the default model for requests.
// Default: "codex"
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

// WithInstallationID sets the x-codex-installation-id header value.
// This is a unique identifier for the installation, used for analytics.
// If not set, a random UUID will be generated.
func WithInstallationID(id string) Option {
	return func(o *options) {
		o.installationID = id
	}
}

// WithBetaFeatures sets the x-codex-beta-features header value.
// This is a comma-separated list of beta features to enable.
// See https://github.com/openai/codex/blob/main/codex-rs/features/src/lib.rs for available features.
func WithBetaFeatures(features string) Option {
	return func(o *options) {
		o.betaFeatures = features
	}
}
