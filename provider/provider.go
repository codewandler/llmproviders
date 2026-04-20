package provider

import (
	"context"

	"github.com/codewandler/agentapis/api/unified"
	"github.com/codewandler/agentapis/client"
)

// Streamer is the minimal provider-backed unified streaming surface expected by
// downstream consumers such as miniagent.
type Streamer interface {
	Stream(ctx context.Context, req unified.Request) (<-chan client.StreamResult, error)
}

// Provider extends Streamer with provider metadata useful for selection and
// capability-aware behavior.
type Provider interface {
	Streamer
	Capabilities() Capabilities
	Name() string
}
