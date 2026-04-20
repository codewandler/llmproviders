package modelcatalog

// Offering can later capture provider/service/model-specific offering details.
type Offering struct {
	Provider string
	Service  string
	ModelID  string
}
