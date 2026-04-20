package dockermr

import (
	"context"
	"net/http"

	completionsapi "github.com/codewandler/agentapis/api/completions"
	"github.com/codewandler/agentapis/api/unified"
	"github.com/codewandler/agentapis/client"
	"github.com/codewandler/agentapis/conversation"
)

// Provider implements the Docker Model Runner local LLM provider.
// It wraps the OpenAI Chat Completions API exposed by Docker Model Runner.
type Provider struct {
	cfg               Config
	completionsClient *client.CompletionsClient
	httpClient        *http.Client
}

// New creates a new Docker Model Runner provider.
// No authentication is required as Docker Model Runner runs locally.
func New(opts ...Option) (*Provider, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	// Build config
	cfg := Config{
		BaseURL: o.baseURL,
		Engine:  o.engine,
		Model:   o.model,
	}

	httpClient := o.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	// Build the engine-specific URL
	engineURL := EngineURL(cfg.BaseURL, cfg.Engine)

	// Build protocol options
	protocolOpts := []completionsapi.Option{
		completionsapi.WithBaseURL(engineURL),
	}

	if o.httpClient != nil {
		protocolOpts = append(protocolOpts, completionsapi.WithHTTPClient(o.httpClient))
	}

	// Create the low-level Completions API client
	protocol := completionsapi.NewClient(protocolOpts...)

	// Create the high-level unified client
	completionsClient := client.NewCompletionsClient(protocol)

	return &Provider{
		cfg:               cfg,
		completionsClient: completionsClient,
		httpClient:        httpClient,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return ProviderName
}

// Capabilities returns the provider capabilities.
func (p *Provider) Capabilities() Capabilities {
	return Capabilities{
		// Docker Model Runner does not support previous_response_id
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

	return p.completionsClient.Stream(ctx, req)
}

// Session creates a new conversation session backed by this provider.
func (p *Provider) Session(opts ...conversation.Option) *conversation.Session {
	caps := p.Capabilities()
	defaults := []conversation.Option{
		conversation.WithModel(p.cfg.Model),
		conversation.WithCapabilities(conversation.Capabilities{
			SupportsResponsesPreviousResponseID: caps.SupportsResponsesPreviousResponseID,
		}),
	}
	return conversation.New(p, append(defaults, opts...)...)
}

// WithEngine returns a new Provider configured with the specified engine.
// This is a convenience method for switching engines on an existing provider.
func (p *Provider) WithEngine(engine string) (*Provider, error) {
	return New(
		WithBaseURL(p.cfg.BaseURL),
		WithEngine(engine),
		WithModel(p.cfg.Model),
		WithHTTPClient(p.httpClient),
	)
}

// Ensure Provider implements conversation.Streamer.
var _ conversation.Streamer = (*Provider)(nil)
