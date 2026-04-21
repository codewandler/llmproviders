package anthropic

import (
	"context"

	llmproviders "github.com/codewandler/llmproviders"
	"github.com/codewandler/llmproviders/registry"
)

// Registration priority order.
const OrderPriority = 20

// IntentAliases maps intent aliases to wire model IDs for this provider.
var IntentAliases = map[string]string{
	llmproviders.IntentFast:     ModelHaiku,
	llmproviders.IntentDefault:  ModelSonnet,
	llmproviders.IntentPowerful: ModelOpus,
}

// Register is the registration for the Anthropic provider.
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
		return New(WithAutoSystemCacheControl(""))
	},
}
