package openrouter

import (
	"context"
	"fmt"
	"net/http"
	"os"
)

// Auth provides API key authentication for OpenRouter.
type Auth struct {
	apiKey string
}

// NewAPIKeyAuth creates an Auth that uses a static API key.
func NewAPIKeyAuth(key string) *Auth {
	return &Auth{apiKey: key}
}

// NewEnvAuth creates an Auth that reads the API key from environment variables.
func NewEnvAuth() (*Auth, error) {
	key := EnvAPIKeyValue()
	if key == "" {
		return nil, fmt.Errorf("openrouter: %s environment variable not set", EnvAPIKey)
	}
	return &Auth{apiKey: key}, nil
}

// EnvAPIKeyValue returns the OpenRouter API key from the environment variable.
// Returns empty string if not set.
func EnvAPIKeyValue() string {
	return os.Getenv(EnvAPIKey)
}

// SetHeaders sets the authorization headers on the request.
func (a *Auth) SetHeaders(_ context.Context, h http.Header) error {
	if a.apiKey == "" {
		return fmt.Errorf("openrouter: API key not configured")
	}
	h.Set("Authorization", "Bearer "+a.apiKey)
	return nil
}

// APIKey returns the configured API key.
func (a *Auth) APIKey() string {
	return a.apiKey
}
