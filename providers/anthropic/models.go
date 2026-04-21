package anthropic

import (
	"fmt"
	"sort"
	"strings"

	"github.com/codewandler/modeldb"
)

// Wire model ID constants for programmatic use.
const (
	// Claude 4.6 (current).
	ModelOpus   = "claude-opus-4-6"
	ModelSonnet = "claude-sonnet-4-6"

	// Claude 4.5 (Haiku latest).
	ModelHaiku = "claude-haiku-4-5-20251001"

	// Claude 4.5 others
	ModelSonnet45 = "claude-sonnet-4-5-20250929"
	ModelOpus45   = "claude-opus-4-5-20251101"
)

// Alias constants for short names.
const (
	AliasSonnet = "sonnet"
	AliasOpus   = "opus"
	AliasHaiku  = "haiku"
)

// ProviderAliases maps short names to wire model IDs for this provider.
var ProviderAliases = map[string]string{
	AliasSonnet: ModelSonnet,
	AliasOpus:   ModelOpus,
	AliasHaiku:  ModelHaiku,
}

// ModelAliases is kept for backward compatibility.
var ModelAliases = ProviderAliases

// Model represents a model available from the provider.
type Model struct {
	ID       string
	Name     string
	Provider string
	Aliases  []string
}

// Models is a list of models with resolution capability.
type Models []Model

// fallbackModels provides a static list when modeldb is unavailable.
var fallbackModels = Models{
	{ID: ModelSonnet, Name: "Claude Sonnet 4.6", Provider: ProviderName, Aliases: []string{AliasSonnet}},
	{ID: ModelOpus, Name: "Claude Opus 4.6", Provider: ProviderName, Aliases: []string{AliasOpus}},
	{ID: ModelHaiku, Name: "Claude Haiku 4.5", Provider: ProviderName, Aliases: []string{AliasHaiku}},
	{ID: ModelSonnet45, Name: "Claude Sonnet 4.5", Provider: ProviderName},
	{ID: "claude-opus-4-5", Name: "Claude Opus 4.5", Provider: ProviderName},
	{ID: ModelOpus45, Name: "Claude Opus 4.5", Provider: ProviderName},
	{ID: "claude-haiku-4-5", Name: "Claude Haiku 4.5", Provider: ProviderName},
	{ID: "claude-opus-4-1", Name: "Claude Opus 4.1", Provider: ProviderName},
	{ID: "claude-opus-4-1-20250805", Name: "Claude Opus 4.1", Provider: ProviderName},
	{ID: "claude-opus-4", Name: "Claude Opus 4.0", Provider: ProviderName},
	{ID: "claude-opus-4-20250514", Name: "Claude Opus 4.0", Provider: ProviderName},
	{ID: "claude-sonnet-4", Name: "Claude Sonnet 4.0", Provider: ProviderName},
	{ID: "claude-sonnet-4-20250514", Name: "Claude Sonnet 4.0", Provider: ProviderName},
}

var fallbackModelsByID = func() map[string]Model {
	models := make(map[string]Model, len(fallbackModels))
	for _, model := range fallbackModels {
		models[model.ID] = model
	}
	return models
}()

// loadedModels caches the loaded models.
var loadedModels Models

func init() {
	loadedModels = loadAnthropicModels()
}

// LoadModels returns the available Anthropic models.
func LoadModels() Models {
	return loadedModels
}

// loadAnthropicModels loads models from modeldb with fallback to static list.
func loadAnthropicModels() Models {
	catalog, err := modeldb.LoadBuiltIn()
	if err != nil {
		return fallbackModels
	}

	// Get offerings for anthropic service
	offerings := catalog.OfferingsByService(ProviderName)
	if len(offerings) == 0 {
		return fallbackModels
	}

	// Build models from offerings
	remaining := make(map[string]Model, len(offerings))
	for _, offering := range offerings {
		model := Model{
			ID:       offering.WireModelID,
			Provider: ProviderName,
			Aliases:  offering.Aliases,
		}

		// Get display name from model record if available
		if record, ok := catalog.Models[offering.ModelKey]; ok {
			model.Name = record.Name
		}
		if model.Name == "" || model.Name == model.ID {
			model.Name = offering.WireModelID
		}

		// Merge with fallback aliases if available
		if fallback, ok := fallbackModelsByID[model.ID]; ok {
			if model.Name == "" || model.Name == model.ID {
				model.Name = fallback.Name
			}
			model.Aliases = mergeAliases(fallback.Aliases, model.Aliases)
		}

		remaining[model.ID] = model
	}

	// Preserve ordering from fallback list, then add any new models
	out := make(Models, 0, len(remaining))
	for _, fallback := range fallbackModels {
		if model, ok := remaining[fallback.ID]; ok {
			out = append(out, model)
			delete(remaining, fallback.ID)
			continue
		}
		out = append(out, fallback)
	}

	// Add remaining models not in fallback list
	ids := make([]string, 0, len(remaining))
	for id := range remaining {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		out = append(out, remaining[id])
	}

	return out
}

// Resolve resolves a model alias or ID to a Model.
// It checks aliases first, then looks for exact ID match.
func (m Models) Resolve(alias string) (Model, error) {
	alias = strings.ToLower(alias)

	// Check direct alias mapping
	if id, ok := ModelAliases[alias]; ok {
		alias = id
	}

	// Search models
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

	return Model{}, fmt.Errorf("anthropic: unknown model %q", alias)
}

// mergeAliases combines two alias slices, deduplicating.
func mergeAliases(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, values := range [][]string{a, b} {
		for _, value := range values {
			if value == "" {
				continue
			}
			lower := strings.ToLower(value)
			if _, ok := seen[lower]; ok {
				continue
			}
			seen[lower] = struct{}{}
			out = append(out, value)
		}
	}
	return out
}
