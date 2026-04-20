package openrouter

import (
	_ "embed"
	"encoding/json"
	"strings"
)

//go:embed models.json
var modelsJSON []byte

// ModelData represents the full structure of a model from OpenRouter API.
type ModelData struct {
	ID            string `json:"id"`
	CanonicalSlug string `json:"canonical_slug"`
	HuggingFaceID string `json:"hugging_face_id"`
	Name          string `json:"name"`
	Created       int64  `json:"created"`
	Description   string `json:"description"`
	ContextLength int    `json:"context_length"`
	Architecture  struct {
		Modality         string   `json:"modality"`
		InputModalities  []string `json:"input_modalities"`
		OutputModalities []string `json:"output_modalities"`
		Tokenizer        string   `json:"tokenizer"`
		InstructType     *string  `json:"instruct_type"`
	} `json:"architecture"`
	Pricing struct {
		Prompt         string `json:"prompt"`
		Completion     string `json:"completion"`
		InputCacheRead string `json:"input_cache_read"`
	} `json:"pricing"`
	TopProvider struct {
		ContextLength       int  `json:"context_length"`
		MaxCompletionTokens int  `json:"max_completion_tokens"`
		IsModerated         bool `json:"is_moderated"`
	} `json:"top_provider"`
	PerRequestLimits    interface{}            `json:"per_request_limits"`
	SupportedParameters []string               `json:"supported_parameters"`
	DefaultParameters   map[string]interface{} `json:"default_parameters"`
}

// Model represents an OpenRouter model.
type Model struct {
	ID       string
	Name     string
	Provider string
	Aliases  []string
}

// Models is a list of available models.
type Models []Model

// Resolve resolves a model alias or ID to a Model.
func (m Models) Resolve(alias string) (Model, error) {
	if alias == "" || alias == "default" || alias == "openrouter" {
		// Return the auto model
		for _, model := range m {
			if model.ID == "openrouter/auto" {
				return model, nil
			}
		}
		return Model{ID: DefaultModel, Name: "OpenRouter Auto", Provider: ProviderName}, nil
	}

	lower := strings.ToLower(alias)

	// Check for aliases
	for _, model := range m {
		if strings.EqualFold(model.ID, alias) {
			return model, nil
		}
		for _, a := range model.Aliases {
			if strings.EqualFold(a, lower) {
				return model, nil
			}
		}
	}

	// Return as-is for unknown models (may be valid but not in registry)
	return Model{ID: alias, Name: alias, Provider: ProviderName}, nil
}

// LoadModels loads the curated list of tool-enabled models from the embedded models.json file.
func LoadModels() Models {
	var models []ModelData

	if err := json.Unmarshal(modelsJSON, &models); err != nil {
		// If we can't parse the embedded file, return minimal list
		return Models{{
			ID:       "openrouter/auto",
			Name:     "OpenRouter Auto",
			Provider: ProviderName,
			Aliases:  []string{"default", "auto", "fast"},
		}}
	}

	result := make(Models, len(models))
	for i, m := range models {
		model := Model{
			ID:       m.ID,
			Name:     m.Name,
			Provider: ProviderName,
		}
		if m.ID == "openrouter/auto" {
			model.Aliases = []string{"default", "auto", "fast", "openrouter"}
		}
		result[i] = model
	}

	return result
}

// GetModelData returns the full model data from the embedded models.json file.
func GetModelData() ([]ModelData, error) {
	var models []ModelData
	if err := json.Unmarshal(modelsJSON, &models); err != nil {
		return nil, err
	}
	return models, nil
}

// SupportsPromptCaching reports whether the model supports prompt caching.
func SupportsPromptCaching(model string) bool {
	models, err := GetModelData()
	if err != nil {
		return false
	}
	for _, m := range models {
		if m.ID == model {
			return m.Pricing.InputCacheRead != "" && m.Pricing.InputCacheRead != "0" && m.Pricing.InputCacheRead != "0.0"
		}
	}
	return false
}

// UseMessagesAPI reports whether the model should use the Anthropic Messages API.
// Models prefixed with "anthropic/" are routed to the Messages API.
func UseMessagesAPI(model string) bool {
	return strings.HasPrefix(model, AnthropicModelPrefix)
}

// StripAnthropicPrefix removes the "anthropic/" prefix from a model ID.
func StripAnthropicPrefix(model string) string {
	return strings.TrimPrefix(model, AnthropicModelPrefix)
}
