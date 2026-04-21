package auto

import (
	"testing"

	"github.com/codewandler/llmproviders/providers/anthropic"
	"github.com/codewandler/llmproviders/providers/dockermr"
	"github.com/codewandler/llmproviders/providers/minimax"
	"github.com/codewandler/llmproviders/providers/ollama"
	"github.com/codewandler/llmproviders/providers/openai"
	"github.com/codewandler/llmproviders/providers/openai/codex"
	"github.com/codewandler/llmproviders/providers/openrouter"
)

func TestNewAutoDetectRegistry_InstanceNamesOrderedByPriority(t *testing.T) {
	reg := NewAutoDetectRegistry()

	got := reg.InstanceNames()
	want := []string{
		codex.ProviderName,
		"claude",
		anthropic.ProviderName,
		openai.ProviderName,
		openrouter.ProviderName,
		minimax.ProviderName,
		ollama.ProviderName,
		dockermr.ProviderName,
	}

	if len(got) != len(want) {
		t.Fatalf("InstanceNames() len = %d, want %d; values = %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("InstanceNames()[%d] = %q, want %q; values = %v", i, got[i], want[i], got)
		}
	}
}

func TestNewAutoDetectRegistry_ServiceIDs(t *testing.T) {
	reg := NewAutoDetectRegistry()

	got := reg.ServiceIDs()
	want := []string{
		anthropic.ServiceID,
		codex.ServiceID,
		dockermr.ServiceID,
		minimax.ServiceID,
		ollama.ServiceID,
		openai.ServiceID,
		openrouter.ServiceID,
	}

	if len(got) != len(want) {
		t.Fatalf("ServiceIDs() len = %d, want %d; values = %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ServiceIDs()[%d] = %q, want %q; values = %v", i, got[i], want[i], got)
		}
	}
}

func TestNewAutoDetectRegistry_AnthropicPrecedesOpenAIProviders(t *testing.T) {
	reg := NewAutoDetectRegistry()
	got := reg.InstanceNames()

	indexOf := func(name string) int {
		for i, v := range got {
			if v == name {
				return i
			}
		}
		return -1
	}

	codexIdx := indexOf(codex.ProviderName)
	claudeIdx := indexOf("claude")
	anthropicIdx := indexOf(anthropic.ProviderName)
	openAIIdx := indexOf(openai.ProviderName)
	openRouterIdx := indexOf(openrouter.ProviderName)

	if codexIdx == -1 || claudeIdx == -1 || anthropicIdx == -1 || openAIIdx == -1 || openRouterIdx == -1 {
		t.Fatalf("expected core providers to be registered, got %v", got)
	}

	if !(codexIdx < openAIIdx && codexIdx < openRouterIdx) {
		t.Fatalf("expected codex to precede openai-style fallbacks, got %v", got)
	}
	if !(claudeIdx < anthropicIdx) {
		t.Fatalf("expected claude oauth to precede anthropic api key provider, got %v", got)
	}
	if !(anthropicIdx < openAIIdx && openAIIdx < openRouterIdx) {
		t.Fatalf("expected anthropic < openai < openrouter priority order, got %v", got)
	}
}
