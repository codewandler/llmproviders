package openrouter

const (
	// DefaultBaseURL is the default OpenRouter API endpoint.
	DefaultBaseURL = "https://openrouter.ai/api"

	// ProviderName is the canonical name for this provider instance.
	ProviderName = "openrouter"

	// ServiceID is the modeldb service identifier for this provider.
	ServiceID = "openrouter"

	// DefaultModel is the default model when none is specified.
	DefaultModel = "anthropic/claude-3.7-sonnet"

	// EnvAPIKey is the environment variable for the OpenRouter API key.
	EnvAPIKey = "OPENROUTER_API_KEY"

	// AnthropicModelPrefix is the prefix for Anthropic models routed through OpenRouter.
	// Models with this prefix use the Anthropic Messages API instead of the Responses API.
	AnthropicModelPrefix = "anthropic/"
)

// Wire model ID constants for commonly used models via OpenRouter.
const (
	// Claude models via OpenRouter (these use the provider/model naming convention)
	ModelClaude37Sonnet = "anthropic/claude-3.7-sonnet"
	ModelClaude35Haiku  = "anthropic/claude-3-5-haiku"
	ModelClaudeOpus4    = "anthropic/claude-opus-4"

	// OpenAI models via OpenRouter
	ModelGPT4o     = "openai/gpt-4o"
	ModelGPT4oMini = "openai/gpt-4o-mini"
	ModelO3        = "openai/o3"
)

// Alias constants for short names.
const (
	AliasSonnet = "sonnet"
	AliasHaiku  = "haiku"
	AliasOpus   = "opus"
	AliasGPT4o  = "gpt-4o"
)

// ProviderAliases maps short names to wire model IDs for this provider.
var ProviderAliases = map[string]string{
	AliasSonnet: ModelClaude37Sonnet,
	AliasHaiku:  ModelClaude35Haiku,
	AliasOpus:   ModelClaudeOpus4,
	AliasGPT4o:  ModelGPT4o,
}

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
