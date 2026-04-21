package anthropic

import (
	"context"

	"github.com/codewandler/llmproviders/registry"
)

// ClaudeOrderPriority is the registration priority for the Claude OAuth provider.
// Higher priority than API-key anthropic (15 < 20).
const ClaudeOrderPriority = 15

// ClaudeRegister is the registration for the Claude OAuth provider.
// Uses ~/.claude/.credentials.json for authentication (Claude Code/CLI credentials).
// This provider shares the same ServiceID as anthropic (same models) but uses
// OAuth authentication instead of API keys.
var ClaudeRegister = registry.Registration{
	InstanceName:  "claude",
	ServiceID:     ServiceID, // "anthropic" - shares models with API-key provider
	Order:         ClaudeOrderPriority,
	Aliases:       ProviderAliases,
	IntentAliases: IntentAliases,
	Detect: func(ctx context.Context) (bool, error) {
		return LocalTokenStoreAvailable(), nil
	},
	Build: func(ctx context.Context, cfg registry.BuildConfig) (registry.Provider, error) {
		return NewWithOAuthAndClaudeHeaders()
	},
}
