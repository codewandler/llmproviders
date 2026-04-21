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

	reg.Register(anthropic.Register)
	reg.Register(openai.Register)
	reg.Register(openrouter.Register)
	reg.Register(minimax.Register)
	reg.Register(codex.Register)
	reg.Register(ollama.Register)
	reg.Register(dockermr.Register)

	return reg
}
