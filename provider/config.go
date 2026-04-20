package provider

// CommonConfig contains shared provider bootstrap fields. Individual provider
// packages may embed or ignore this depending on their needs.
type CommonConfig struct {
	BaseURL string
	APIKey  string
}
