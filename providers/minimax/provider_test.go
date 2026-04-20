package minimax

import (
	"context"
	"net/http"
	"testing"
)

func TestModelAliasResolution(t *testing.T) {
	models := LoadModels()

	tests := []struct {
		alias    string
		expected string
		wantErr  bool
	}{
		{"minimax", ModelM27, false},
		{"default", ModelM27, false},
		{"fast", ModelM27, false},
		{"m2.7", ModelM27, false},
		{"2.7", ModelM27, false},
		{"m2.5", ModelM25, false},
		{"m2.1", ModelM21, false},
		{"m2", ModelM2, false},
		{ModelM27, ModelM27, false},
		{ModelM25, ModelM25, false},
		{"MINIMAX", ModelM27, false}, // Case insensitive
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

func TestAPIKeyAuth(t *testing.T) {
	auth := NewAPIKeyAuth("test-key")

	h := make(http.Header)
	err := auth.ApplyAuth(context.Background(), h)
	if err != nil {
		t.Fatalf("ApplyAuth() error = %v", err)
	}

	// MiniMax requires both headers
	gotAuth := h.Get("Authorization")
	if gotAuth != "Bearer test-key" {
		t.Errorf("ApplyAuth() Authorization = %q, want %q", gotAuth, "Bearer test-key")
	}

	gotKey := h.Get("x-api-key")
	if gotKey != "test-key" {
		t.Errorf("ApplyAuth() x-api-key = %q, want %q", gotKey, "test-key")
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

func TestLoadModels(t *testing.T) {
	models := LoadModels()

	if len(models) == 0 {
		t.Fatal("LoadModels() returned empty list")
	}

	// Should include the main models
	expectedModels := []string{ModelM27, ModelM25, ModelM21, ModelM2}
	for _, id := range expectedModels {
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
	if ProviderName != "minimax" {
		t.Errorf("ProviderName = %q, want %q", ProviderName, "minimax")
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

func TestNewWithAPIKey(t *testing.T) {
	p, err := NewWithAPIKey("test-key", WithModel("MiniMax-M2.7"))
	if err != nil {
		t.Fatalf("NewWithAPIKey() error = %v", err)
	}
	if p.Name() != ProviderName {
		t.Errorf("Provider.Name() = %q, want %q", p.Name(), ProviderName)
	}
}

func TestProviderOptions(t *testing.T) {
	p, err := NewWithAPIKey("test-key",
		WithBaseURL("https://custom.example.com"),
		WithModel("custom-model"),
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
}
