package dockermr

import (
	"context"

	"github.com/codewandler/llmproviders/registry"
)

// Registration priority order.
const OrderPriority = 90

// Register is the registration for the Docker Model Runner provider.
// Detection probes the Docker Model Runner API to check if it's running.
// Set DOCKER_MODEL_RUNNER_URL to override the default base URL (localhost:12434).
var Register = registry.Registration{
	InstanceName: ProviderName,
	ServiceID:    ServiceID,
	Order:        OrderPriority,
	// No aliases or intent aliases - DockerMR models are dynamic
	Detect: func(ctx context.Context) (bool, error) {
		return ProbeAvailable(ctx), nil
	},
	Build: func(ctx context.Context, cfg registry.BuildConfig) (registry.Provider, error) {
		return New()
	},
}
