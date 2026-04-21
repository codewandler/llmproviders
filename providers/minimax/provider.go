package minimax

import (
	"context"
	"errors"
	"net/http"

	messagesapi "github.com/codewandler/agentapis/api/messages"
	"github.com/codewandler/agentapis/api/unified"
	"github.com/codewandler/agentapis/client"
	"github.com/codewandler/agentapis/conversation"
)

// Provider implements the MiniMax Messages API provider.
// MiniMax provides an Anthropic Messages API-compatible endpoint.
type Provider struct {
	cfg            Config
	auth           Auth
	models         Models
	messagesClient *client.MessagesClient
}

// New creates a new MiniMax provider.
// By default, it uses the MINIMAX_API_KEY environment variable for authentication.
func New(opts ...Option) (*Provider, error) {
	o := applyOptions(opts)

	// Resolve auth
	auth := o.auth
	if auth == nil {
		var err error
		auth, err = NewAPIKeyAuthFromEnv()
		if err != nil {
			return nil, err
		}
	}

	// Build config
	cfg := Config{
		BaseURL:    o.baseURL,
		Model:      o.model,
		HTTPClient: o.httpClient,
	}

	// Create the low-level messages API client
	protocol := messagesapi.NewClient(
		messagesapi.WithBaseURL(cfg.baseURL()),
		messagesapi.WithHTTPClient(cfg.httpClient()),
		messagesapi.WithHeaderFunc(func(ctx context.Context, req *messagesapi.Request) (http.Header, error) {
			h := make(http.Header)
			if err := auth.ApplyAuth(ctx, h); err != nil {
				return nil, err
			}
			return h, nil
		}),
		messagesapi.WithHTTPRequestMutator(func(ctx context.Context, httpReq *http.Request, req *messagesapi.Request) error {
			// Set required Anthropic-compatible headers
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("Anthropic-Version", AnthropicVersion)
			httpReq.Header.Set("Anthropic-Beta", BetaInterleavedThinking)
			return nil
		}),
		messagesapi.WithRequestTransform(func(ctx context.Context, req *messagesapi.Request) error {
			// MiniMax requires a model to be specified
			if req.Model == "" {
				return errors.New("minimax: model is required")
			}
			return nil
		}),
	)

	// Create the high-level unified client
	messagesClient := client.NewMessagesClient(protocol)

	p := &Provider{
		cfg:            cfg,
		auth:           auth,
		models:         LoadModels(),
		messagesClient: messagesClient,
	}

	return p, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return ProviderName
}

// Models returns the available models.
func (p *Provider) Models() Models {
	return p.models
}

// Capabilities returns the provider capabilities.
func (p *Provider) Capabilities() Capabilities {
	return Capabilities{
		SupportsResponsesPreviousResponseID: false, // MiniMax uses replay strategy
	}
}

// Capabilities holds provider capability flags.
type Capabilities struct {
	// SupportsResponsesPreviousResponseID indicates if the provider supports
	// the Responses API previous_response_id for conversation continuity.
	// MiniMax does not; it uses message replay.
	SupportsResponsesPreviousResponseID bool
}

// Stream implements conversation.Streamer.
// It sends a unified request and returns a channel of stream results.
func (p *Provider) Stream(ctx context.Context, req unified.Request) (<-chan client.StreamResult, error) {
	// Resolve model alias
	if req.Model != "" {
		if model, err := p.models.Resolve(req.Model); err == nil {
			req.Model = model.ID
		}
	} else {
		req.Model = p.cfg.model()
	}

	return p.messagesClient.Stream(ctx, req)
}

// Session creates a new conversation session backed by this provider.
func (p *Provider) CreateSession(opts ...conversation.Option) *conversation.Session {
	caps := p.Capabilities()
	defaults := []conversation.Option{
		conversation.WithModel(p.cfg.model()),
		conversation.WithCapabilities(conversation.Capabilities{
			SupportsResponsesPreviousResponseID: caps.SupportsResponsesPreviousResponseID,
		}),
	}
	return conversation.New(p, append(defaults, opts...)...)
}

// Ensure Provider implements conversation.Streamer.
var _ conversation.Streamer = (*Provider)(nil)
