package codex

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// tokenStore holds the OAuth tokens from the auth file.
type tokenStore struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	AccountID    string `json:"account_id"`
}

// authFile represents the ~/.codex/auth.json structure.
type authFile struct {
	AuthMode    string     `json:"auth_mode"`
	APIKey      *string    `json:"OPENAI_API_KEY"`
	Tokens      tokenStore `json:"tokens"`
	LastRefresh time.Time  `json:"last_refresh"`
}

// Auth manages Codex OAuth authentication with token refresh.
// It reads credentials from ~/.codex/auth.json (created by `codex login`)
// and automatically refreshes tokens when they expire.
type Auth struct {
	mu         sync.Mutex
	auth       authFile
	path       string
	expiry     time.Time
	httpClient *http.Client
}

// LoadAuth loads authentication from the default location (~/.codex/auth.json).
// Returns an error if the auth file doesn't exist or is invalid.
// Use LocalAvailable() to check if auth is available before calling this.
func LoadAuth() (*Auth, error) {
	// Check for override path
	if path := os.Getenv(EnvCodexAuthPath); path != "" {
		return loadAuthFrom(path)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("codex: get home dir: %w", err)
	}
	return loadAuthFrom(filepath.Join(home, AuthFilePath))
}

// LocalAvailable returns true if Codex OAuth credentials are available.
// This checks for the existence and validity of ~/.codex/auth.json.
func LocalAvailable() bool {
	path := os.Getenv(EnvCodexAuthPath)
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return false
		}
		path = filepath.Join(home, AuthFilePath)
	}
	a, err := loadAuthFrom(path)
	return err == nil && (a.auth.Tokens.AccessToken != "" || a.auth.Tokens.RefreshToken != "")
}

// loadAuthFrom loads and validates the auth file from the given path.
func loadAuthFrom(path string) (*Auth, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("codex: read %s: %w", path, err)
	}

	var auth authFile
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, fmt.Errorf("codex: parse auth file: %w", err)
	}

	// Validate auth mode
	if auth.AuthMode != "" && auth.AuthMode != ChatGPTAuthMode {
		return nil, fmt.Errorf("codex: unsupported auth mode %q", auth.AuthMode)
	}

	// Must have either access token or refresh token
	if auth.Tokens.AccessToken == "" && auth.Tokens.RefreshToken == "" {
		return nil, fmt.Errorf("codex: no tokens in %s", path)
	}

	a := &Auth{
		auth:       auth,
		path:       path,
		httpClient: http.DefaultClient,
	}

	// Parse JWT expiry if we have an access token
	if exp, err := jwtExpiry(auth.Tokens.AccessToken); err == nil {
		a.expiry = exp
	}

	return a, nil
}

// Token returns a valid access token, refreshing if necessary.
// This method is safe for concurrent use.
func (a *Auth) Token(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Token still valid with buffer
	if !a.expiry.IsZero() && time.Now().Add(TokenExpiryBuffer).Before(a.expiry) {
		return a.auth.Tokens.AccessToken, nil
	}

	// No expiry known but we have a token and no refresh token - use as-is
	if a.expiry.IsZero() && a.auth.Tokens.AccessToken != "" && a.auth.Tokens.RefreshToken == "" {
		return a.auth.Tokens.AccessToken, nil
	}

	// No refresh token - return existing access token or error
	if a.auth.Tokens.RefreshToken == "" {
		if a.auth.Tokens.AccessToken != "" {
			return a.auth.Tokens.AccessToken, nil
		}
		return "", fmt.Errorf("codex: no refresh token and access token is empty")
	}

	// Refresh the token
	return a.refresh(ctx)
}

// AccountID returns the account/workspace ID.
func (a *Auth) AccountID() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.auth.Tokens.AccountID
}

// SetHeaders sets the authentication headers on an HTTP request.
func (a *Auth) SetHeaders(ctx context.Context, h http.Header) error {
	token, err := a.Token(ctx)
	if err != nil {
		return err
	}

	h.Set("Authorization", "Bearer "+token)
	if accountID := a.AccountID(); accountID != "" {
		h.Set(HeaderChatGPTAccountID, accountID)
	}

	return nil
}

// refresh exchanges the refresh token for a new access token.
func (a *Auth) refresh(ctx context.Context) (string, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {a.auth.Tokens.RefreshToken},
		"client_id":     {ClientID},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("codex: build refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("codex: token refresh: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("codex: decode refresh response (status %d): %w", resp.StatusCode, err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("codex: token refresh failed: %s: %s", result.Error, result.ErrorDesc)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("codex: empty access token in refresh response (status %d)", resp.StatusCode)
	}

	// Update tokens
	a.auth.Tokens.AccessToken = result.AccessToken
	if result.RefreshToken != "" {
		a.auth.Tokens.RefreshToken = result.RefreshToken
	}

	// Update expiry
	if result.ExpiresIn > 0 {
		a.expiry = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	} else if exp, err := jwtExpiry(result.AccessToken); err == nil {
		a.expiry = exp
	} else {
		a.expiry = time.Time{}
	}

	// Persist to disk
	a.saveLocked()

	return result.AccessToken, nil
}

// saveLocked persists the current auth state to disk.
// Must be called with mu held.
func (a *Auth) saveLocked() {
	a.auth.LastRefresh = time.Now().UTC()
	data, err := json.MarshalIndent(a.auth, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(a.path, data, 0o600)
}

// jwtExpiry extracts the expiration time from a JWT token.
func jwtExpiry(token string) (time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("not a JWT")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("decode JWT payload: %w", err)
	}

	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return time.Time{}, fmt.Errorf("unmarshal JWT claims: %w", err)
	}

	if claims.Exp == 0 {
		return time.Time{}, fmt.Errorf("JWT has no exp claim")
	}

	return time.Unix(claims.Exp, 0), nil
}
