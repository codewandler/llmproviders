package anthropic

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
)

const (
	// Claude CLI compatibility headers
	claudeUserAgent     = "claude-cli/2.1.85 (external, sdk-cli)"
	claudeBeta          = "claude-code-20250219,oauth-2025-04-20,interleaved-thinking-2025-05-14,context-management-2025-06-27,prompt-caching-scope-2026-01-05,effort-2025-11-24"
	stainlessPackageVer = "0.74.0"
	stainlessNodeVer    = "v24.3.0"
)

// OAuthAuth implements Auth using OAuth tokens.
type OAuthAuth struct {
	TokenProvider TokenProvider
	ClaudeHeaders bool // Enable Claude CLI compatibility headers
}

// NewOAuthAuth creates an OAuthAuth with the given token provider.
func NewOAuthAuth(tp TokenProvider) *OAuthAuth {
	return &OAuthAuth{
		TokenProvider: tp,
		ClaudeHeaders: false,
	}
}

// NewOAuthAuthWithClaudeHeaders creates an OAuthAuth with Claude CLI headers enabled.
func NewOAuthAuthWithClaudeHeaders(tp TokenProvider) *OAuthAuth {
	return &OAuthAuth{
		TokenProvider: tp,
		ClaudeHeaders: true,
	}
}

// NewLocalOAuthAuth creates an OAuthAuth using ~/.claude/.credentials.json.
// Returns an error if the credentials file doesn't exist.
func NewLocalOAuthAuth() (*OAuthAuth, error) {
	store, err := NewLocalTokenStore()
	if err != nil {
		return nil, err
	}
	tp := NewManagedTokenProvider(localTokenKey, store, nil)
	return NewOAuthAuth(tp), nil
}

// NewLocalOAuthAuthWithClaudeHeaders creates an OAuthAuth with Claude CLI headers
// using ~/.claude/.credentials.json.
func NewLocalOAuthAuthWithClaudeHeaders() (*OAuthAuth, error) {
	store, err := NewLocalTokenStore()
	if err != nil {
		return nil, err
	}
	tp := NewManagedTokenProvider(localTokenKey, store, nil)
	return NewOAuthAuthWithClaudeHeaders(tp), nil
}

// ApplyAuth sets the Authorization header and optionally Claude CLI headers.
func (a *OAuthAuth) ApplyAuth(ctx context.Context, h http.Header) error {
	if a.TokenProvider == nil {
		return fmt.Errorf("anthropic: OAuth token provider is nil")
	}

	token, err := a.TokenProvider.Token(ctx)
	if err != nil {
		return fmt.Errorf("anthropic: get OAuth token: %w", err)
	}

	h.Set("Authorization", "Bearer "+token.AccessToken)

	if a.ClaudeHeaders {
		setClaudeHeaders(h)
	}

	return nil
}

// setClaudeHeaders adds Claude CLI compatibility headers.
func setClaudeHeaders(h http.Header) {
	h.Set("User-Agent", claudeUserAgent)
	h.Set("Anthropic-Beta", claudeBeta)
	h.Set("Anthropic-Dangerous-Direct-Browser-Access", "true")
	h.Set("X-App", "cli")
	h.Set("X-Stainless-Lang", "js")
	h.Set("X-Stainless-Os", stainlessOS())
	h.Set("X-Stainless-Arch", stainlessArch())
	h.Set("X-Stainless-Package-Version", stainlessPackageVer)
	h.Set("X-Stainless-Retry-Count", "0")
	h.Set("X-Stainless-Runtime", "node")
	h.Set("X-Stainless-Runtime-Version", stainlessNodeVer)
	h.Set("X-Stainless-Timeout", "600")
}

func stainlessOS() string {
	switch runtime.GOOS {
	case "darwin":
		return "MacOS"
	case "windows":
		return "Windows"
	default:
		return "Linux"
	}
}

func stainlessArch() string {
	if runtime.GOARCH == "arm64" {
		return "arm64"
	}
	return "x64"
}

// Ensure OAuthAuth implements Auth.
var _ Auth = (*OAuthAuth)(nil)
