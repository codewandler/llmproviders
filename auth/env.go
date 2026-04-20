package auth

import "os"

func OpenAIAPIKeyFromEnv() string     { return os.Getenv("OPENAI_API_KEY") }
func AnthropicAPIKeyFromEnv() string  { return os.Getenv("ANTHROPIC_API_KEY") }
func OpenRouterAPIKeyFromEnv() string { return os.Getenv("OPENROUTER_API_KEY") }
func OllamaBaseURLFromEnv() string    { return os.Getenv("OLLAMA_BASE_URL") }
