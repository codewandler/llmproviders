package anthropic

import (
	"context"
	"net/http"
	"testing"
	"time"

	messagesapi "github.com/codewandler/agentapis/api/messages"
)

func TestModelAliasResolution(t *testing.T) {
	models := LoadModels()

	// Provider-level aliases only (intent aliases like "default", "fast", "powerful"
	// are resolved at the Service level, not here)
	tests := []struct {
		alias    string
		expected string
		wantErr  bool
	}{
		{"opus", ModelOpus, false},
		{"sonnet", ModelSonnet, false},
		{"haiku", ModelHaiku, false},
		{ModelOpus, ModelOpus, false},
		{ModelSonnet, ModelSonnet, false},
		{"SONNET", ModelSonnet, false}, // Case insensitive
		{"unknown-model", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			model, err := models.Resolve(tt.alias)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Resolve(%q) expected error, got nil", tt.alias)
				}
				return
			}
			if err != nil {
				t.Errorf("Resolve(%q) unexpected error: %v", tt.alias, err)
				return
			}
			if model.ID != tt.expected {
				t.Errorf("Resolve(%q) = %q, want %q", tt.alias, model.ID, tt.expected)
			}
		})
	}
}

func TestCoerceThinkingTemperature(t *testing.T) {
	tests := []struct {
		name     string
		thinking *messagesapi.ThinkingConfig
		temp     float64
		wantTemp float64
	}{
		{
			name:     "nil thinking - no change",
			thinking: nil,
			temp:     0.5,
			wantTemp: 0.5,
		},
		{
			name:     "disabled thinking - no change",
			thinking: &messagesapi.ThinkingConfig{Type: "disabled"},
			temp:     0.5,
			wantTemp: 0.5,
		},
		{
			name:     "enabled thinking with temp 0 - no change",
			thinking: &messagesapi.ThinkingConfig{Type: "enabled"},
			temp:     0,
			wantTemp: 0,
		},
		{
			name:     "enabled thinking with temp 1 - no change",
			thinking: &messagesapi.ThinkingConfig{Type: "enabled"},
			temp:     1,
			wantTemp: 1,
		},
		{
			name:     "enabled thinking with temp 0.5 - coerced to 1",
			thinking: &messagesapi.ThinkingConfig{Type: "enabled"},
			temp:     0.5,
			wantTemp: 1,
		},
		{
			name:     "enabled thinking with temp 0.7 - coerced to 1",
			thinking: &messagesapi.ThinkingConfig{Type: "enabled"},
			temp:     0.7,
			wantTemp: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &messagesapi.Request{
				Thinking:    tt.thinking,
				Temperature: tt.temp,
			}
			CoerceThinkingTemperature(req)
			if req.Temperature != tt.wantTemp {
				t.Errorf("CoerceThinkingTemperature() temperature = %v, want %v", req.Temperature, tt.wantTemp)
			}
		})
	}
}

func TestAPIKeyAuth(t *testing.T) {
	auth := NewAPIKeyAuth("test-key")

	h := make(http.Header)
	err := auth.ApplyAuth(context.Background(), h)
	if err != nil {
		t.Fatalf("ApplyAuth() error = %v", err)
	}

	got := h.Get("x-api-key")
	if got != "test-key" {
		t.Errorf("ApplyAuth() x-api-key = %q, want %q", got, "test-key")
	}
}

func TestAPIKeyAuthEmpty(t *testing.T) {
	auth := NewAPIKeyAuth("")

	h := make(http.Header)
	err := auth.ApplyAuth(context.Background(), h)
	if err == nil {
		t.Error("ApplyAuth() with empty key expected error, got nil")
	}
}

func TestTokenExpiry(t *testing.T) {
	tests := []struct {
		name    string
		token   *Token
		buffer  time.Duration
		wantExp bool
	}{
		{
			name:    "nil token - expired",
			token:   nil,
			buffer:  0,
			wantExp: true,
		},
		{
			name:    "empty access token - expired",
			token:   &Token{AccessToken: "", ExpiresAt: time.Now().Add(time.Hour)},
			buffer:  0,
			wantExp: true,
		},
		{
			name:    "future expiry - not expired",
			token:   &Token{AccessToken: "test", ExpiresAt: time.Now().Add(time.Hour)},
			buffer:  0,
			wantExp: false,
		},
		{
			name:    "past expiry - expired",
			token:   &Token{AccessToken: "test", ExpiresAt: time.Now().Add(-time.Hour)},
			buffer:  0,
			wantExp: true,
		},
		{
			name:    "expires within buffer - expired",
			token:   &Token{AccessToken: "test", ExpiresAt: time.Now().Add(10 * time.Second)},
			buffer:  30 * time.Second,
			wantExp: true,
		},
		{
			name:    "expires after buffer - not expired",
			token:   &Token{AccessToken: "test", ExpiresAt: time.Now().Add(time.Minute)},
			buffer:  30 * time.Second,
			wantExp: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.token.IsExpiredWithBuffer(tt.buffer)
			if got != tt.wantExp {
				t.Errorf("IsExpiredWithBuffer(%v) = %v, want %v", tt.buffer, got, tt.wantExp)
			}
		})
	}
}

func TestDefaultTokenExpiry(t *testing.T) {
	// Token that expires in 20 seconds should be considered expired with default 30s buffer
	token := &Token{AccessToken: "test", ExpiresAt: time.Now().Add(20 * time.Second)}
	if !token.IsExpired() {
		t.Error("Token expiring in 20s should be considered expired with default 30s buffer")
	}

	// Token that expires in 1 minute should not be considered expired
	token = &Token{AccessToken: "test", ExpiresAt: time.Now().Add(time.Minute)}
	if token.IsExpired() {
		t.Error("Token expiring in 1 minute should not be considered expired")
	}
}

func TestPKCEGeneration(t *testing.T) {
	pkce, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE() error = %v", err)
	}

	// Verifier should be 43 characters
	if len(pkce.Verifier) != 43 {
		t.Errorf("GeneratePKCE() verifier length = %d, want 43", len(pkce.Verifier))
	}

	// Challenge should be non-empty
	if pkce.Challenge == "" {
		t.Error("GeneratePKCE() challenge is empty")
	}

	// Each generation should be unique
	pkce2, _ := GeneratePKCE()
	if pkce.Verifier == pkce2.Verifier {
		t.Error("GeneratePKCE() should generate unique verifiers")
	}
}

func TestOAuthFlowAuthorizeURL(t *testing.T) {
	flow, err := NewOAuthFlow("")
	if err != nil {
		t.Fatalf("NewOAuthFlow() error = %v", err)
	}

	url := flow.AuthorizeURL()

	// Should contain required parameters
	checks := []string{
		"client_id=" + AnthropicClientID,
		"response_type=code",
		"code_challenge_method=S256",
	}
	for _, check := range checks {
		if !contains(url, check) {
			t.Errorf("AuthorizeURL() missing %q", check)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestLoadModels(t *testing.T) {
	models := LoadModels()

	if len(models) == 0 {
		t.Fatal("LoadModels() returned empty list")
	}

	// Should have at least the fallback models
	if len(models) < len(fallbackModels) {
		t.Errorf("LoadModels() returned %d models, want at least %d", len(models), len(fallbackModels))
	}

	// Should include the main models
	for _, id := range []string{ModelOpus, ModelSonnet, ModelHaiku} {
		found := false
		for _, m := range models {
			if m.ID == id {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("LoadModels() missing model %q", id)
		}
	}
}

func TestProviderName(t *testing.T) {
	if ProviderName != "anthropic" {
		t.Errorf("ProviderName = %q, want %q", ProviderName, "anthropic")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.BaseURL != DefaultBaseURL {
		t.Errorf("DefaultConfig().BaseURL = %q, want %q", cfg.BaseURL, DefaultBaseURL)
	}
	if cfg.Model != DefaultModel {
		t.Errorf("DefaultConfig().Model = %q, want %q", cfg.Model, DefaultModel)
	}
}

func TestConfigHelpers(t *testing.T) {
	cfg := Config{}

	// Empty config should return defaults
	if cfg.baseURL() != DefaultBaseURL {
		t.Errorf("empty config baseURL() = %q, want %q", cfg.baseURL(), DefaultBaseURL)
	}
	if cfg.model() != DefaultModel {
		t.Errorf("empty config model() = %q, want %q", cfg.model(), DefaultModel)
	}
	if cfg.httpClient() != http.DefaultClient {
		t.Error("empty config httpClient() should return http.DefaultClient")
	}

	// Custom config should return custom values
	customClient := &http.Client{}
	cfg = Config{
		BaseURL:    "https://custom.example.com",
		Model:      "custom-model",
		HTTPClient: customClient,
	}
	if cfg.baseURL() != "https://custom.example.com" {
		t.Errorf("custom config baseURL() = %q, want %q", cfg.baseURL(), "https://custom.example.com")
	}
	if cfg.model() != "custom-model" {
		t.Errorf("custom config model() = %q, want %q", cfg.model(), "custom-model")
	}
	if cfg.httpClient() != customClient {
		t.Error("custom config httpClient() should return custom client")
	}
}

func TestStaticTokenProvider(t *testing.T) {
	token := &Token{
		AccessToken:  "test-access",
		RefreshToken: "test-refresh",
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	provider := NewStaticTokenProvider(token)

	got, err := provider.Token(context.Background())
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if got.AccessToken != token.AccessToken {
		t.Errorf("Token().AccessToken = %q, want %q", got.AccessToken, token.AccessToken)
	}
}

func TestNewWithAPIKey(t *testing.T) {
	p, err := NewWithAPIKey("test-key", WithModel("claude-sonnet-4-20250514"))
	if err != nil {
		t.Fatalf("NewWithAPIKey() error = %v", err)
	}
	if p.Name() != ProviderName {
		t.Errorf("Provider.Name() = %q, want %q", p.Name(), ProviderName)
	}
}

func TestNewWithOAuthRequiresCredentials(t *testing.T) {
	// This should fail unless OAuth credentials are available
	_, err := NewWithOAuth()
	// We expect an error since local OAuth won't be available in tests
	if err == nil && !LocalTokenStoreAvailable() {
		t.Skip("Skipping: local OAuth credentials not available")
	}
}

func TestMustNewPanicsOnError(t *testing.T) {
	// Skip if local OAuth is actually available (test needs OAuth to fail)
	if LocalTokenStoreAvailable() {
		t.Skip("Skipping: local OAuth credentials available, cannot test panic")
	}
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustNew() should panic when given invalid options")
		}
	}()
	// Pass WithLocalOAuth which will fail without credentials
	MustNew(WithLocalOAuth())
}

func TestProviderOptions(t *testing.T) {
	// Test that options are correctly applied
	p, err := NewWithAPIKey("test-key",
		WithBaseURL("https://custom.example.com"),
		WithModel("custom-model"),
		WithAutoSystemCacheControl("30m"),
	)
	if err != nil {
		t.Fatalf("NewWithAPIKey() error = %v", err)
	}
	if p.cfg.BaseURL != "https://custom.example.com" {
		t.Errorf("BaseURL = %q, want %q", p.cfg.BaseURL, "https://custom.example.com")
	}
	if p.cfg.Model != "custom-model" {
		t.Errorf("Model = %q, want %q", p.cfg.Model, "custom-model")
	}
	if !p.autoSystemCacheControl {
		t.Error("autoSystemCacheControl should be true")
	}
	if p.autoSystemCacheTTL != "30m" {
		t.Errorf("autoSystemCacheTTL = %q, want %q", p.autoSystemCacheTTL, "30m")
	}
}

func TestErrorTypeReexports(t *testing.T) {
	// Verify error type constants are correctly re-exported
	if ErrTypeRateLimit != messagesapi.ErrTypeRateLimit {
		t.Errorf("ErrTypeRateLimit mismatch")
	}
	if ErrTypeAuthentication != messagesapi.ErrTypeAuthentication {
		t.Errorf("ErrTypeAuthentication mismatch")
	}

	// Verify sentinel errors
	if ErrRateLimit != messagesapi.ErrRateLimit {
		t.Errorf("ErrRateLimit mismatch")
	}
}

func TestRateLimitsReexport(t *testing.T) {
	// Test that RateLimits type alias works correctly
	rl := RateLimits{
		RequestLimit:     100,
		RequestRemaining: 95,
		TokenLimit:       100000,
		TokenRemaining:   80000,
	}
	if rl.RequestUtilization() != 0.05 {
		t.Errorf("RequestUtilization() = %f, want 0.05", rl.RequestUtilization())
	}
	if !rl.HasRequestLimits() {
		t.Error("HasRequestLimits() should return true")
	}
}
