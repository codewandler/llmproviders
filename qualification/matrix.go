package qualification

type Status string

const (
	StatusQualified    Status = "qualified"
	StatusExperimental Status = "experimental"
	StatusPartial      Status = "partial"
	StatusUnsupported  Status = "unsupported"
	StatusRegressed    Status = "regressed"
)

type Dimensions struct {
	Tools        bool `json:"tools"`
	Caching      bool `json:"caching"`
	Pricing      bool `json:"pricing"`
	Thinking     bool `json:"thinking"`
	Conversation bool `json:"conversation"`
	CodingLoop   bool `json:"coding_loop"`
	Usage        bool `json:"usage"`
}

type Entry struct {
	Provider     string     `json:"provider"`
	Service      string     `json:"service"`
	Model        string     `json:"model"`
	Status       Status     `json:"status"`
	Dimensions   Dimensions `json:"dimensions"`
	Notes        []string   `json:"notes,omitempty"`
}

func DeriveStatus(d Dimensions) Status {
	if d.Tools && d.Caching && d.Pricing && d.Thinking && d.Conversation && d.CodingLoop && d.Usage {
		return StatusQualified
	}
	if d.Tools && d.Pricing && d.Conversation && d.CodingLoop && d.Usage {
		return StatusExperimental
	}
	if d.Tools || d.Conversation || d.Usage {
		return StatusPartial
	}
	return StatusUnsupported
}
