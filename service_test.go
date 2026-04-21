package llmproviders

import (
	"context"
	"testing"

	"github.com/codewandler/llmproviders/registry"
)

// testProvider is defined in registry_test.go

// newTestRegistry creates a registry with test providers for resolution testing.
func newTestRegistry(t *testing.T) *registry.Registry {
	t.Helper()
	reg := registry.New()

	// Register anthropic (highest priority, order=20)
	reg.Register(registry.Registration{
		InstanceName: "anthropic",
		ServiceID:    "anthropic",
		Order:        20,
		Aliases: map[string]string{
			"sonnet": "claude-sonnet-4-6",
			"opus":   "claude-opus-4-6",
			"haiku":  "claude-haiku-4-5-20251001",
		},
		IntentAliases: map[string]string{
			IntentFast:     "claude-haiku-4-5-20251001",
			IntentDefault:  "claude-sonnet-4-6",
			IntentPowerful: "claude-opus-4-6",
		},
		Detect: func(ctx context.Context) (bool, error) { return true, nil },
		Build: func(ctx context.Context, cfg registry.BuildConfig) (registry.Provider, error) {
			return &testProvider{name: "anthropic"}, nil
		},
	})

	// Register openai (order=40)
	reg.Register(registry.Registration{
		InstanceName: "openai",
		ServiceID:    "openai",
		Order:        40,
		Aliases: map[string]string{
			"mini": "gpt-5.4-mini",
			"nano": "gpt-5.4-nano",
			"gpt":  "gpt-5.4",
		},
		IntentAliases: map[string]string{
			IntentFast:     "gpt-5.4-nano",
			IntentDefault:  "gpt-5.4-mini",
			IntentPowerful: "o3",
		},
		Detect: func(ctx context.Context) (bool, error) { return true, nil },
		Build: func(ctx context.Context, cfg registry.BuildConfig) (registry.Provider, error) {
			return &testProvider{name: "openai"}, nil
		},
	})

	// Register openrouter (order=50)
	reg.Register(registry.Registration{
		InstanceName: "openrouter",
		ServiceID:    "openrouter",
		Order:        50,
		Aliases: map[string]string{
			"sonnet": "anthropic/claude-3.7-sonnet", // shadowed by anthropic
			"opus":   "anthropic/claude-opus-4",     // shadowed by anthropic
			"gpt-4o": "openai/gpt-4o",
		},
		IntentAliases: map[string]string{
			IntentFast:     "anthropic/claude-3-5-haiku",
			IntentDefault:  "anthropic/claude-3.7-sonnet",
			IntentPowerful: "anthropic/claude-opus-4",
		},
		Detect: func(ctx context.Context) (bool, error) { return true, nil },
		Build: func(ctx context.Context, cfg registry.BuildConfig) (registry.Provider, error) {
			return &testProvider{name: "openrouter"}, nil
		},
	})

	return reg
}

func TestService_IntentAliases(t *testing.T) {
	reg := newTestRegistry(t)
	svc, err := NewService(WithRegistry(reg))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	tests := []struct {
		name         string
		input        string
		wantProvider string
		wantModel    string
	}{
		{
			name:         "fast resolves to anthropic haiku (highest priority)",
			input:        IntentFast,
			wantProvider: "anthropic",
			wantModel:    "claude-haiku-4-5-20251001",
		},
		{
			name:         "default resolves to anthropic sonnet",
			input:        IntentDefault,
			wantProvider: "anthropic",
			wantModel:    "claude-sonnet-4-6",
		},
		{
			name:         "powerful resolves to anthropic opus",
			input:        IntentPowerful,
			wantProvider: "anthropic",
			wantModel:    "claude-opus-4-6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, model, err := svc.ProviderFor(tt.input)
			if err != nil {
				t.Fatalf("ProviderFor(%q) error = %v", tt.input, err)
			}
			if provider.Name() != tt.wantProvider {
				t.Errorf("ProviderFor(%q) provider = %q, want %q", tt.input, provider.Name(), tt.wantProvider)
			}
			if model != tt.wantModel {
				t.Errorf("ProviderFor(%q) model = %q, want %q", tt.input, model, tt.wantModel)
			}
		})
	}
}

func TestService_ProviderAliases(t *testing.T) {
	reg := newTestRegistry(t)
	svc, err := NewService(WithRegistry(reg))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	tests := []struct {
		name         string
		input        string
		wantProvider string
		wantModel    string
	}{
		{
			name:         "sonnet resolves via anthropic (highest priority)",
			input:        "sonnet",
			wantProvider: "anthropic",
			wantModel:    "claude-sonnet-4-6",
		},
		{
			name:         "opus resolves via anthropic",
			input:        "opus",
			wantProvider: "anthropic",
			wantModel:    "claude-opus-4-6",
		},
		{
			name:         "haiku resolves via anthropic",
			input:        "haiku",
			wantProvider: "anthropic",
			wantModel:    "claude-haiku-4-5-20251001",
		},
		{
			name:         "mini resolves via openai",
			input:        "mini",
			wantProvider: "openai",
			wantModel:    "gpt-5.4-mini",
		},
		{
			name:         "gpt-4o resolves via openrouter (not shadowed)",
			input:        "gpt-4o",
			wantProvider: "openrouter",
			wantModel:    "openai/gpt-4o",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, model, err := svc.ProviderFor(tt.input)
			if err != nil {
				t.Fatalf("ProviderFor(%q) error = %v", tt.input, err)
			}
			if provider.Name() != tt.wantProvider {
				t.Errorf("ProviderFor(%q) provider = %q, want %q", tt.input, provider.Name(), tt.wantProvider)
			}
			if model != tt.wantModel {
				t.Errorf("ProviderFor(%q) model = %q, want %q", tt.input, model, tt.wantModel)
			}
		})
	}
}

func TestService_ServiceModelResolution(t *testing.T) {
	reg := newTestRegistry(t)
	svc, err := NewService(WithRegistry(reg))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	// Note: Models that exist in OpenRouter's catalog as wire model IDs
	// (like "openai/gpt-5.4-mini") will route to OpenRouter because
	// catalog wire model lookup happens BEFORE service/model parsing.
	// This is intentional - use explicit instance prefix for native routing.

	tests := []struct {
		name         string
		input        string
		wantProvider string
		wantModel    string
	}{
		{
			name:         "anthropic/claude-sonnet-4-6 routes to native anthropic",
			input:        "anthropic/claude-sonnet-4-6",
			wantProvider: "anthropic",
			wantModel:    "claude-sonnet-4-6",
		},
		{
			// openai/gpt-5.4-mini exists in OpenRouter catalog, so it routes there
			name:         "openai/gpt-5.4-mini routes to openrouter (catalog wire model)",
			input:        "openai/gpt-5.4-mini",
			wantProvider: "openrouter",
			wantModel:    "openai/gpt-5.4-mini",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, model, err := svc.ProviderFor(tt.input)
			if err != nil {
				t.Fatalf("ProviderFor(%q) error = %v", tt.input, err)
			}
			if provider.Name() != tt.wantProvider {
				t.Errorf("ProviderFor(%q) provider = %q, want %q", tt.input, provider.Name(), tt.wantProvider)
			}
			if model != tt.wantModel {
				t.Errorf("ProviderFor(%q) model = %q, want %q", tt.input, model, tt.wantModel)
			}
		})
	}
}

func TestService_CatalogWireModelLookup(t *testing.T) {
	reg := newTestRegistry(t)
	svc, err := NewService(WithRegistry(reg))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	// This tests that OpenRouter wire models like "anthropic/claude-3-5-haiku"
	// are found in the catalog BEFORE being parsed as service/model.
	// The key insight: "anthropic/claude-3-5-haiku" exists in openrouter's catalog
	// as a wire model ID, so it should route to openrouter, NOT native anthropic.

	tests := []struct {
		name         string
		input        string
		wantProvider string
		wantModel    string
	}{
		{
			name:         "anthropic/claude-3-5-haiku routes to openrouter (catalog lookup)",
			input:        "anthropic/claude-3-5-haiku",
			wantProvider: "openrouter",
			wantModel:    "anthropic/claude-3-5-haiku",
		},
		{
			name:         "openai/gpt-4o routes to openrouter (catalog lookup)",
			input:        "openai/gpt-4o",
			wantProvider: "openrouter",
			wantModel:    "openai/gpt-4o",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, model, err := svc.ProviderFor(tt.input)
			if err != nil {
				t.Fatalf("ProviderFor(%q) error = %v", tt.input, err)
			}
			if provider.Name() != tt.wantProvider {
				t.Errorf("ProviderFor(%q) provider = %q, want %q", tt.input, provider.Name(), tt.wantProvider)
			}
			if model != tt.wantModel {
				t.Errorf("ProviderFor(%q) model = %q, want %q", tt.input, model, tt.wantModel)
			}
		})
	}
}

func TestService_IntentAliases_FirstProviderWins(t *testing.T) {
	// Test that the provider with lowest Order value wins for intent aliases
	aliases := map[string]ResolvedRef{
		IntentFast:     {},
		IntentDefault:  {},
		IntentPowerful: {},
	}

	reg := newTestRegistry(t)
	svc, err := NewService(WithRegistry(reg))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	aliases = svc.IntentAliases()

	// All intents should resolve via anthropic (order=20, highest priority)
	for intent, ref := range aliases {
		if ref.InstanceName != "anthropic" {
			t.Errorf("IntentAliases()[%q].InstanceName = %q, want anthropic (highest priority)",
				intent, ref.InstanceName)
		}
	}
}

func TestService_ProviderAliases_MergeFromAllProviders(t *testing.T) {
	reg := newTestRegistry(t)
	svc, err := NewService(WithRegistry(reg))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	aliases := svc.ProviderAliases()

	// Check that aliases from multiple providers are present
	expectedAliases := map[string]string{
		"sonnet": "anthropic",  // from anthropic (shadows openrouter)
		"opus":   "anthropic",  // from anthropic (shadows openrouter)
		"haiku":  "anthropic",  // from anthropic
		"mini":   "openai",     // from openai
		"nano":   "openai",     // from openai
		"gpt":    "openai",     // from openai
		"gpt-4o": "openrouter", // from openrouter (not shadowed)
	}

	for alias, wantService := range expectedAliases {
		target, ok := aliases[alias]
		if !ok {
			t.Errorf("ProviderAliases() missing alias %q", alias)
			continue
		}
		if target.ServiceID != wantService {
			t.Errorf("ProviderAliases()[%q].ServiceID = %q, want %q", alias, target.ServiceID, wantService)
		}
	}
}

func TestService_RegisteredServices(t *testing.T) {
	reg := newTestRegistry(t)
	svc, err := NewService(WithRegistry(reg))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	services := svc.RegisteredServices()
	want := []string{"anthropic", "openai", "openrouter"}

	if len(services) != len(want) {
		t.Fatalf("RegisteredServices() = %v, want %v", services, want)
	}

	for i, s := range services {
		if s != want[i] {
			t.Errorf("RegisteredServices()[%d] = %q, want %q", i, s, want[i])
		}
	}
}

func TestService_RegisteredInstances(t *testing.T) {
	reg := newTestRegistry(t)
	svc, err := NewService(WithRegistry(reg))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	instances := svc.RegisteredInstances()
	want := []string{"anthropic", "openai", "openrouter"}

	if len(instances) != len(want) {
		t.Fatalf("RegisteredInstances() = %v, want %v", instances, want)
	}

	for i, s := range instances {
		if s != want[i] {
			t.Errorf("RegisteredInstances()[%d] = %q, want %q", i, s, want[i])
		}
	}
}
