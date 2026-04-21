package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
)

// Model represents a Codex model.
type Model struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Models is a collection of Codex models with lookup capabilities.
type Models []Model

// Resolve looks up a model by ID or alias.
// Returns the model and true if found, otherwise a fallback model and false.
func (m Models) Resolve(id string) (Model, bool) {
	for _, model := range m {
		if model.ID == id {
			return model, true
		}
	}
	// Check aliases
	if canonical, ok := modelAliases[id]; ok {
		for _, model := range m {
			if model.ID == canonical {
				return model, true
			}
		}
	}
	// Return fallback
	return Model{ID: id, Name: id}, false
}

// Wire model ID constants for programmatic use.
const (
	ModelGPT54     = "gpt-5.4"
	ModelGPT54Mini = "gpt-5.4-mini"
	ModelGPT53     = "gpt-5.3-codex"
	ModelO3        = "o3"
	ModelO4Mini    = "o4-mini"
)

// Alias constants for short names.
const (
	AliasCodex = "codex"
	AliasMini  = "mini"
	AliasO3    = "o3"
)

// ProviderAliases maps short names to wire model IDs for this provider.
var ProviderAliases = map[string]string{
	AliasCodex: ModelGPT54,
	AliasMini:  ModelGPT54Mini,
	AliasO3:    ModelO3,
}

// modelAliases maps common aliases to canonical model IDs (for internal use).
var modelAliases = map[string]string{
	"codex":   ModelGPT54,
	"default": ModelGPT54,
	"fast":    ModelGPT54Mini,
	"mini":    ModelGPT54Mini,
}

// LoadModels returns the embedded fallback model list.
func LoadModels() Models {
	return fallbackModels()
}

// fallbackModels returns a static list of known Codex models.
// These are based on the model slugs returned by the Codex API.
func fallbackModels() Models {
	return Models{
		{ID: "gpt-5.4", Name: "GPT-5.4", Description: "Default Codex model"},
		{ID: "gpt-5.4-mini", Name: "GPT-5.4 Mini", Description: "Faster, lighter Codex model"},
		{ID: "gpt-5.3-codex", Name: "GPT-5.3 Codex", Description: "Previous generation Codex model"},
		{ID: "gpt-5.3-codex-spark", Name: "GPT-5.3 Codex Spark", Description: "Fast previous generation model"},
		{ID: "o3", Name: "o3", Description: "OpenAI o3 reasoning model"},
		{ID: "o4-mini", Name: "o4-mini", Description: "OpenAI o4-mini model"},
	}
}

// modelsResponse is the API response structure for /codex/models.
type modelsResponse struct {
	Models []apiModel `json:"models"`
}

// apiModel is a single model from the API response.
type apiModel struct {
	Slug         string `json:"slug"`
	DisplayName  string `json:"display_name"`
	MaxTokens    int    `json:"max_tokens,omitempty"`
	IsDefault    bool   `json:"is_default,omitempty"`
	Capabilities struct {
		Reasoning bool `json:"reasoning,omitempty"`
		Vision    bool `json:"vision,omitempty"`
	} `json:"capabilities,omitempty"`
}

// FetchModels fetches available models from the Codex API.
func FetchModels(ctx context.Context, auth *Auth, httpClient *http.Client) ([]Model, error) {
	raw, err := FetchRawModels(ctx, auth, httpClient)
	if err != nil {
		return nil, err
	}

	var payload modelsResponse
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("codex: decode models response: %w", err)
	}

	models := make([]Model, 0, len(payload.Models))
	for _, m := range payload.Models {
		name := m.DisplayName
		if name == "" {
			name = m.Slug
		}
		models = append(models, Model{
			ID:   m.Slug,
			Name: name,
		})
	}

	sort.Slice(models, func(i, j int) bool {
		return models[i].ID < models[j].ID
	})

	return models, nil
}

// FetchRawModels fetches the raw JSON response from the Codex models endpoint.
func FetchRawModels(ctx context.Context, auth *Auth, httpClient *http.Client) ([]byte, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	token, err := auth.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("codex: get token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ModelsEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("codex: create models request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	if accountID := auth.AccountID(); accountID != "" {
		req.Header.Set(HeaderChatGPTAccountID, accountID)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("codex: list models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("codex: list models failed (status %d): %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}
