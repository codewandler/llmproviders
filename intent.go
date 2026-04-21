package llmproviders

// Intent alias constants for model selection policies.
// These provide semantic shortcuts for common use cases.
const (
	// IntentFast selects the fastest available model (e.g., Haiku, GPT-4o-mini).
	IntentFast = "fast"

	// IntentDefault selects a balanced model for general use (e.g., Sonnet, GPT-4o).
	IntentDefault = "default"

	// IntentPowerful selects the most capable model (e.g., Opus, o3).
	IntentPowerful = "powerful"
)

// AllIntents returns all built-in intent alias names.
func AllIntents() []string {
	return []string{IntentFast, IntentDefault, IntentPowerful}
}
