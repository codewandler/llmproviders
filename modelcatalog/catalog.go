package modelcatalog

// Model describes provider-facing model metadata.
type Model struct {
	ID           string
	Provider     string
	Service      string
	DisplayName  string
	Aliases      []string
	Capabilities []string
}
