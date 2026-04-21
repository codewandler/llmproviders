package codex

import (
	"context"

	llmproviders "github.com/codewandler/llmproviders"
	"github.com/codewandler/llmproviders/registry"
)

// Registration priority order.
const OrderPriority = 80

// IntentAliases maps intent aliases to wire model IDs for this provider.
var IntentAliases = map[string]string{
	llmproviders.IntentFast:     ModelGPT54Mini,
	llmproviders.IntentDefault:  ModelGPT54,
	llmproviders.IntentPowerful: ModelO3,
}

// Register is the registration for the Codex provider.
var Register = registry.Registration{
	InstanceName:  ProviderName,
	ServiceID:     ServiceID,
	Order:         OrderPriority,
	Aliases:       ProviderAliases,
	IntentAliases: IntentAliases,
	Detect: func(ctx context.Context) (bool, error) {
		return LocalAvailable(), nil
	},
	Build: func(ctx context.Context, cfg registry.BuildConfig) (registry.Provider, error) {
		return New()
	},
}
