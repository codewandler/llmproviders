package minimax

import (
	"context"
	"fmt"
	"net/http"
	"os"
)

// Auth provides authentication for MiniMax API requests.
type Auth interface {
	// ApplyAuth sets authentication headers on the request.
	ApplyAuth(ctx context.Context, h http.Header) error
}

// APIKeyAuth provides API key authentication for MiniMax.
// MiniMax requires the API key in both Authorization header and x-api-key header.
type APIKeyAuth struct {
	key string
}

// NewAPIKeyAuth creates an APIKeyAuth that uses a static API key.
func NewAPIKeyAuth(key string) *APIKeyAuth {
	return &APIKeyAuth{key: key}
}

// NewAPIKeyAuthFromEnv creates an APIKeyAuth from the MINIMAX_API_KEY environment variable.
func NewAPIKeyAuthFromEnv() (*APIKeyAuth, error) {
	key := EnvAPIKeyValue()
	if key == "" {
		return nil, fmt.Errorf("minimax: %s environment variable not set", EnvAPIKey)
	}
	return &APIKeyAuth{key: key}, nil
}

// EnvAPIKeyValue returns the MiniMax API key from the environment variable.
// Returns empty string if not set.
func EnvAPIKeyValue() string {
	return os.Getenv(EnvAPIKey)
}

// ApplyAuth sets the authentication headers on the request.
// MiniMax requires both Authorization: Bearer and x-api-key headers.
func (a *APIKeyAuth) ApplyAuth(_ context.Context, h http.Header) error {
	if a.key == "" {
		return fmt.Errorf("minimax: API key is empty")
	}
	h.Set("Authorization", "Bearer "+a.key)
	h.Set("x-api-key", a.key)
	return nil
}

// Ensure APIKeyAuth implements Auth.
var _ Auth = (*APIKeyAuth)(nil)
