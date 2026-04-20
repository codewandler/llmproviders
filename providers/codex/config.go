package codex

import (
	"fmt"
	"os"
	"time"
)

const (
	defaultBaseURL = "https://chatgpt.com/backend-api"
	defaultModel   = "codex-mini-latest"
)

// Config contains bootstrap settings for codex.
type Config struct {
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
}

func DefaultConfig() Config {
	return Config{
		BaseURL: defaultBaseURL,
		Model:   defaultModel,
		Timeout: 60 * time.Second,
	}
}

func FromEnv() Config {
	cfg := DefaultConfig()
	if v := os.Getenv("CODEX_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}
	if v := os.Getenv("CODEX_API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("OPENAI_API_KEY")
	}
	if v := os.Getenv("CODEX_MODEL"); v != "" {
		cfg.Model = v
	}
	return cfg
}

func (c Config) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("codex: base URL is required")
	}
	if c.APIKey == "" {
		return fmt.Errorf("codex: API key is required")
	}
	return nil
}
