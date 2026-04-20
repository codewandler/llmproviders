package openrouter

const (
	// DefaultBaseURL is the default OpenRouter API endpoint.
	DefaultBaseURL = "https://openrouter.ai/api"

	// ProviderName is the canonical name for this provider.
	ProviderName = "openrouter"

	// DefaultModel is the default model when none is specified.
	DefaultModel = "openrouter/auto"

	// EnvAPIKey is the environment variable for the OpenRouter API key.
	EnvAPIKey = "OPENROUTER_API_KEY"

	// AnthropicModelPrefix is the prefix for Anthropic models routed through OpenRouter.
	// Models with this prefix use the Anthropic Messages API instead of the Responses API.
	AnthropicModelPrefix = "anthropic/"
)

// Config holds OpenRouter provider configuration.
type Config struct {
	BaseURL string
	APIKey  string
	Model   string
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		BaseURL: DefaultBaseURL,
		Model:   DefaultModel,
	}
}
