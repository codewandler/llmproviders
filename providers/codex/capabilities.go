package codex

import lpprovider "github.com/codewandler/llmproviders/provider"

func staticCapabilities() lpprovider.Capabilities {
	return lpprovider.Capabilities{
		SupportsResponsesPreviousResponseID: true,
		SupportsTools:                       true,
		SupportsStrictTools:                 true,
		SupportsJSONSchemaOutput:            false,
		SupportsThinking:                    true,
		SupportsEffort:                      true,
		SupportsPromptCaching:               false,
	}
}
