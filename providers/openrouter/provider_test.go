package openrouter

import (
	"testing"
)

func TestModelAliasResolution(t *testing.T) {
	models := LoadModels()

	tests := []struct {
		alias string
		want  string
	}{
		{"", "openrouter/auto"},
		{"default", "openrouter/auto"},
		{"openrouter", "openrouter/auto"},
		{"auto", "openrouter/auto"},
		{"fast", "openrouter/auto"},
		{"unknown-model", "unknown-model"},
	}

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			model, _ := models.Resolve(tt.alias)
			if model.ID != tt.want {
				t.Errorf("Resolve(%q) = %q, want %q", tt.alias, model.ID, tt.want)
			}
		})
	}
}

func TestAPIKeyAuth(t *testing.T) {
	auth := NewAPIKeyAuth("test-key")
	if auth.APIKey() != "test-key" {
		t.Errorf("APIKey() = %q, want %q", auth.APIKey(), "test-key")
	}
}

func TestLoadModels(t *testing.T) {
	models := LoadModels()
	if len(models) == 0 {
		t.Fatal("LoadModels() returned empty list")
	}

	// Check that first model is openrouter/auto or has the auto alias
	foundAuto := false
	for _, model := range models {
		if model.ID == "openrouter/auto" {
			foundAuto = true
			break
		}
	}
	if !foundAuto {
		t.Error("Expected openrouter/auto model in list")
	}

	// Check that all models have provider set
	for _, model := range models {
		if model.Provider != ProviderName {
			t.Errorf("Model %q has provider %q, want %q", model.ID, model.Provider, ProviderName)
		}
	}
}

func TestProviderName(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "test-key")

	p, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if p.Name() != ProviderName {
		t.Errorf("Name() = %q, want %q", p.Name(), ProviderName)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, DefaultBaseURL)
	}
	if cfg.Model != DefaultModel {
		t.Errorf("Model = %q, want %q", cfg.Model, DefaultModel)
	}
}

func TestNewWithAPIKey(t *testing.T) {
	p, err := NewWithAPIKey("test-key")
	if err != nil {
		t.Fatalf("NewWithAPIKey() error = %v", err)
	}

	if p.Name() != ProviderName {
		t.Errorf("Name() = %q, want %q", p.Name(), ProviderName)
	}
}

func TestProviderCapabilities(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "test-key")

	p, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	caps := p.Capabilities()
	if !caps.SupportsResponsesPreviousResponseID {
		t.Error("Expected SupportsResponsesPreviousResponseID to be true")
	}
}

func TestUseMessagesAPI(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"anthropic/claude-sonnet-4-6", true},
		{"anthropic/claude-3.5-sonnet", true},
		{"anthropic/claude-3-opus", true},
		{"openai/gpt-5.4", false},
		{"google/gemini-2.0-flash", false},
		{"openrouter/auto", false},
		{"meta-llama/llama-3.1-70b-instruct", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := UseMessagesAPI(tt.model)
			if got != tt.want {
				t.Errorf("UseMessagesAPI(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestStripAnthropicPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"anthropic/claude-sonnet-4-6", "claude-sonnet-4-6"},
		{"anthropic/claude-3.5-sonnet", "claude-3.5-sonnet"},
		{"openai/gpt-5.4", "openai/gpt-5.4"},
		{"openrouter/auto", "openrouter/auto"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := StripAnthropicPrefix(tt.input)
			if got != tt.want {
				t.Errorf("StripAnthropicPrefix(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestProviderOptions(t *testing.T) {
	p, err := NewWithAPIKey("test-key",
		WithBaseURL("https://custom.example.com"),
		WithModel("openai/gpt-5.4"),
	)
	if err != nil {
		t.Fatalf("NewWithAPIKey() error = %v", err)
	}

	if p.cfg.BaseURL != "https://custom.example.com" {
		t.Errorf("BaseURL = %q, want %q", p.cfg.BaseURL, "https://custom.example.com")
	}
	if p.cfg.Model != "openai/gpt-5.4" {
		t.Errorf("Model = %q, want %q", p.cfg.Model, "openai/gpt-5.4")
	}
}

func TestSupportsPromptCaching(t *testing.T) {
	// Test with known model from models.json
	// The result depends on the model data in models.json
	// Just verify the function doesn't panic
	_ = SupportsPromptCaching("openrouter/auto")
	_ = SupportsPromptCaching("unknown-model")
}
