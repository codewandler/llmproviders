package anthropic

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	messagesapi "github.com/codewandler/agentapis/api/messages"
	"github.com/codewandler/agentapis/api/unified"
	"github.com/codewandler/agentapis/client"
	"github.com/codewandler/agentapis/conversation"
)

const (
	claudeBillingHeader = "x-anthropic-billing-header: cc_version=2.1.85.613; cc_entrypoint=sdk-cli; cch=1757e;"
	claudeSystemCore    = "You are a Claude agent, built on Anthropic's Claude Agent SDK."
)

// Provider implements the Anthropic Messages API provider.
type Provider struct {
	name           string
	cfg            Config
	auth           Auth
	models         Models
	messagesClient *client.MessagesClient
	claudeHeaders  bool
	userID         string
	sessionID      string

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

	name := o.name
	if name == "" {
		name = ProviderName
	}

	p := &Provider{
		name:                   name,
		cfg:                    cfg,
		auth:                   auth,
		models:                 LoadModels(),
		claudeHeaders:          o.claudeHeaders,
		autoSystemCacheControl: o.autoSystemCacheControl,
		autoSystemCacheTTL:     o.autoSystemCacheTTL,
		sessionID:              randomUUID(),
	}
	if p.claudeHeaders {
		p.userID = buildClaudeUserID(p.sessionID)
	}

	// Build protocol options
	// Capture claudeHeaders for use in closures
	claudeHeaders := o.claudeHeaders

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
			httpReq.Header.Set("Accept", "application/json")
			httpReq.Header.Set("Anthropic-Version", AnthropicVersion)

			// Only set Anthropic-Beta if NOT using Claude headers.
			// Claude headers (set in oauth_auth.go) include the full beta string
			// with oauth-2025-04-20 which is required for OAuth authentication.
			if !claudeHeaders {
				httpReq.Header.Set("Anthropic-Beta", BetaInterleavedThinking)
			}

			// Add ?beta=true query parameter for Claude OAuth compatibility
			if claudeHeaders {
				q := httpReq.URL.Query()
				q.Set("beta", "true")
				httpReq.URL.RawQuery = q.Encode()
			}
			return nil
		}),
		messagesapi.WithRequestTransform(func(ctx context.Context, req *messagesapi.Request) error {
			// Apply thinking temperature coercion
			CoerceThinkingTemperature(req)
			if p.claudeHeaders {
				augmentClaudeRequest(req, p.userID, p.autoSystemCacheControl, p.autoSystemCacheTTL)
			}
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
	p.messagesClient = client.NewMessagesClient(protocol)

	return p, nil
}

// Name returns the provider instance name.
func (p *Provider) Name() string {
	return p.name
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

func augmentClaudeRequest(req *messagesapi.Request, userID string, autoSystemCacheControl bool, autoSystemCacheTTL string) {
	if req == nil {
		return
	}

	req.System = append(messagesapi.SystemBlocks{
		&messagesapi.TextBlock{Type: messagesapi.BlockTypeText, Text: claudeBillingHeader},
		&messagesapi.TextBlock{Type: messagesapi.BlockTypeText, Text: claudeSystemCore},
	}, req.System...)

	if autoSystemCacheControl && len(req.System) > 1 && req.System[1] != nil && req.System[1].CacheControl == nil {
		ttl := autoSystemCacheTTL
		if ttl == "" {
			ttl = "1h"
		}
		req.System[1].CacheControl = &messagesapi.CacheControl{Type: "ephemeral", TTL: ttl}
	}

	if userID != "" {
		req.Metadata = &messagesapi.Metadata{UserID: userID}
	}
}

func buildClaudeUserID(sessionID string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(home, ".claude.json"))
	if err != nil {
		return ""
	}

	var cfg struct {
		UserID       string `json:"userID"`
		OAuthAccount struct {
			AccountUUID string `json:"accountUuid"`
		} `json:"oauthAccount"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil || cfg.UserID == "" {
		return ""
	}

	id := map[string]string{
		"device_id":    cfg.UserID,
		"account_uuid": cfg.OAuthAccount.AccountUUID,
		"session_id":   sessionID,
	}
	data, err = json.Marshal(id)
	if err != nil {
		return ""
	}
	return string(data)
}

func randomUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("anthropic: crypto/rand unavailable: " + err.Error())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
var _ conversation.Streamer = (*Provider)(nil)
