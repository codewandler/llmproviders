package llmproviders

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/codewandler/llmproviders/registry"
	"github.com/codewandler/modeldb"
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

func newTestCatalog() modeldb.Catalog {
	catalog := modeldb.NewCatalog()
	for _, offering := range []modeldb.Offering{
		{ServiceID: "anthropic", WireModelID: "claude-sonnet-4-6", Aliases: []string{"sonnet-latest"}},
		{ServiceID: "anthropic", WireModelID: "claude-opus-4-6"},
		{ServiceID: "anthropic", WireModelID: "claude-haiku-4-5-20251001"},
		{ServiceID: "anthropic", WireModelID: "shared-model"},
		{ServiceID: "openai", WireModelID: "gpt-5.4", Aliases: []string{"gpt-latest"}},
		{ServiceID: "openai", WireModelID: "gpt-5.4-mini"},
		{ServiceID: "openai", WireModelID: "gpt-5.4-nano"},
		{ServiceID: "openai", WireModelID: "o3"},
		{ServiceID: "openai", WireModelID: "shared-model"},
		{ServiceID: "openrouter", WireModelID: "openai/gpt-4o"},
		{ServiceID: "openrouter", WireModelID: "openai/gpt-5.4-mini"},
		{ServiceID: "openrouter", WireModelID: "openai/gpt-5.4"},
		{ServiceID: "openrouter", WireModelID: "anthropic/claude-3.7-sonnet"},
		{ServiceID: "openrouter", WireModelID: "anthropic/claude-3-5-haiku"},
		{ServiceID: "openrouter", WireModelID: "anthropic/claude-3-5-sonnet"},
		{ServiceID: "openrouter", WireModelID: "meta-llama/llama-3-70b"},
	} {
		catalog.Offerings[modeldb.OfferingRef{ServiceID: offering.ServiceID, WireModelID: offering.WireModelID}] = offering
	}
	return catalog
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	svc, err := NewService(WithRegistry(newTestRegistry(t)), WithCatalog(newTestCatalog()))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return svc
}

func TestService_IntentAliases(t *testing.T) {
	svc := newTestService(t)

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
	svc := newTestService(t)

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
	svc := newTestService(t)

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
	svc := newTestService(t)

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

func TestService_BrokerSlashedWireModel(t *testing.T) {
	// Test that openrouter/openai/gpt-5.4 resolves correctly.
	// OpenRouter is a broker that uses slashed wire model IDs (e.g., "openai/gpt-5.4").
	// When we write "openrouter/openai/gpt-5.4", the parser should NOT interpret
	// "openai" as a service specifier - it should be part of the wire model.

	svc := newTestService(t)

	tests := []struct {
		name         string
		input        string
		wantProvider string
		wantModel    string
	}{
		{
			name:         "openrouter/openai/gpt-5.4 routes to openrouter with slashed wire model",
			input:        "openrouter/openai/gpt-5.4",
			wantProvider: "openrouter",
			wantModel:    "openai/gpt-5.4",
		},
		{
			name:         "openrouter/anthropic/claude-3-5-sonnet routes to openrouter",
			input:        "openrouter/anthropic/claude-3-5-sonnet",
			wantProvider: "openrouter",
			wantModel:    "anthropic/claude-3-5-sonnet",
		},
		{
			name:         "openrouter/meta-llama/llama-3-70b routes to openrouter (unknown service in path)",
			input:        "openrouter/meta-llama/llama-3-70b",
			wantProvider: "openrouter",
			wantModel:    "meta-llama/llama-3-70b",
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

func TestService_InstancePrefixedAliases(t *testing.T) {
	// Test that instance/intent and instance/alias patterns work.
	// For example, "anthropic/fast" should resolve to anthropic's fast intent model,
	// and "openai/mini" should resolve to openai's mini alias.

	svc := newTestService(t)

	tests := []struct {
		name         string
		input        string
		wantProvider string
		wantModel    string
	}{
		// Intent aliases with instance prefix
		{
			name:         "anthropic/fast resolves to anthropic's fast intent",
			input:        "anthropic/fast",
			wantProvider: "anthropic",
			wantModel:    "claude-haiku-4-5-20251001",
		},
		{
			name:         "anthropic/default resolves to anthropic's default intent",
			input:        "anthropic/default",
			wantProvider: "anthropic",
			wantModel:    "claude-sonnet-4-6",
		},
		{
			name:         "anthropic/powerful resolves to anthropic's powerful intent",
			input:        "anthropic/powerful",
			wantProvider: "anthropic",
			wantModel:    "claude-opus-4-6",
		},
		{
			name:         "openai/fast resolves to openai's fast intent",
			input:        "openai/fast",
			wantProvider: "openai",
			wantModel:    "gpt-5.4-nano",
		},
		{
			name:         "openrouter/fast resolves to openrouter's fast intent",
			input:        "openrouter/fast",
			wantProvider: "openrouter",
			wantModel:    "anthropic/claude-3-5-haiku",
		},
		// Provider aliases with instance prefix
		{
			name:         "anthropic/sonnet resolves to anthropic's sonnet alias",
			input:        "anthropic/sonnet",
			wantProvider: "anthropic",
			wantModel:    "claude-sonnet-4-6",
		},
		{
			name:         "anthropic/opus resolves to anthropic's opus alias",
			input:        "anthropic/opus",
			wantProvider: "anthropic",
			wantModel:    "claude-opus-4-6",
		},
		{
			name:         "openai/mini resolves to openai's mini alias",
			input:        "openai/mini",
			wantProvider: "openai",
			wantModel:    "gpt-5.4-mini",
		},
		{
			name:         "openai/gpt resolves to openai's gpt alias",
			input:        "openai/gpt",
			wantProvider: "openai",
			wantModel:    "gpt-5.4",
		},
		{
			name:         "openrouter/gpt-4o resolves to openrouter's gpt-4o alias",
			input:        "openrouter/gpt-4o",
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

	svc := newTestService(t)

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
	svc := newTestService(t)

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
	svc := newTestService(t)

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
	svc := newTestService(t)

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

func TestService_CatalogWireModelPriority(t *testing.T) {
	svc := newTestService(t)

	provider, model, err := svc.ProviderFor("shared-model")
	if err != nil {
		t.Fatalf("ProviderFor(shared-model) error = %v", err)
	}
	if provider.Name() != "anthropic" {
		t.Fatalf("ProviderFor(shared-model) provider = %q, want anthropic", provider.Name())
	}
	if model != "shared-model" {
		t.Fatalf("ProviderFor(shared-model) model = %q, want shared-model", model)
	}
}

func TestService_ServiceModelErrors(t *testing.T) {
	svc := newTestService(t)

	t.Run("unknown service returns provider not found", func(t *testing.T) {
		_, _, err := svc.ProviderFor("missing/model")
		if err == nil {
			t.Fatal("ProviderFor(missing/model) error = nil, want error")
		}
		if !errors.Is(err, ErrProviderNotFound) {
			t.Fatalf("ProviderFor(missing/model) error = %v, want ErrProviderNotFound", err)
		}
	})

	t.Run("service model not in catalog returns model not found", func(t *testing.T) {
		_, _, err := svc.ProviderFor("openai/not-a-model")
		if err == nil {
			t.Fatal("ProviderFor(openai/not-a-model) error = nil, want error")
		}
		if !errors.Is(err, ErrModelNotFound) {
			t.Fatalf("ProviderFor(openai/not-a-model) error = %v, want ErrModelNotFound", err)
		}
	})
}

func TestService_ServiceModelCatalogAlias(t *testing.T) {
	svc := newTestService(t)

	provider, model, err := svc.ProviderFor("openai/gpt-latest")
	if err != nil {
		t.Fatalf("ProviderFor(openai/gpt-latest) error = %v", err)
	}
	if provider.Name() != "openai" {
		t.Fatalf("ProviderFor(openai/gpt-latest) provider = %q, want openai", provider.Name())
	}
	if model != "gpt-latest" {
		t.Fatalf("ProviderFor(openai/gpt-latest) model = %q, want gpt-latest", model)
	}

	models := svc.Models("openai")
	if !containsString(models, "openai/gpt-latest") {
		t.Fatalf("Models(openai) = %v, want openai/gpt-latest", models)
	}
}

func TestService_ResolveByInstanceServiceMismatch(t *testing.T) {
	svc := newTestService(t)

	_, _, err := svc.resolveByInstance(parsedModelRef{
		InstanceName: "openai",
		ServiceID:    "anthropic",
		WireModel:    "claude-sonnet-4-6",
	})
	if err == nil {
		t.Fatal("resolveByInstance(...) error = nil, want error")
	}
	if !errors.Is(err, ErrProviderNotFound) {
		t.Fatalf("resolveByInstance(...) error = %v, want ErrProviderNotFound", err)
	}
	if !strings.Contains(err.Error(), `instance "openai" is service "openai", not "anthropic"`) {
		t.Fatalf("resolveByInstance(...) error = %v, want service mismatch detail", err)
	}
}

func TestService_ParseModelRef(t *testing.T) {
	svc := newTestService(t)

	tests := []struct {
		name  string
		input string
		want  parsedModelRef
	}{
		{
			name:  "bare model",
			input: "gpt-5.4",
			want:  parsedModelRef{WireModel: "gpt-5.4"},
		},
		{
			name:  "known service prefix",
			input: "openai/gpt-5.4",
			want:  parsedModelRef{ServiceID: "openai", WireModel: "gpt-5.4"},
		},
		{
			name:  "known service prefix wins when instance and service overlap",
			input: "openrouter/gpt-4o",
			want:  parsedModelRef{ServiceID: "openrouter", WireModel: "gpt-4o"},
		},
		{
			name:  "instance service model",
			input: "openai/openai/gpt-5.4",
			want:  parsedModelRef{InstanceName: "openai", ServiceID: "openai", WireModel: "gpt-5.4"},
		},
		{
			name:  "broker slash wire model remains wire model",
			input: "openrouter/openai/gpt-5.4",
			want:  parsedModelRef{InstanceName: "openrouter", WireModel: "openai/gpt-5.4"},
		},
		{
			name:  "unknown two part input treated as service model",
			input: "custom/model-x",
			want:  parsedModelRef{ServiceID: "custom", WireModel: "model-x"},
		},
		{
			name:  "unknown multi part input treated as bare model",
			input: "custom/vendor/model-x",
			want:  parsedModelRef{WireModel: "custom/vendor/model-x"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := svc.parseModelRef(tt.input); got != tt.want {
				t.Fatalf("parseModelRef(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func containsString(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
