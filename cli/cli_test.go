package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/codewandler/agentapis/conversation"
	llmproviders "github.com/codewandler/llmproviders"
	"github.com/codewandler/llmproviders/registry"
	"github.com/codewandler/modeldb"
)

type testProvider struct{ name string }

func (p *testProvider) Name() string                                                    { return p.name }
func (p *testProvider) CreateSession(opts ...conversation.Option) *conversation.Session { return nil }

func testService(t *testing.T) *llmproviders.Service {
	t.Helper()

	reg := registry.New()
	reg.Register(registry.Registration{
		InstanceName: "anthropic",
		ServiceID:    "anthropic",
		Order:        20,
		Aliases: map[string]string{
			"sonnet": "claude-sonnet-4-6",
			"opus":   "claude-opus-4-6",
		},
		IntentAliases: map[string]string{
			llmproviders.IntentFast:     "claude-haiku-4-5-20251001",
			llmproviders.IntentDefault:  "claude-sonnet-4-6",
			llmproviders.IntentPowerful: "claude-opus-4-6",
		},
		Detect: func(ctx context.Context) (bool, error) { return true, nil },
		Build: func(ctx context.Context, cfg registry.BuildConfig) (registry.Provider, error) {
			return &testProvider{name: "anthropic"}, nil
		},
	})
	reg.Register(registry.Registration{
		InstanceName: "openai",
		ServiceID:    "openai",
		Order:        30,
		Aliases: map[string]string{
			"mini": "gpt-5.4-mini",
		},
		IntentAliases: map[string]string{
			llmproviders.IntentFast:     "gpt-5.4-nano",
			llmproviders.IntentDefault:  "gpt-5.4-mini",
			llmproviders.IntentPowerful: "o3",
		},
		Detect: func(ctx context.Context) (bool, error) { return true, nil },
		Build: func(ctx context.Context, cfg registry.BuildConfig) (registry.Provider, error) {
			return &testProvider{name: "openai"}, nil
		},
	})
	reg.Register(registry.Registration{
		InstanceName: "openrouter",
		ServiceID:    "openrouter",
		Order:        50,
		Aliases: map[string]string{
			"gpt-4o": "openai/gpt-4o",
		},
		IntentAliases: map[string]string{
			llmproviders.IntentDefault: "anthropic/claude-3.7-sonnet",
		},
		Detect: func(ctx context.Context) (bool, error) { return true, nil },
		Build: func(ctx context.Context, cfg registry.BuildConfig) (registry.Provider, error) {
			return &testProvider{name: "openrouter"}, nil
		},
	})

	catalog := modeldb.NewCatalog()
	for _, offering := range []modeldb.Offering{
		{ServiceID: "anthropic", WireModelID: "claude-sonnet-4-6"},
		{ServiceID: "anthropic", WireModelID: "claude-opus-4-6"},
		{ServiceID: "anthropic", WireModelID: "claude-haiku-4-5-20251001"},
		{ServiceID: "openai", WireModelID: "gpt-5.4-mini"},
		{ServiceID: "openai", WireModelID: "gpt-5.4-nano"},
		{ServiceID: "openai", WireModelID: "o3"},
		{ServiceID: "openrouter", WireModelID: "openai/gpt-4o"},
		{ServiceID: "openrouter", WireModelID: "anthropic/claude-3.7-sonnet"},
	} {
		catalog.Offerings[modeldb.OfferingRef{ServiceID: offering.ServiceID, WireModelID: offering.WireModelID}] = offering
	}

	svc, err := llmproviders.NewService(llmproviders.WithRegistry(reg), llmproviders.WithCatalog(catalog))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return svc
}

func TestFormatHelpers(t *testing.T) {
	if got := checkMark(true); got != "yes" {
		t.Fatalf("checkMark(true) = %q", got)
	}
	if got := checkMark(false); got != "no" {
		t.Fatalf("checkMark(false) = %q", got)
	}
	if got := checkSymbol(true); got != "[x]" {
		t.Fatalf("checkSymbol(true) = %q", got)
	}
	if got := checkSymbol(false); got != "[ ]" {
		t.Fatalf("checkSymbol(false) = %q", got)
	}

	got := sortedMapKeys(map[string]int{"b": 2, "a": 1, "c": 3})
	want := []string{"a", "b", "c"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sortedMapKeys() = %v, want %v", got, want)
		}
	}
}

func TestPrintModels(t *testing.T) {
	svc := testService(t)

	t.Run("query searches across services", func(t *testing.T) {
		var out bytes.Buffer
		if err := printModels(&out, svc, "", "sonnet", true); err != nil {
			t.Fatalf("printModels() error = %v", err)
		}
		text := out.String()
		for _, want := range []string{
			"AVAILABLE MODELS",
			"anthropic/claude-sonnet-4-6  [default]",
			"openrouter/anthropic/claude-3.7-sonnet",
			"2 models available",
			"Models marked with [intent]",
		} {
			if !strings.Contains(text, want) {
				t.Fatalf("printModels() output missing %q:\n%s", want, text)
			}
		}
	})

	t.Run("service filter narrows model list", func(t *testing.T) {
		var out bytes.Buffer
		if err := printModels(&out, svc, "anthropic", "sonnet", true); err != nil {
			t.Fatalf("printModels() error = %v", err)
		}
		text := out.String()
		for _, want := range []string{
			"MODELS FOR ANTHROPIC",
			"anthropic/claude-sonnet-4-6  [default]",
			"1 models available",
		} {
			if !strings.Contains(text, want) {
				t.Fatalf("printModels() output missing %q:\n%s", want, text)
			}
		}
		if strings.Contains(text, "openrouter/") {
			t.Fatalf("printModels() output unexpectedly included openrouter models:\n%s", text)
		}
	})
}

func TestPrintAliasesMergedAndJSON(t *testing.T) {
	svc := testService(t)
	var out bytes.Buffer
	if err := printAliasesMerged(&out, svc); err != nil {
		t.Fatalf("printAliasesMerged() error = %v", err)
	}
	text := out.String()
	for _, want := range []string{"MERGED ALIASES", "mini", "sonnet", "Use `llmcli aliases --by-provider`"} {
		if !strings.Contains(text, want) {
			t.Fatalf("printAliasesMerged() output missing %q:\n%s", want, text)
		}
	}

	out.Reset()
	if err := printAliasesJSON(&out, svc); err != nil {
		t.Fatalf("printAliasesJSON() error = %v", err)
	}
	jsonText := out.String()
	for _, want := range []string{`"aliases"`, `"sonnet"`, `"service": "anthropic"`} {
		if !strings.Contains(jsonText, want) {
			t.Fatalf("printAliasesJSON() output missing %q:\n%s", want, jsonText)
		}
	}
}

func TestPrintAliasesByProvider(t *testing.T) {
	svc := testService(t)
	var out bytes.Buffer
	if err := printAliasesByProvider(&out, svc); err != nil {
		t.Fatalf("printAliasesByProvider() error = %v", err)
	}
	text := out.String()
	for _, want := range []string{"ALIASES BY PROVIDER", "anthropic (detected)", "openai (detected)", "openrouter (detected)", "sonnet", "mini", "gpt-4o"} {
		if !strings.Contains(text, want) {
			t.Fatalf("printAliasesByProvider() output missing %q:\n%s", want, text)
		}
	}
}

func TestPrintProvidersAndJSON(t *testing.T) {
	svc := testService(t)
	var out bytes.Buffer
	if err := printProviders(&out, svc, false); err != nil {
		t.Fatalf("printProviders() error = %v", err)
	}
	text := out.String()
	for _, want := range []string{"DETECTED PROVIDERS", "anthropic", "openai", "3 providers detected"} {
		if !strings.Contains(text, want) {
			t.Fatalf("printProviders() output missing %q:\n%s", want, text)
		}
	}

	out.Reset()
	if err := printProvidersJSON(&out, svc); err != nil {
		t.Fatalf("printProvidersJSON() error = %v", err)
	}
	jsonText := out.String()
	for _, want := range []string{`"providers"`, `"instance": "anthropic"`, `"detected": true`} {
		if !strings.Contains(jsonText, want) {
			t.Fatalf("printProvidersJSON() output missing %q:\n%s", want, jsonText)
		}
	}
}

func TestPrintProvidersUsesRealServiceIDs(t *testing.T) {
	reg := registry.New()
	reg.Register(registry.Registration{
		InstanceName: "claude",
		ServiceID:    "anthropic",
		Order:        15,
		Detect:       func(ctx context.Context) (bool, error) { return true, nil },
		Build: func(ctx context.Context, cfg registry.BuildConfig) (registry.Provider, error) {
			return &testProvider{name: "claude"}, nil
		},
	})
	catalog := modeldb.NewCatalog()
	svc, err := llmproviders.NewService(llmproviders.WithRegistry(reg), llmproviders.WithCatalog(catalog))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	var out bytes.Buffer
	if err := printProviders(&out, svc, false); err != nil {
		t.Fatalf("printProviders() error = %v", err)
	}
	text := out.String()
	for _, want := range []string{"claude", "anthropic", "detected"} {
		if !strings.Contains(text, want) {
			t.Fatalf("printProviders() output missing %q:\n%s", want, text)
		}
	}

	out.Reset()
	if err := printProvidersJSON(&out, svc); err != nil {
		t.Fatalf("printProvidersJSON() error = %v", err)
	}
	jsonText := out.String()
	for _, want := range []string{`"instance": "claude"`, `"service": "anthropic"`} {
		if !strings.Contains(jsonText, want) {
			t.Fatalf("printProvidersJSON() output missing %q:\n%s", want, jsonText)
		}
	}
}

func TestResolveWithDetailsAndPrintResolve(t *testing.T) {
	svc := testService(t)

	result := resolveWithDetails(svc, "sonnet")
	if result.Type != "provider_alias" || result.InstanceName != "anthropic" || result.WireModelID != "claude-sonnet-4-6" {
		t.Fatalf("resolveWithDetails(sonnet) = %+v", result)
	}

	bare := resolveWithDetails(svc, "gpt-4o")
	if bare.Type != "provider_alias" || bare.InstanceName != "openrouter" {
		t.Fatalf("resolveWithDetails(gpt-4o) = %+v", bare)
	}

	var out bytes.Buffer
	if err := printResolve(&out, svc, "fast"); err != nil {
		t.Fatalf("printResolve() error = %v", err)
	}
	text := out.String()
	for _, want := range []string{"Resolving: fast", "Parsed as:    bare model", "Type:         Intent alias", "Instance:     anthropic", "Service ID:   anthropic", "Resolution path:"} {
		if !strings.Contains(text, want) {
			t.Fatalf("printResolve() output missing %q:\n%s", want, text)
		}
	}

	out.Reset()
	if err := printResolve(&out, svc, "anthropic/sonnet"); err != nil {
		t.Fatalf("printResolve() error = %v", err)
	}
	text = out.String()
	for _, want := range []string{"Parsed as:    service/model", "Prefix:       anthropic", "Instance:     anthropic", "Service ID:   anthropic"} {
		if !strings.Contains(text, want) {
			t.Fatalf("printResolve() output missing %q:\n%s", want, text)
		}
	}
}

func TestPrintIntentsActiveAndJSON(t *testing.T) {
	svc := testService(t)
	var out bytes.Buffer
	if err := printIntentsActive(&out, svc); err != nil {
		t.Fatalf("printIntentsActive() error = %v", err)
	}
	text := out.String()
	for _, want := range []string{"ACTIVE INTENT ALIASES", "fast", "default", "powerful", "Note: anthropic has highest priority"} {
		if !strings.Contains(text, want) {
			t.Fatalf("printIntentsActive() output missing %q:\n%s", want, text)
		}
	}

	out.Reset()
	if err := printIntentsJSON(&out, svc, false, nil, context.Background()); err != nil {
		t.Fatalf("printIntentsJSON() error = %v", err)
	}
	jsonText := out.String()
	for _, want := range []string{`"active"`, `"fast"`, `"provider": "anthropic"`} {
		if !strings.Contains(jsonText, want) {
			t.Fatalf("printIntentsJSON() output missing %q:\n%s", want, jsonText)
		}
	}
}

func TestNewLLMCommand(t *testing.T) {
	cmd := NewLLMCommand(LLMCommandOptions{
		LoadService: func(ctx context.Context) (*llmproviders.Service, error) { return testService(t), nil },
		IO:          IO{In: bytes.NewBuffer(nil), Out: &bytes.Buffer{}, Err: &bytes.Buffer{}},
	})
	if cmd.Use != "llm" {
		t.Fatalf("Use = %q, want llm", cmd.Use)
	}
	for _, sub := range []string{"infer", "serve", "opencode", "models", "intents", "providers", "aliases", "resolve", "skill"} {
		if _, _, err := cmd.Find([]string{sub}); err != nil {
			t.Fatalf("expected subcommand %q: %v", sub, err)
		}
	}
}

func TestResolveAddr(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{":8080", "localhost:8080"},
		{"0.0.0.0:3000", "0.0.0.0:3000"},
		{"localhost:9090", "localhost:9090"},
		{"invalid", "invalid"},
	}
	for _, tt := range tests {
		got := resolveAddr(tt.input)
		if got != tt.want {
			t.Errorf("resolveAddr(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
