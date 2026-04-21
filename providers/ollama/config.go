// Package ollama provides an Ollama local LLM provider.
//
// Ollama is a local LLM server that runs models on your machine.
// This provider wraps the native Ollama API via agentapis.
//
// # Quick Start
//
//	provider, err := ollama.New()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	session := provider.Session()
//
// # Requirements
//
// Ollama must be running locally on the default port (11434).
// Install and start Ollama: https://ollama.ai
//
// Pull a model before use:
//
//	ollama pull qwen2.5:0.5b
//
// # Environment Variables
//
// Set OLLAMA_URL to override the default base URL.
package ollama

import "os"

// Provider name constant.
const (
	// ProviderName is the identifier for this provider instance.
	ProviderName = "ollama"

	// ServiceID is the modeldb service identifier for this provider.
	ServiceID = "ollama"
)

// Environment variable names.
const (
	// EnvOllamaURL overrides the default Ollama API base URL.
	EnvOllamaURL = "OLLAMA_URL"
)

// EnvOllamaURLValue returns the value of OLLAMA_URL environment variable.
func EnvOllamaURLValue() string {
	return os.Getenv(EnvOllamaURL)
}

// ResolveBaseURL returns the Ollama API base URL.
// Uses OLLAMA_URL env var if set, otherwise returns DefaultBaseURL.
func ResolveBaseURL() string {
	if url := EnvOllamaURLValue(); url != "" {
		return url
	}
	return DefaultBaseURL
}

// Default configuration values.
const (
	// DefaultBaseURL is the default Ollama API endpoint.
	DefaultBaseURL = "http://localhost:11434"

	// DefaultModel is the default model to use.
	// llama3.2:1b is small (1B params) but capable enough for basic chat.
	DefaultModel = "llama3.2:1b"
)

// Model ID constants for commonly used Ollama models.
const (
	// Small models (< 2B params) - fast, low memory
	ModelQwen2505B      = "qwen2.5:0.5b"      // 0.5B params - recommended for testing
	ModelQwen306B       = "qwen3:0.6b"        // 0.6B params
	ModelLlama321B      = "llama3.2:1b"       // 1B params
	ModelQwen317B       = "qwen3:1.7b"        // 1.7B params
	ModelGranite31MoE1B = "granite3.1-moe:1b" // 1B params MoE

	// Medium models (3-8B params)
	ModelLlama32      = "llama3.2"       // 3B params default
	ModelLlama31      = "llama3.1"       // 8B params default
	ModelQwen25       = "qwen2.5"        // 7B params default
	ModelPhi3         = "phi3"           // 3.8B params
	ModelMistral      = "mistral"        // 7B params
	ModelGemma3       = "gemma3"         // 4B params default
	ModelGLM47Flash   = "glm-4.7-flash"  // Flash model
	ModelMinistral38B = "ministral-3:8b" // 8B params

	// Specialized models
	ModelDeepSeekR1     = "deepseek-r1"      // Reasoning model
	ModelFunctionGemma  = "functiongemma"    // Function calling optimized
	ModelDevstralSmall2 = "devstral-small-2" // Code model
)

// Config holds the Ollama provider configuration.
type Config struct {
	// BaseURL is the Ollama API endpoint URL.
	BaseURL string

	// Model is the default model to use.
	Model string
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() Config {
	return Config{
		BaseURL: DefaultBaseURL,
		Model:   DefaultModel,
	}
}
