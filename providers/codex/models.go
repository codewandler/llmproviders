package codex

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/codewandler/llmproviders/modelcatalog"
)

//go:embed models.json
var embeddedModelsJSON []byte

type embeddedModel struct {
	Slug                      string   `json:"slug"`
	DisplayName               string   `json:"display_name"`
	Description               string   `json:"description"`
	SupportedReasoningLevels  []string `json:"supported_reasoning_levels"`
	Visibility                string   `json:"visibility"`
	SupportedInAPI            bool     `json:"supported_in_api"`
	AvailableInPlans          []string `json:"available_in_plans"`
	Priority                  int      `json:"priority"`
	SupportsReasoningSummary  bool     `json:"supports_reasoning_summaries"`
	SupportsParallelToolCalls bool     `json:"supports_parallel_tool_calls"`
	SupportsPromptCaching     bool     `json:"supports_prompt_caching"`
}

type embeddedModelsPayload struct {
	Models []embeddedModel `json:"models"`
}

var embeddedModels = mustLoadEmbeddedModels()

func EmbeddedModels() []embeddedModel {
	out := make([]embeddedModel, len(embeddedModels.Models))
	copy(out, embeddedModels.Models)
	return out
}

func DefaultModelID() string {
	if model, ok := firstPresent(embeddedModels.Models, "gpt-5.4"); ok {
		return model.Slug
	}
	if model, ok := firstVisibleByPriority(embeddedModels.Models); ok {
		return model.Slug
	}
	if len(embeddedModels.Models) == 0 {
		return ""
	}
	return embeddedModels.Models[0].Slug
}

func FastModelID() string {
	if model, ok := firstPresent(embeddedModels.Models, "gpt-5.4-mini"); ok {
		return model.Slug
	}
	return DefaultModelID()
}

func PowerfulModelID() string {
	if model, ok := firstPresent(embeddedModels.Models, "gpt-5.4"); ok {
		return model.Slug
	}
	return DefaultModelID()
}

func BuiltinAliasModels() (fast, normal, powerful string) {
	return FastModelID(), DefaultModelID(), PowerfulModelID()
}

func ModelAliases() map[string]string {
	aliases := map[string]string{}
	if model := DefaultModelID(); model != "" {
		aliases["codex"] = model
	}
	if model := FastModelID(); model != "" {
		aliases["mini"] = model
	}
	return aliases
}

func CatalogModels() []modelcatalog.Model {
	models := EmbeddedModels()
	sort.SliceStable(models, func(i, j int) bool {
		if models[i].Priority != models[j].Priority {
			return models[i].Priority < models[j].Priority
		}
		return models[i].Slug < models[j].Slug
	})
	out := make([]modelcatalog.Model, 0, len(models))
	for _, model := range models {
		if model.Visibility != "list" || !model.SupportedInAPI {
			continue
		}
		caps := []string{"responses", "tools", "thinking", "usage"}
		if model.SupportsReasoningSummary {
			caps = append(caps, "reasoning_summary")
		}
		if model.SupportsParallelToolCalls {
			caps = append(caps, "parallel_tools")
		}
		if model.SupportsPromptCaching {
			caps = append(caps, "prompt_caching")
		}
		out = append(out, modelcatalog.Model{
			ID:           model.Slug,
			Provider:     "codex",
			Service:      "responses",
			DisplayName:  firstNonEmpty(model.DisplayName, model.Slug),
			Aliases:      aliasesForModel(model.Slug),
			Capabilities: caps,
		})
	}
	return out
}

func aliasesForModel(id string) []string {
	var aliases []string
	for alias, target := range ModelAliases() {
		if target == id {
			aliases = append(aliases, alias)
		}
	}
	sort.Strings(aliases)
	return aliases
}

func mustLoadEmbeddedModels() embeddedModelsPayload {
	var payload embeddedModelsPayload
	if err := json.Unmarshal(embeddedModelsJSON, &payload); err != nil {
		panic(fmt.Sprintf("codex: parse embedded models.json: %v", err))
	}
	return payload
}

func firstPresent(models []embeddedModel, preferred ...string) (embeddedModel, bool) {
	for _, slug := range preferred {
		for _, model := range models {
			if model.Slug == slug {
				return model, true
			}
		}
	}
	return embeddedModel{}, false
}

func firstVisibleByPriority(models []embeddedModel) (embeddedModel, bool) {
	visible := make([]embeddedModel, 0, len(models))
	for _, model := range models {
		if model.Visibility == "list" {
			visible = append(visible, model)
		}
	}
	if len(visible) == 0 {
		return embeddedModel{}, false
	}
	sort.SliceStable(visible, func(i, j int) bool {
		if visible[i].Priority != visible[j].Priority {
			return visible[i].Priority < visible[j].Priority
		}
		return visible[i].Slug < visible[j].Slug
	})
	return visible[0], true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
