package minimax

import (
	"fmt"
	"strings"
)

// Model ID constants for programmatic use.
const (
	ModelM27          = "MiniMax-M2.7"
	ModelM27Highspeed = "MiniMax-M2.7-highspeed"
	ModelM25          = "MiniMax-M2.5"
	ModelM25Highspeed = "MiniMax-M2.5-highspeed"
	ModelM21          = "MiniMax-M2.1"
	ModelM21Highspeed = "MiniMax-M2.1-highspeed"
	ModelM2           = "MiniMax-M2"
)

// Alias constants for short names.
const (
	AliasM27  = "m2.7"
	AliasM25  = "m2.5"
	AliasM21  = "m2.1"
	AliasM2   = "m2"
	AliasFast = "highspeed"
)

// ProviderAliases maps short names to wire model IDs for this provider.
var ProviderAliases = map[string]string{
	AliasM27:  ModelM27,
	AliasM25:  ModelM25,
	AliasM21:  ModelM21,
	AliasM2:   ModelM2,
	AliasFast: ModelM27Highspeed,
}

// Model represents a MiniMax model.
type Model struct {
	ID          string
	DisplayName string
	Aliases     []string
}

// Models is a list of available models.
type Models []Model

// Resolve resolves a model alias or ID to a Model.
// Returns an error if the alias is not found.
func (m Models) Resolve(alias string) (Model, error) {
	lower := strings.ToLower(alias)

	// Check direct ID match first
	for _, model := range m {
		if strings.ToLower(model.ID) == lower {
			return model, nil
		}
	}

	// Check aliases
	for _, model := range m {
		for _, a := range model.Aliases {
			if strings.ToLower(a) == lower {
				return model, nil
			}
		}
	}

	return Model{}, fmt.Errorf("minimax: unknown model %q", alias)
}

// LoadModels returns the list of available MiniMax models.
func LoadModels() Models {
	return Models{
		{
			ID:          ModelM27,
			DisplayName: "MiniMax M2.7",
			Aliases:     []string{"minimax", "default", "fast", "m2.7", "2.7"},
		},
		{
			ID:          ModelM27Highspeed,
			DisplayName: "MiniMax M2.7 Highspeed",
			Aliases:     []string{"m2.7-highspeed", "2.7-highspeed"},
		},
		{
			ID:          ModelM25,
			DisplayName: "MiniMax M2.5",
			Aliases:     []string{"m2.5", "2.5"},
		},
		{
			ID:          ModelM25Highspeed,
			DisplayName: "MiniMax M2.5 Highspeed",
			Aliases:     []string{"m2.5-highspeed", "2.5-highspeed"},
		},
		{
			ID:          ModelM21,
			DisplayName: "MiniMax M2.1",
			Aliases:     []string{"m2.1", "2.1"},
		},
		{
			ID:          ModelM21Highspeed,
			DisplayName: "MiniMax M2.1 Highspeed",
			Aliases:     []string{"m2.1-highspeed", "2.1-highspeed"},
		},
		{
			ID:          ModelM2,
			DisplayName: "MiniMax M2",
			Aliases:     []string{"m2", "2"},
		},
	}
}
