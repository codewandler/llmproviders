package ollama

import (
	"context"
	"net/http"

	ollamaapi "github.com/codewandler/agentapis/api/ollama"
	"github.com/codewandler/agentapis/api/unified"
	"github.com/codewandler/agentapis/client"
	"github.com/codewandler/agentapis/conversation"
)

// Provider implements the Ollama local LLM provider.
// It wraps the native Ollama API via agentapis.
type Provider struct {
	cfg          Config
	ollamaClient *client.OllamaClient
	httpClient   *http.Client
}

// New creates a new Ollama provider.
// No authentication is required as Ollama runs locally.
func New(opts ...Option) (*Provider, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	// Build config
	cfg := Config{
		BaseURL: o.baseURL,
		Model:   o.model,
	}

	httpClient := o.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	// Build protocol options
	protocolOpts := []ollamaapi.Option{
		ollamaapi.WithBaseURL(cfg.BaseURL),
	}

	if o.httpClient != nil {
		protocolOpts = append(protocolOpts, ollamaapi.WithHTTPClient(o.httpClient))
	}

	// Create the low-level Ollama API client
	protocol := ollamaapi.NewClient(protocolOpts...)

	// Create the high-level unified client
	ollamaClient := client.NewOllamaClient(protocol)

	return &Provider{
		cfg:          cfg,
		ollamaClient: ollamaClient,
		httpClient:   httpClient,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return ProviderName
}

// Capabilities returns the provider capabilities.
func (p *Provider) Capabilities() Capabilities {
	return Capabilities{
		// Ollama does not support previous_response_id
		SupportsResponsesPreviousResponseID: false,
	}
}

// Capabilities holds provider capability flags.
type Capabilities struct {
	// SupportsResponsesPreviousResponseID indicates if the provider supports
	// the Responses API previous_response_id for conversation continuity.
	SupportsResponsesPreviousResponseID bool
}

// Stream implements conversation.Streamer.
// It sends a unified request and returns a channel of stream results.
func (p *Provider) Stream(ctx context.Context, req unified.Request) (<-chan client.StreamResult, error) {
	// Use default model if not specified
	if req.Model == "" {
		req.Model = p.cfg.Model
	}

	return p.ollamaClient.Stream(ctx, req)
}

// Session creates a new conversation session backed by this provider.
func (p *Provider) CreateSession(opts ...conversation.Option) *conversation.Session {
	caps := p.Capabilities()
	defaults := []conversation.Option{
		conversation.WithModel(p.cfg.Model),
		conversation.WithCapabilities(conversation.Capabilities{
			SupportsResponsesPreviousResponseID: caps.SupportsResponsesPreviousResponseID,
		}),
	}
	return conversation.New(p, append(defaults, opts...)...)
}

// Ensure Provider implements conversation.Streamer.
var _ conversation.Streamer = (*Provider)(nil)
