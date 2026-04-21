package openai

import (
	"context"
	"net/http"

	responsesapi "github.com/codewandler/agentapis/api/responses"
	"github.com/codewandler/agentapis/api/unified"
	"github.com/codewandler/agentapis/client"
	"github.com/codewandler/agentapis/conversation"
)

// Provider implements the OpenAI Responses API provider.
type Provider struct {
	cfg             Config
	auth            *Auth
	models          Models
	responsesClient *client.ResponsesClient
}

// New creates a new OpenAI provider.
// By default, it uses the OPENAI_API_KEY or OPENAI_KEY environment variable for authentication.
func New(opts ...Option) (*Provider, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	// Resolve auth
	var auth *Auth
	if o.apiKey != "" {
		auth = NewAPIKeyAuth(o.apiKey)
	} else {
		var err error
		auth, err = NewEnvAuth()
		if err != nil {
			return nil, err
		}
	}

	// Build config
	cfg := Config{
		BaseURL: o.baseURL,
		APIKey:  auth.APIKey(),
		Model:   o.model,
	}

	// Build protocol options
	protocolOpts := []responsesapi.Option{
		responsesapi.WithBaseURL(cfg.BaseURL),
		responsesapi.WithHeaderFunc(func(ctx context.Context, req *responsesapi.Request) (http.Header, error) {
			h := make(http.Header)
			if err := auth.SetHeaders(ctx, h); err != nil {
				return nil, err
			}
			return h, nil
		}),
		responsesapi.WithHTTPRequestMutator(func(ctx context.Context, httpReq *http.Request, req *responsesapi.Request) error {
			httpReq.Header.Set("Content-Type", "application/json")
			return nil
		}),
		responsesapi.WithRequestTransform(func(ctx context.Context, req *responsesapi.Request) error {
			// Set reasoning summary to auto if reasoning is configured
			if req.Reasoning != nil && req.Reasoning.Summary == nil {
				s := responsesapi.ReasoningSummaryAuto
				req.Reasoning.Summary = &s
			}
			return nil
		}),
	}

	if o.httpClient != nil {
		protocolOpts = append(protocolOpts, responsesapi.WithHTTPClient(o.httpClient))
	}

	// Create the low-level responses API client
	protocol := responsesapi.NewClient(protocolOpts...)

	// Client-level transforms operate on unified.Request and can access Effort/Thinking
	clientOpts := []client.Option{
		client.WithRequestTransform(func(ctx context.Context, req *unified.Request) error {
			// Map effort based on model and thinking mode
			thinkingOff := req.Thinking == unified.ThinkingModeOff
			effort := MapEffortAndThinking(req.Model, string(req.Effort), thinkingOff)
			if effort != "" {
				req.Effort = unified.Effort(effort)
			}
			return nil
		}),
	}

	// Create the high-level unified client
	responsesClient := client.NewResponsesClient(protocol, clientOpts...)

	p := &Provider{
		cfg:             cfg,
		auth:            auth,
		models:          LoadModels(),
		responsesClient: responsesClient,
	}

	return p, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return ProviderName
}

// Models returns the available models.
func (p *Provider) GetModels() Models {
	return p.models
}

// Capabilities returns the provider capabilities.
func (p *Provider) Capabilities() Capabilities {
	return Capabilities{
		SupportsResponsesPreviousResponseID: true, // OpenAI supports previous_response_id
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
	// Resolve model alias
	if req.Model != "" {
		model, _ := p.models.Resolve(req.Model)
		req.Model = model.ID
	} else {
		req.Model = p.cfg.Model
	}

	return p.responsesClient.Stream(ctx, req)
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
