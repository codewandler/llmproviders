// Package dockermr provides a Docker Model Runner local LLM provider.
//
// Docker Model Runner serves local LLM models via the llama.cpp engine.
// This provider wraps the OpenAI Chat Completions API exposed by the runner.
//
// # Quick Start
//
//	provider, err := dockermr.New()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	session := provider.Session()
//
// # Requirements
//
// Docker Model Runner must be running locally on the default port (12434).
// Models are pulled from Docker Hub's ai/ namespace.
//
// Pull a model before use:
//
//	docker model pull ai/smollm2
package dockermr

// Provider name constant.
const ProviderName = "dockermr"

// Default configuration values.
const (
	// DefaultBaseURL is the default Docker Model Runner API endpoint.
	DefaultBaseURL = "http://localhost:12434"

	// ContainerBaseURL is the base URL when running inside a Docker container.
	ContainerBaseURL = "http://model-runner.docker.internal"

	// DefaultEngine is the default inference engine.
	DefaultEngine = "llama.cpp"

	// DefaultModel is the default model to use.
	// ai/llama3.2 is small (3B params) but capable enough for basic chat.
	DefaultModel = "ai/llama3.2"
)

// Model ID constants for Docker Hub's ai/ namespace models.
// Model IDs use the format ai/<name>:<tag>.
// Tags follow the pattern <size>-<quantization> (e.g. 7B-Q4_K_M).
// Using the bare name without a tag resolves to the recommended default.
const (
	// Tiny models (< 500M params) - very fast, minimal memory
	ModelSmoLLM2Tiny = "ai/smollm2:135M-Q4_K_M" // 135M params (smallest)
	ModelSmoLLM2     = "ai/smollm2"             // 360M params (default)

	// Small models (0.5-1B params)
	ModelQwen25Small = "ai/qwen2.5:0.5B-F16" // 0.5B params

	// Medium models (3-8B params)
	ModelQwen25        = "ai/qwen2.5"          // 7B params default
	ModelQwen3         = "ai/qwen3"            // 7B params
	ModelQwen3Coder    = "ai/qwen3-coder"      // Code-optimized
	ModelLlama32       = "ai/llama3.2"         // 3B params
	ModelLlama33       = "ai/llama3.3"         // 8B params
	ModelPhi4Mini      = "ai/phi4-mini"        // Mini version
	ModelPhi4          = "ai/phi4"             // Full version
	ModelGemma3        = "ai/gemma3"           // 4B params
	ModelGemma4        = "ai/gemma4"           // Gemma 4
	ModelMistralSmall  = "ai/mistral-small3.2" // Small Mistral
	ModelGLM47Flash    = "ai/glm-4.7-flash"    // Flash model
	ModelGranite4Nano  = "ai/granite4.0-nano"  // Nano version
	ModelFunctionGemma = "ai/functiongemma"    // Function calling

	// Large models
	ModelDeepSeekR1 = "ai/deepseek-r1" // Reasoning model
)

// Config holds the Docker Model Runner provider configuration.
type Config struct {
	// BaseURL is the Docker Model Runner API endpoint URL.
	BaseURL string

	// Engine is the inference engine to use (e.g., "llama.cpp").
	Engine string

	// Model is the default model to use.
	Model string
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() Config {
	return Config{
		BaseURL: DefaultBaseURL,
		Engine:  DefaultEngine,
		Model:   DefaultModel,
	}
}

// EngineURL returns the full URL for the given engine.
// Format: {baseURL}/engines/{engine}
func EngineURL(baseURL, engine string) string {
	if engine == "" {
		engine = DefaultEngine
	}
	return baseURL + "/engines/" + engine
}
