package openai

const (
	// DefaultBaseURL is the default OpenAI API endpoint.
	DefaultBaseURL = "https://api.openai.com"

	// ProviderName is the canonical name for this provider instance.
	ProviderName = "openai"

	// ServiceID is the modeldb service identifier for this provider.
	ServiceID = "openai"

	// DefaultModel is the default model when none is specified.
	DefaultModel = ModelGPT54Mini

	// EnvAPIKey is the primary environment variable for the OpenAI API key.
	EnvAPIKey = "OPENAI_API_KEY"

	// EnvAPIKeyAlt is the alternative environment variable for the OpenAI API key.
	EnvAPIKeyAlt = "OPENAI_KEY"
)

// Config holds OpenAI provider configuration.
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
