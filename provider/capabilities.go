package provider

// Capabilities describes provider features that may influence request shaping
// or conversation strategy selection.
type Capabilities struct {
	SupportsResponsesPreviousResponseID bool
	SupportsTools                       bool
	SupportsStrictTools                 bool
	SupportsJSONSchemaOutput            bool
	SupportsThinking                    bool
	SupportsEffort                      bool
	SupportsPromptCaching               bool
}
