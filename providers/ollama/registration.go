package ollama

import (
	"context"

	"github.com/codewandler/llmproviders/registry"
)

// Registration priority order.
const OrderPriority = 70

// Register is the registration for the Ollama provider.
// Detection probes the Ollama API to check if it's running.
// Set OLLAMA_URL to override the default base URL (localhost:11434).
var Register = registry.Registration{
	InstanceName: ProviderName,
	ServiceID:    ServiceID,
	Order:        OrderPriority,
	// No aliases or intent aliases - Ollama models are dynamic
	Detect: func(ctx context.Context) (bool, error) {
		return ProbeAvailable(ctx), nil
	},
	Build: func(ctx context.Context, cfg registry.BuildConfig) (registry.Provider, error) {
		return New()
	},
}
