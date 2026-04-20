package anthropic

import (
	"context"
	"fmt"
	"net/http"
	"os"
)

const (
	// EnvAPIKey is the environment variable for the Anthropic API key.
	EnvAPIKey = "ANTHROPIC_API_KEY"
)

// Auth provides authentication for API requests.
type Auth interface {
	// ApplyAuth sets authentication headers on the request.
	ApplyAuth(ctx context.Context, h http.Header) error
}

// APIKeyAuth authenticates using a static API key via x-api-key header.
type APIKeyAuth struct {
	Key string
}

// NewAPIKeyAuth creates an APIKeyAuth with the given key.
func NewAPIKeyAuth(key string) *APIKeyAuth {
	return &APIKeyAuth{Key: key}
}

// NewAPIKeyAuthFromEnv creates an APIKeyAuth from the ANTHROPIC_API_KEY env var.
// Returns an error if the env var is not set.
func NewAPIKeyAuthFromEnv() (*APIKeyAuth, error) {
	key := EnvAPIKeyValue()
	if key == "" {
		return nil, fmt.Errorf("anthropic: %s environment variable not set", EnvAPIKey)
	}
	return &APIKeyAuth{Key: key}, nil
}

// EnvAPIKeyValue returns the Anthropic API key from the environment variable.
// Returns empty string if not set.
func EnvAPIKeyValue() string {
	return os.Getenv(EnvAPIKey)
}

// ApplyAuth sets the x-api-key header.
func (a *APIKeyAuth) ApplyAuth(ctx context.Context, h http.Header) error {
	if a.Key == "" {
		return fmt.Errorf("anthropic: API key is empty")
	}
	h.Set("x-api-key", a.Key)
	return nil
}

// Ensure APIKeyAuth implements Auth.
var _ Auth = (*APIKeyAuth)(nil)
