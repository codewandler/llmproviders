package minimax

import (
	"context"

	llmproviders "github.com/codewandler/llmproviders"
	"github.com/codewandler/llmproviders/registry"
)

// Registration priority order.
const OrderPriority = 60

// IntentAliases maps intent aliases to wire model IDs for this provider.
// Note: IntentFast is intentionally omitted because MiniMax's "highspeed" model
// is a premium tier (fast but not cheap), which doesn't fit the "fast" intent semantics.
var IntentAliases = map[string]string{
	llmproviders.IntentDefault:  ModelM27,
	llmproviders.IntentPowerful: ModelM27,
}

// Register is the registration for the MiniMax provider.
var Register = registry.Registration{
	InstanceName:  ProviderName,
	ServiceID:     ServiceID,
	Order:         OrderPriority,
	Aliases:       ProviderAliases,
	IntentAliases: IntentAliases,
	Detect: func(ctx context.Context) (bool, error) {
		return EnvAPIKeyValue() != "", nil
	},
	Build: func(ctx context.Context, cfg registry.BuildConfig) (registry.Provider, error) {
		return New()
	},
}
