package codex

import "time"

// Provider constants.
const (
	// ProviderName is the identifier for this provider.
	ProviderName = "codex"

	// DefaultBaseURL is the Codex backend API base URL.
	// This is different from the standard OpenAI API - it goes through chatgpt.com.
	DefaultBaseURL = "https://chatgpt.com/backend-api"

	// DefaultPath is the API path for Codex responses.
	// The agentapis Responses client will append this to the base URL.
	DefaultPath = "/codex/responses"

	// ModelsEndpoint is the endpoint for fetching available models.
	ModelsEndpoint = "https://chatgpt.com/backend-api/codex/models"

	// DefaultModel is the default model for Codex requests.
	DefaultModel = "gpt-5.4"
)

// Environment variable names.
const (
	// EnvCodexAuthPath can override the default auth file location.
	EnvCodexAuthPath = "CODEX_AUTH_PATH"
)

// Auth constants.
const (
	// AuthFilePath is the relative path to the Codex auth file from user home.
	AuthFilePath = ".codex/auth.json"

	// TokenEndpoint is the OAuth token refresh endpoint.
	// Source: https://github.com/openai/codex/blob/main/codex-rs/login/src/auth/manager.rs
	TokenEndpoint = "https://auth.openai.com/oauth/token"

	// ClientID is the OAuth client ID for the Codex CLI.
	// Source: https://github.com/openai/codex/blob/main/codex-rs/login/src/auth/manager.rs
	ClientID = "app_EMoamEEZ73f0CkXaXp7hrann"

	// TokenExpiryBuffer is how early to refresh tokens before expiry.
	TokenExpiryBuffer = 5 * time.Minute

	// ChatGPTAuthMode is the auth mode value for ChatGPT OAuth.
	ChatGPTAuthMode = "chatgpt"
)

// Auth headers.
// Source: https://github.com/openai/codex/blob/main/codex-rs/model-provider/src/bearer_auth_provider.rs
const (
	// HeaderChatGPTAccountID identifies the workspace/account for ChatGPT auth.
	// Source: https://github.com/openai/codex/blob/main/codex-rs/model-provider/src/bearer_auth_provider.rs#L33
	HeaderChatGPTAccountID = "ChatGPT-Account-ID"

	// HeaderOpenAIFedRAMP is set to "true" for FedRAMP accounts (special routing).
	// Source: https://github.com/openai/codex/blob/main/codex-rs/model-provider/src/bearer_auth_provider.rs#L36
	HeaderOpenAIFedRAMP = "X-OpenAI-Fedramp"
)

// Session and routing headers.
// Source: https://github.com/openai/codex/blob/main/codex-rs/core/src/client.rs
const (
	// HeaderSessionID tracks the conversation/thread ID.
	// Source: https://github.com/openai/codex/blob/main/codex-rs/codex-api/src/requests/headers.rs#L8
	HeaderSessionID = "session_id"

	// HeaderCodexTurnState is a sticky routing token returned by the server.
	// Replaying it routes subsequent requests to the same backend instance
	// for better KV cache hits within a turn. The server returns this in response
	// headers; the client must replay it on subsequent requests in the same turn.
	// Source: https://github.com/openai/codex/blob/main/codex-rs/core/src/client.rs#L130
	HeaderCodexTurnState = "x-codex-turn-state"

	// HeaderCodexWindowID format: "{conversation_id}:{window_generation}"
	// Used for prompt caching key derivation.
	// Source: https://github.com/openai/codex/blob/main/codex-rs/core/src/client.rs#L133
	HeaderCodexWindowID = "x-codex-window-id"

	// HeaderCodexInstallationID is a unique installation identifier for analytics.
	// Source: https://github.com/openai/codex/blob/main/codex-rs/core/src/client.rs#L129
	HeaderCodexInstallationID = "x-codex-installation-id"

	// HeaderCodexBetaFeatures is a comma-separated list of enabled beta feature keys.
	// Source: https://github.com/openai/codex/blob/main/codex-rs/core/src/client.rs#L1600
	HeaderCodexBetaFeatures = "x-codex-beta-features"

	// HeaderCodexTurnMetadata is optional per-turn metadata for observability.
	// Source: https://github.com/openai/codex/blob/main/codex-rs/core/src/client.rs#L131
	HeaderCodexTurnMetadata = "x-codex-turn-metadata"

	// HeaderCodexParentThreadID is the parent thread ID for spawned sub-agents.
	// Source: https://github.com/openai/codex/blob/main/codex-rs/core/src/client.rs#L132
	HeaderCodexParentThreadID = "x-codex-parent-thread-id"
)

// Sub-agent headers.
// Source: https://github.com/openai/codex/blob/main/codex-rs/core/src/client.rs
const (
	// HeaderOpenAISubagent identifies the sub-agent type.
	// Values: "review", "compact", "memory_consolidation", "collab_spawn"
	// Source: https://github.com/openai/codex/blob/main/codex-rs/core/src/client.rs#L135
	HeaderOpenAISubagent = "x-openai-subagent"

	// HeaderOpenAIMemgenRequest is set to "true" for memory consolidation requests.
	// Source: https://github.com/openai/codex/blob/main/codex-rs/core/src/client.rs#L134
	HeaderOpenAIMemgenRequest = "x-openai-memgen-request"

	// HeaderTimingMetrics requests timing breakdown in response.
	// Source: https://github.com/openai/codex/blob/main/codex-rs/core/src/client.rs#L136
	HeaderTimingMetrics = "x-responsesapi-include-timing-metrics"
)

// WebSocket headers.
const (
	// HeaderOpenAIBeta is used for WebSocket protocol version.
	// Value for WebSocket v2: "responses_websockets=2026-02-06"
	// Source: https://github.com/openai/codex/blob/main/codex-rs/core/src/client.rs#L138
	HeaderOpenAIBeta = "OpenAI-Beta"

	// WebSocketBetaValue is the OpenAI-Beta header value for WebSocket v2.
	WebSocketBetaValue = "responses_websockets=2026-02-06"
)

// Effort mapping constants.
const (
	// EffortXHigh is the Codex-specific effort level above "high".
	// Mapped from EffortMax in the unified API.
	EffortXHigh = "xhigh"
)

// Config holds the Codex provider configuration.
type Config struct {
	// BaseURL is the API endpoint URL.
	BaseURL string

	// Model is the default model to use.
	Model string

	// InstallationID is the unique installation identifier.
	// If empty, a random UUID will be generated.
	InstallationID string

	// BetaFeatures is a comma-separated list of beta features to enable.
	BetaFeatures string
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() Config {
	return Config{
		BaseURL: DefaultBaseURL,
		Model:   DefaultModel,
	}
}
