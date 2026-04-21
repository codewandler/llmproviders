package anthropic

import (
	"context"
	"net/http"

	messagesapi "github.com/codewandler/agentapis/api/messages"
	"github.com/codewandler/agentapis/api/unified"
	"github.com/codewandler/agentapis/client"
	"github.com/codewandler/agentapis/conversation"
)

// Provider implements the Anthropic Messages API provider.
type Provider struct {
	cfg            Config
	auth           Auth
	models         Models
	messagesClient *client.MessagesClient
	claudeHeaders  bool

	// Behavior options
	autoSystemCacheControl bool
	autoSystemCacheTTL     string
}

// New creates a new Anthropic provider.
// By default, it uses the ANTHROPIC_API_KEY environment variable for authentication.
func New(opts ...Option) (*Provider, error) {
	o := applyOptions(opts)

	// Resolve auth
	auth := o.auth
	switch auth.(type) {
	case nil:
		// No auth specified - use API key from environment
		var err error
		auth, err = NewAPIKeyAuthFromEnv()
		if err != nil {
			return nil, err
		}
	case localOAuthMarker:
		// WithLocalOAuth() was called - explicitly create local OAuth auth
		var err error
		if o.claudeHeaders {
			auth, err = NewLocalOAuthAuthWithClaudeHeaders()
		} else {
			auth, err = NewLocalOAuthAuth()
		}
		if err != nil {
			return nil, err // No implicit fallback - fail if OAuth not available
		}
	}

	// Apply Claude headers to OAuth auth if requested
	if o.claudeHeaders {
		if oauthAuth, ok := auth.(*OAuthAuth); ok {
			oauthAuth.ClaudeHeaders = true
		}
	}

	// Build config
	cfg := Config{
		BaseURL:    o.baseURL,
		Model:      o.model,
		HTTPClient: o.httpClient,
	}

	// Build protocol options
	protocolOpts := []messagesapi.Option{
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
			// Set required Anthropic headers
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("Anthropic-Version", AnthropicVersion)
			httpReq.Header.Set("Anthropic-Beta", BetaInterleavedThinking)
			return nil
		}),
		messagesapi.WithRequestTransform(func(ctx context.Context, req *messagesapi.Request) error {
			// Apply thinking temperature coercion
			CoerceThinkingTemperature(req)
			return nil
		}),
	}

	// Add auto system cache control transform if enabled
	if o.autoSystemCacheControl {
		protocolOpts = append(protocolOpts, messagesapi.WithRequestTransform(
			messagesapi.AutoSystemCacheControlWithTTL(o.autoSystemCacheTTL),
		))
	}

	// Add rate limit callback if provided
	if o.rateLimitCallback != nil {
		protocolOpts = append(protocolOpts, messagesapi.WithResponseHook(
			func(_ context.Context, meta messagesapi.ResponseMeta) {
				rl := messagesapi.ParseRateLimits(meta.Headers)
				o.rateLimitCallback(rl)
			},
		))
	}

	// Create the low-level messages API client
	protocol := messagesapi.NewClient(protocolOpts...)

	// Create the high-level unified client
	messagesClient := client.NewMessagesClient(protocol)

	p := &Provider{
		cfg:                    cfg,
		auth:                   auth,
		models:                 LoadModels(),
		messagesClient:         messagesClient,
		claudeHeaders:          o.claudeHeaders,
		autoSystemCacheControl: o.autoSystemCacheControl,
		autoSystemCacheTTL:     o.autoSystemCacheTTL,
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
		SupportsResponsesPreviousResponseID: false, // Anthropic uses replay strategy
	}
}

// Capabilities holds provider capability flags.
type Capabilities struct {
	// SupportsResponsesPreviousResponseID indicates if the provider supports
	// the Responses API previous_response_id for conversation continuity.
	// Anthropic does not; it uses message replay.
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
