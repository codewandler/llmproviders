package llmproviders

import (
	"github.com/codewandler/llmproviders/registry"
)

// Provider is the interface that all LLM providers must implement.
// This is a type alias to registry.Provider for convenience - users can import
// just llmproviders instead of llmproviders/registry.
//
// Provider implementations must support:
//   - Name(): Returns the unique instance name (e.g., "anthropic", "openai")
//   - CreateSession(): Creates a conversation session for streaming responses
type Provider = registry.Provider
