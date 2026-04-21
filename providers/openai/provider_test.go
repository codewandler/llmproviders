package openai

import (
	"testing"
)

func TestModelAliasResolution(t *testing.T) {
	// Provider-level aliases only (intent aliases like "default", "fast" and
	// provider name "openai" are resolved at the Service level, not here)
	tests := []struct {
		alias string
		want  string
	}{
		{"flagship", ModelGPT54},
		{"mini", ModelGPT54Mini},
		{"nano", ModelGPT54Nano},
		{"pro", ModelGPT54Pro},
		{"codex", ModelGPT53Codex},
		{"o3", ModelO3},
		{"o4", ModelO4Mini},
		{"o1", ModelO1},
		{"gpt-5.4", ModelGPT54},
		{"gpt-4o", ModelGPT4o},
		{"gpt-3.5-turbo", ModelGPT35},
		{"unknown-model", "unknown-model"},
	}

	models := LoadModels()

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

	// Check that first model is gpt-5.4 (top of the list)
	if models[0].ID != ModelGPT54 {
		t.Errorf("First model = %q, want %q", models[0].ID, ModelGPT54)
	}

	// Check that all models have provider set
	for _, model := range models {
		if model.Provider != ProviderName {
			t.Errorf("Model %q has provider %q, want %q", model.ID, model.Provider, ProviderName)
		}
	}
}

func TestProviderName(t *testing.T) {
	// Set env var for test
	t.Setenv("OPENAI_API_KEY", "test-key")

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
	t.Setenv("OPENAI_API_KEY", "test-key")

	p, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	caps := p.Capabilities()
	if !caps.SupportsResponsesPreviousResponseID {
		t.Error("Expected SupportsResponsesPreviousResponseID to be true")
	}
}

func TestMapEffortAndThinking(t *testing.T) {
	tests := []struct {
		name        string
		model       string
		effort      string
		thinkingOff bool
		want        string
	}{
		// Non-reasoning models
		{"gpt-4o no effort", ModelGPT4o, "", false, ""},
		{"gpt-4o with effort", ModelGPT4o, EffortHigh, false, ""},
		{"gpt-3.5 no effort", ModelGPT35, "", false, ""},

		// Pre-GPT-5.1 reasoning models
		{"gpt-5.4 no effort", ModelGPT54, "", false, ""},
		{"gpt-5.4 with effort", ModelGPT54, EffortHigh, false, EffortHigh},
		{"gpt-5.4 thinking off", ModelGPT54, "", true, ""}, // Can't disable

		// GPT-5.1 (supports none)
		{"gpt-5.1 no effort", ModelGPT51, "", false, ""},
		{"gpt-5.1 with effort", ModelGPT51, EffortMedium, false, EffortMedium},
		{"gpt-5.1 thinking off", ModelGPT51, "", true, "none"},

		// Codex models (support xhigh)
		{"codex no effort", ModelGPT53Codex, "", false, ""},
		{"codex with max", ModelGPT53Codex, EffortMax, false, "xhigh"},
		{"codex with high", ModelGPT53Codex, EffortHigh, false, EffortHigh},

		// Pro models
		{"pro no effort", ModelGPT54Pro, "", false, ""},
		{"pro with max", ModelGPT54Pro, EffortMax, false, EffortHigh}, // Clamped

		// Unknown model
		{"unknown model", "custom-model", EffortHigh, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapEffortAndThinking(tt.model, tt.effort, tt.thinkingOff)
			if got != tt.want {
				t.Errorf("MapEffortAndThinking(%q, %q, %v) = %q, want %q",
					tt.model, tt.effort, tt.thinkingOff, got, tt.want)
			}
		})
	}
}

func TestUseResponsesAPI(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		// Responses API models
		{ModelGPT54, true},
		{ModelGPT54Mini, true},
		{ModelGPT51, true},

		// Codex models (always Responses API)
		{ModelGPT53Codex, true},
		{ModelGPT51Codex, true},

		// Chat completions models (not in registry as Responses)
		{ModelGPT4o, false},
		{ModelGPT4, false},
		{ModelGPT35, false},

		// Unknown model
		{"custom-model", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := UseResponsesAPI(tt.model)
			if got != tt.want {
				t.Errorf("UseResponsesAPI(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestSupportsPromptCaching(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{ModelGPT54, true},
		{ModelGPT54Mini, true},
		{ModelGPT51, true},
		{ModelGPT53Codex, true},
		{ModelGPT4o, false},
		{ModelGPT4, false},
		{"custom-model", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := SupportsPromptCaching(tt.model)
			if got != tt.want {
				t.Errorf("SupportsPromptCaching(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestProviderOptions(t *testing.T) {
	p, err := NewWithAPIKey("test-key",
		WithBaseURL("https://custom.example.com"),
		WithModel(ModelGPT54),
	)
	if err != nil {
		t.Fatalf("NewWithAPIKey() error = %v", err)
	}

	if p.cfg.BaseURL != "https://custom.example.com" {
		t.Errorf("BaseURL = %q, want %q", p.cfg.BaseURL, "https://custom.example.com")
	}
	if p.cfg.Model != ModelGPT54 {
		t.Errorf("Model = %q, want %q", p.cfg.Model, ModelGPT54)
	}
}

func TestResolveModel(t *testing.T) {
	// Provider-level resolution only (provider name "openai" is resolved at the
	// Service level, not here)
	tests := []struct {
		input string
		want  string
	}{
		{"", DefaultModel},
		{"gpt-5.4", ModelGPT54},
		{"flagship", ModelGPT54},
		{"mini", ModelGPT54Mini},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ResolveModel(tt.input)
			if got != tt.want {
				t.Errorf("ResolveModel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
