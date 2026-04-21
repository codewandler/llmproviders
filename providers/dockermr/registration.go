package dockermr

import (
	"context"

	"github.com/codewandler/llmproviders/registry"
)

// Registration priority order.
const OrderPriority = 90

// Register is the registration for the Docker Model Runner provider.
// Docker Model Runner is always detected as available since it's a local provider
// (connection errors happen at runtime, not detection time).
var Register = registry.Registration{
	InstanceName: ProviderName,
	ServiceID:    ServiceID,
	Order:        OrderPriority,
	// No aliases or intent aliases - DockerMR models are dynamic
	Detect: func(ctx context.Context) (bool, error) {
		return true, nil // Always available - errors happen at runtime
	},
	Build: func(ctx context.Context, cfg registry.BuildConfig) (registry.Provider, error) {
		return New()
	},
}
