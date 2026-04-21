package auto

import (
	"github.com/codewandler/llmproviders/providers/anthropic"
	"github.com/codewandler/llmproviders/providers/dockermr"
	"github.com/codewandler/llmproviders/providers/minimax"
	"github.com/codewandler/llmproviders/providers/ollama"
	"github.com/codewandler/llmproviders/providers/openai"
	"github.com/codewandler/llmproviders/providers/openai/codex"
	"github.com/codewandler/llmproviders/providers/openrouter"
	"github.com/codewandler/llmproviders/registry"
)

func NewAutoDetectRegistry() *registry.Registry {
	reg := registry.New()

	// Registration order doesn't matter - providers are sorted by Order value.
	// Priority order (lower = higher priority):
	//   codex:10, claude:15, anthropic:20, openai:30, openrouter:50,
	//   minimax:60, ollama:70, dockermr:90

	reg.Register(codex.Register)           // Order 10 - ChatGPT OAuth
	reg.Register(anthropic.ClaudeRegister) // Order 15 - Claude.ai OAuth
	reg.Register(anthropic.Register)       // Order 20 - Anthropic API key
	reg.Register(openai.Register)          // Order 30 - OpenAI API key
	reg.Register(openrouter.Register)      // Order 50 - OpenRouter fallback
	reg.Register(minimax.Register)         // Order 60
	reg.Register(ollama.Register)          // Order 70 - local, probed
	reg.Register(dockermr.Register)        // Order 90 - local, probed

	return reg
}
