package openai

import (
	"fmt"
	"strings"
)

// Model ID constants for programmatic use.
const (
	// GPT-5.4 series (flagship, latest).
	ModelGPT54     = "gpt-5.4"
	ModelGPT54Mini = "gpt-5.4-mini"
	ModelGPT54Nano = "gpt-5.4-nano"
	ModelGPT54Pro  = "gpt-5.4-pro"

	// GPT-5.3 series.
	ModelGPT53Codex = "gpt-5.3-codex"

	// GPT-5.2 series.
	ModelGPT52      = "gpt-5.2"
	ModelGPT52Pro   = "gpt-5.2-pro"
	ModelGPT52Codex = "gpt-5.2-codex"

	// GPT-5.1 series.
	ModelGPT51          = "gpt-5.1"
	ModelGPT51Codex     = "gpt-5.1-codex"
	ModelGPT51CodexMax  = "gpt-5.1-codex-max"
	ModelGPT51CodexMini = "gpt-5.1-codex-mini"

	// GPT-5 series.
	ModelGPT5      = "gpt-5"
	ModelGPT5Mini  = "gpt-5-mini"
	ModelGPT5Nano  = "gpt-5-nano"
	ModelGPT5Pro   = "gpt-5-pro"
	ModelGPT5Codex = "gpt-5-codex"

	// GPT-4o series.
	ModelGPT4o     = "gpt-4o"
	ModelGPT4oMini = "gpt-4o-mini"

	// GPT-4.1 series.
	ModelGPT41     = "gpt-4.1"
	ModelGPT41Mini = "gpt-4.1-mini"
	ModelGPT41Nano = "gpt-4.1-nano"

	// Legacy models.
	ModelGPT4Turbo = "gpt-4-turbo"
	ModelGPT4      = "gpt-4"
	ModelGPT35     = "gpt-3.5-turbo"

	// O-series reasoning models.
	ModelO4Mini = "o4-mini"
	ModelO3     = "o3"
	ModelO3Mini = "o3-mini"
	ModelO3Pro  = "o3-pro"
	ModelO1     = "o1"
	ModelO1Mini = "o1-mini"
	ModelO1Pro  = "o1-pro"
)

// ModelAliases maps short alias names to full model IDs.
var ModelAliases = map[string]string{
	// Default and tier aliases
	"openai":   ModelGPT54Mini,
	"default":  ModelGPT54Mini,
	"flagship": ModelGPT54,
	"mini":     ModelGPT54Mini,
	"nano":     ModelGPT54Nano,
	"pro":      ModelGPT54Pro,
	"fast":     ModelGPT54Nano,

	// Coding models
	"codex": ModelGPT53Codex,

	// Reasoning models
	"o4": ModelO4Mini,
	"o3": ModelO3,
	"o1": ModelO1,
}

// modelCategory identifies reasoning support level for a model.
type modelCategory int

const (
	categoryNonReasoning modelCategory = iota // gpt-4o, gpt-4, gpt-3.5, gpt-4.1
	categoryPreGPT51                          // gpt-5, gpt-5-mini, gpt-5-nano, o1, o3, o4-mini
	categoryGPT51                             // gpt-5.1
	categoryPro                               // gpt-5-pro, gpt-5.2-pro, o1-pro, o3-pro
	categoryCodex                             // codex models (support xhigh)
)

// modelInfo contains metadata and routing properties for a model.
type modelInfo struct {
	ID                    string
	Name                  string
	Category              modelCategory
	SupportsExtendedCache bool
	UseResponsesAPI       bool
}

// modelRegistry maps model IDs to provider-specific routing metadata.
var modelRegistry = map[string]modelInfo{
	// GPT-5.4 series (flagship, latest) — requires Responses API
	"gpt-5.4":      {ID: "gpt-5.4", Name: "GPT-5.4", Category: categoryPreGPT51, SupportsExtendedCache: true, UseResponsesAPI: true},
	"gpt-5.4-mini": {ID: "gpt-5.4-mini", Name: "GPT-5.4 Mini", Category: categoryPreGPT51, SupportsExtendedCache: true, UseResponsesAPI: true},
	"gpt-5.4-nano": {ID: "gpt-5.4-nano", Name: "GPT-5.4 Nano", Category: categoryPreGPT51, SupportsExtendedCache: true, UseResponsesAPI: true},
	"gpt-5.4-pro":  {ID: "gpt-5.4-pro", Name: "GPT-5.4 Pro", Category: categoryPro, UseResponsesAPI: true},

	// GPT-5.3 series
	"gpt-5.3-codex": {ID: "gpt-5.3-codex", Name: "GPT-5.3 Codex", Category: categoryCodex, SupportsExtendedCache: true},

	// GPT-5.2 series
	"gpt-5.2":       {ID: "gpt-5.2", Name: "GPT-5.2", Category: categoryPreGPT51, SupportsExtendedCache: true},
	"gpt-5.2-pro":   {ID: "gpt-5.2-pro", Name: "GPT-5.2 Pro", Category: categoryPro},
	"gpt-5.2-codex": {ID: "gpt-5.2-codex", Name: "GPT-5.2 Codex", Category: categoryCodex, SupportsExtendedCache: true},

	// GPT-5.1 series
	"gpt-5.1":            {ID: "gpt-5.1", Name: "GPT-5.1", Category: categoryGPT51, SupportsExtendedCache: true, UseResponsesAPI: true},
	"gpt-5.1-codex":      {ID: "gpt-5.1-codex", Name: "GPT-5.1 Codex", Category: categoryCodex, SupportsExtendedCache: true},
	"gpt-5.1-codex-max":  {ID: "gpt-5.1-codex-max", Name: "GPT-5.1 Codex Max", Category: categoryCodex, SupportsExtendedCache: true},
	"gpt-5.1-codex-mini": {ID: "gpt-5.1-codex-mini", Name: "GPT-5.1 Codex Mini", Category: categoryCodex, SupportsExtendedCache: true},

	// GPT-5 series
	"gpt-5":       {ID: "gpt-5", Name: "GPT-5", Category: categoryPreGPT51, SupportsExtendedCache: true},
	"gpt-5-mini":  {ID: "gpt-5-mini", Name: "GPT-5 Mini", Category: categoryPreGPT51, SupportsExtendedCache: true},
	"gpt-5-nano":  {ID: "gpt-5-nano", Name: "GPT-5 Nano", Category: categoryPreGPT51, SupportsExtendedCache: true},
	"gpt-5-pro":   {ID: "gpt-5-pro", Name: "GPT-5 Pro", Category: categoryPro},
	"gpt-5-codex": {ID: "gpt-5-codex", Name: "GPT-5 Codex", Category: categoryCodex, SupportsExtendedCache: true},

	// GPT-4o series
	"gpt-4o":      {ID: "gpt-4o", Name: "GPT-4o", Category: categoryNonReasoning},
	"gpt-4o-mini": {ID: "gpt-4o-mini", Name: "GPT-4o Mini", Category: categoryNonReasoning},

	// GPT-4.1 series (extended cache supported)
	"gpt-4.1":      {ID: "gpt-4.1", Name: "GPT-4.1", Category: categoryNonReasoning, SupportsExtendedCache: true},
	"gpt-4.1-mini": {ID: "gpt-4.1-mini", Name: "GPT-4.1 Mini", Category: categoryNonReasoning, SupportsExtendedCache: true},
	"gpt-4.1-nano": {ID: "gpt-4.1-nano", Name: "GPT-4.1 Nano", Category: categoryNonReasoning, SupportsExtendedCache: true},

	// GPT-4 series (legacy)
	"gpt-4-turbo": {ID: "gpt-4-turbo", Name: "GPT-4 Turbo", Category: categoryNonReasoning},
	"gpt-4":       {ID: "gpt-4", Name: "GPT-4", Category: categoryNonReasoning},

	// GPT-3.5 series (legacy)
	"gpt-3.5-turbo": {ID: "gpt-3.5-turbo", Name: "GPT-3.5 Turbo", Category: categoryNonReasoning},

	// o4 series
	"o4-mini": {ID: "o4-mini", Name: "o4 Mini", Category: categoryPreGPT51},

	// o3 series
	"o3":      {ID: "o3", Name: "o3", Category: categoryPreGPT51},
	"o3-mini": {ID: "o3-mini", Name: "o3 Mini", Category: categoryPreGPT51},
	"o3-pro":  {ID: "o3-pro", Name: "o3 Pro", Category: categoryPro},

	// o1 series (legacy reasoning)
	"o1":      {ID: "o1", Name: "o1", Category: categoryPreGPT51},
	"o1-mini": {ID: "o1-mini", Name: "o1 Mini", Category: categoryPreGPT51},
	"o1-pro":  {ID: "o1-pro", Name: "o1 Pro", Category: categoryPro},
}

// modelOrder defines the display order for Models().
var modelOrder = []string{
	"gpt-5.4", "gpt-5.4-mini", "gpt-5.4-nano", "gpt-5.4-pro",
	"gpt-5.3-codex",
	"gpt-5.2", "gpt-5.2-pro", "gpt-5.2-codex",
	"gpt-5.1", "gpt-5.1-codex", "gpt-5.1-codex-max", "gpt-5.1-codex-mini",
	"gpt-5", "gpt-5-mini", "gpt-5-nano", "gpt-5-pro", "gpt-5-codex",
	"gpt-4o", "gpt-4o-mini",
	"gpt-4.1", "gpt-4.1-mini", "gpt-4.1-nano",
	"gpt-4-turbo", "gpt-4",
	"gpt-3.5-turbo",
	"o4-mini", "o3", "o3-mini", "o3-pro", "o1", "o1-mini", "o1-pro",
}

// ResolveModel resolves a model alias or ID to a canonical model ID.
func ResolveModel(model string) string {
	if model == "" {
		return DefaultModel
	}

	// Check direct alias match (case-insensitive)
	lower := strings.ToLower(model)
	if resolved, ok := ModelAliases[lower]; ok {
		return resolved
	}

	// Check if it's already a valid model ID
	if _, ok := modelRegistry[model]; ok {
		return model
	}

	// Return as-is for unknown models (may be valid but not in registry)
	return model
}

// UseResponsesAPI reports whether the given model requires the OpenAI
// Responses API (/v1/responses) instead of Chat Completions.
func UseResponsesAPI(model string) bool {
	info, ok := modelRegistry[model]
	if !ok {
		return false
	}
	return info.Category == categoryCodex || info.UseResponsesAPI
}

// SupportsPromptCaching reports whether the model supports extended prompt caching.
func SupportsPromptCaching(model string) bool {
	info, ok := modelRegistry[model]
	if !ok {
		return false
	}
	return info.SupportsExtendedCache
}

// Effort levels for OpenAI reasoning models.
const (
	EffortLow    = "low"
	EffortMedium = "medium"
	EffortHigh   = "high"
	EffortMax    = "max"  // Maps to xhigh for codex models
	EffortNone   = "none" // Disable reasoning (GPT-5.1 only)
)

// MapEffortAndThinking maps the user-requested effort and thinking mode to a
// valid OpenAI reasoning_effort API value.
// Returns empty string if the parameter should be omitted.
func MapEffortAndThinking(model string, effort string, thinkingOff bool) string {
	info, ok := modelRegistry[model]
	if !ok {
		// Unknown model - treat as non-reasoning, omit reasoning_effort
		return ""
	}

	// Non-reasoning models never send reasoning_effort
	if info.Category == categoryNonReasoning {
		return ""
	}

	// Thinking explicitly off - disable where possible
	if thinkingOff {
		if info.Category == categoryGPT51 {
			return "none"
		}
		// pre-GPT-5.1, Pro, Codex: can't reliably disable reasoning
		return ""
	}

	// Clamp EffortMax - xhigh for Codex, else High
	if effort == EffortMax {
		if info.Category == categoryCodex {
			return "xhigh"
		}
		effort = EffortHigh
	}

	// No effort specified - omit, let API use its default
	if effort == "" {
		return ""
	}

	return effort
}

// getModelInfo returns the model info for the given model ID.
func getModelInfo(model string) (modelInfo, error) {
	info, ok := modelRegistry[model]
	if !ok {
		return modelInfo{}, fmt.Errorf("unknown model: %s", model)
	}
	return info, nil
}

// Model represents an OpenAI model.
type Model struct {
	ID       string
	Name     string
	Provider string
	Aliases  []string
}

// Models is a list of available models.
type Models []Model

// Resolve resolves a model alias or ID to a Model.
// It performs case-insensitive matching against model IDs and aliases.
func (m Models) Resolve(alias string) (Model, error) {
	if alias == "" {
		return m[0], nil // Return default (first model)
	}

	lower := strings.ToLower(alias)

	// Check direct alias match
	if resolved, ok := ModelAliases[lower]; ok {
		alias = resolved
	}

	// Find in models list
	for _, model := range m {
		if strings.EqualFold(model.ID, alias) {
			return model, nil
		}
		for _, a := range model.Aliases {
			if strings.EqualFold(a, alias) {
				return model, nil
			}
		}
	}

	// Return as-is for unknown models (may be valid but not in registry)
	return Model{ID: alias, Name: alias, Provider: ProviderName}, nil
}

// LoadModels returns the list of available OpenAI models.
func LoadModels() Models {
	models := make(Models, 0, len(modelOrder))
	for _, id := range modelOrder {
		info, ok := modelRegistry[id]
		if !ok {
			continue
		}
		models = append(models, Model{
			ID:       info.ID,
			Name:     info.Name,
			Provider: ProviderName,
			Aliases:  aliasesForModel(info.ID),
		})
	}
	return models
}

func aliasesForModel(modelID string) []string {
	var aliases []string
	for alias, target := range ModelAliases {
		if target == modelID {
			aliases = append(aliases, alias)
		}
	}
	return aliases
}
