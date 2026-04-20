package codex

import (
	"fmt"
	"net/http"
	"os"
)

const (
	accountIDHeader = "ChatGPT-Account-ID"
	codexBetaHeader = "OpenAI-Beta"
	codexBetaValue  = "responses=experimental"
	codexOriginator = "codex_cli_rs"
)

func APIKeyFromEnv() string {
	if v := os.Getenv("CODEX_API_KEY"); v != "" {
		return v
	}
	return os.Getenv("OPENAI_API_KEY")
}

func ApplyHeaders(h http.Header, cfg Config) error {
	if cfg.APIKey == "" {
		return fmt.Errorf("codex: API key is required")
	}
	h.Set("Authorization", "Bearer "+cfg.APIKey)
	h.Set(codexBetaHeader, codexBetaValue)
	h.Set("originator", codexOriginator)
	if accountID := os.Getenv("CODEX_ACCOUNT_ID"); accountID != "" {
		h.Set(accountIDHeader, accountID)
	}
	return nil
}
