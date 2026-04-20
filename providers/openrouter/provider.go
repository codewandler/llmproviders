package openrouter

import (
	"context"
	"net/http"
	"strings"

	messagesapi "github.com/codewandler/agentapis/api/messages"
	responsesapi "github.com/codewandler/agentapis/api/responses"
	"github.com/codewandler/agentapis/api/unified"
	"github.com/codewandler/agentapis/client"
	"github.com/codewandler/agentapis/conversation"
	"github.com/codewandler/llmproviders/providers/anthropic"
)

// Provider implements the OpenRouter provider.
// It dynamically routes to either the Responses API or Messages API based on the model.
type Provider struct {
	cfg             Config
	auth            *Auth
	models          Models
	responsesClient *client.ResponsesClient
	messagesClient  *client.MessagesClient
}

// New creates a new OpenRouter provider.
// By default, it uses the OPENROUTER_API_KEY environment variable for authentication.
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

	// Build Responses API client (for non-Anthropic models)
	responsesProtocolOpts := []responsesapi.Option{
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
	}

	if o.httpClient != nil {
		responsesProtocolOpts = append(responsesProtocolOpts, responsesapi.WithHTTPClient(o.httpClient))
	}

	responsesProtocol := responsesapi.NewClient(responsesProtocolOpts...)
	responsesClient := client.NewResponsesClient(responsesProtocol,
		// Filter thinking parts from responses API requests
		client.WithRequestTransform(func(ctx context.Context, req *unified.Request) error {
			filterThinkingParts(req)
			return nil
		}),
	)

	// Build Messages API client (for Anthropic models)
	messagesProtocolOpts := []messagesapi.Option{
		messagesapi.WithBaseURL(cfg.BaseURL),
		messagesapi.WithHeaderFunc(func(ctx context.Context, req *messagesapi.Request) (http.Header, error) {
			h := make(http.Header)
			if err := auth.SetHeaders(ctx, h); err != nil {
				return nil, err
			}
			return h, nil
		}),
		messagesapi.WithHTTPRequestMutator(func(ctx context.Context, httpReq *http.Request, req *messagesapi.Request) error {
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("Anthropic-Version", anthropic.AnthropicVersion)
			httpReq.Header.Set("Anthropic-Beta", anthropic.BetaInterleavedThinking)
			return nil
		}),
		messagesapi.WithRequestTransform(func(ctx context.Context, req *messagesapi.Request) error {
			// Strip anthropic/ prefix from model
			if strings.HasPrefix(req.Model, AnthropicModelPrefix) {
				req.Model = strings.TrimPrefix(req.Model, AnthropicModelPrefix)
			}
			// Apply thinking temperature coercion for Anthropic models
			anthropic.CoerceThinkingTemperature(req)
			return nil
		}),
	}

	if o.httpClient != nil {
		messagesProtocolOpts = append(messagesProtocolOpts, messagesapi.WithHTTPClient(o.httpClient))
	}

	messagesProtocol := messagesapi.NewClient(messagesProtocolOpts...)
	messagesClient := client.NewMessagesClient(messagesProtocol)

	p := &Provider{
		cfg:             cfg,
		auth:            auth,
		models:          LoadModels(),
		responsesClient: responsesClient,
		messagesClient:  messagesClient,
	}

	return p, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return ProviderName
}

// GetModels returns the available models.
func (p *Provider) GetModels() Models {
	return p.models
}

// Capabilities returns the provider capabilities.
func (p *Provider) Capabilities() Capabilities {
	return Capabilities{
		// OpenRouter supports previous_response_id for Responses API models
		SupportsResponsesPreviousResponseID: true,
	}
}

// Capabilities holds provider capability flags.
type Capabilities struct {
	SupportsResponsesPreviousResponseID bool
}

// Stream implements conversation.Streamer.
// It routes to either the Responses API or Messages API based on the model.
func (p *Provider) Stream(ctx context.Context, req unified.Request) (<-chan client.StreamResult, error) {
	// Resolve model alias
	if req.Model != "" {
		model, _ := p.models.Resolve(req.Model)
		req.Model = model.ID
	} else {
		req.Model = p.cfg.Model
	}

	// Route based on model prefix
	if UseMessagesAPI(req.Model) {
		return p.messagesClient.Stream(ctx, req)
	}
	return p.responsesClient.Stream(ctx, req)
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

// filterThinkingParts removes thinking parts from messages since the Responses API doesn't support them.
func filterThinkingParts(req *unified.Request) {
	for i := range req.Messages {
		var filteredParts []unified.Part
		for _, part := range req.Messages[i].Parts {
			if part.Type == unified.PartTypeThinking {
				continue // skip thinking parts
			}
			filteredParts = append(filteredParts, part)
		}
		req.Messages[i].Parts = filteredParts
	}
}

// Ensure Provider implements conversation.Streamer.
var _ conversation.Streamer = (*Provider)(nil)
