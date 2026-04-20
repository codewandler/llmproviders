package openai

import (
	"context"
	"fmt"
	"net/http"
	"os"
)

// Auth provides API key authentication for OpenAI.
type Auth struct {
	apiKey string
}

// NewAPIKeyAuth creates an Auth that uses a static API key.
func NewAPIKeyAuth(key string) *Auth {
	return &Auth{apiKey: key}
}

// NewEnvAuth creates an Auth that reads the API key from environment variables.
// It checks OPENAI_API_KEY first, then falls back to OPENAI_KEY.
func NewEnvAuth() (*Auth, error) {
	key := EnvAPIKeyValue()
	if key == "" {
		return nil, fmt.Errorf("openai: %s or %s environment variable not set", EnvAPIKey, EnvAPIKeyAlt)
	}
	return &Auth{apiKey: key}, nil
}

// EnvAPIKeyValue returns the OpenAI API key from environment variables.
// It checks OPENAI_API_KEY first, then falls back to OPENAI_KEY.
// Returns empty string if neither is set.
func EnvAPIKeyValue() string {
	if key := os.Getenv(EnvAPIKey); key != "" {
		return key
	}
	return os.Getenv(EnvAPIKeyAlt)
}

// SetHeaders sets the authorization headers on the request.
func (a *Auth) SetHeaders(_ context.Context, h http.Header) error {
	if a.apiKey == "" {
		return fmt.Errorf("openai: API key not configured")
	}
	h.Set("Authorization", "Bearer "+a.apiKey)
	return nil
}

// APIKey returns the configured API key.
func (a *Auth) APIKey() string {
	return a.apiKey
}
