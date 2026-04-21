package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/google/uuid"

	responsesapi "github.com/codewandler/agentapis/api/responses"
	"github.com/codewandler/agentapis/api/unified"
	"github.com/codewandler/agentapis/client"
	"github.com/codewandler/agentapis/conversation"
)

// Provider implements the Codex provider using ChatGPT OAuth authentication.
// Codex uses the OpenAI Responses API backend at chatgpt.com/backend-api/codex/responses.
type Provider struct {
	cfg             Config
	auth            *Auth
	models          Models
	responsesClient *client.ResponsesClient
	httpClient      *http.Client

	// Turn state for sticky routing
	turnStateMu sync.RWMutex
	turnState   string // Captured from x-codex-turn-state response header
}

// New creates a new Codex provider using ChatGPT OAuth authentication.
// Requires a valid ~/.codex/auth.json file (created by `codex login`).
func New(opts ...Option) (*Provider, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	// Load auth from ~/.codex/auth.json
	auth, err := LoadAuth()
	if err != nil {
		return nil, fmt.Errorf("codex: load auth: %w", err)
	}

	// Generate installation ID if not provided
	installationID := o.installationID
	if installationID == "" {
		installationID = uuid.New().String()
	}

	// Build config
	cfg := Config{
		BaseURL:        o.baseURL,
		Model:          o.model,
		InstallationID: installationID,
		BetaFeatures:   o.betaFeatures,
	}

	p := &Provider{
		cfg:        cfg,
		auth:       auth,
		models:     LoadModels(),
		httpClient: o.httpClient,
	}

	if p.httpClient == nil {
		p.httpClient = http.DefaultClient
	}

	// Build protocol options
	protocolOpts := []responsesapi.Option{
		responsesapi.WithBaseURL(cfg.BaseURL),
		responsesapi.WithPath(DefaultPath),
		responsesapi.WithHeaderFunc(func(ctx context.Context, req *responsesapi.Request) (http.Header, error) {
			return p.buildHeaders(ctx, req)
		}),
		responsesapi.WithHTTPRequestMutator(func(ctx context.Context, httpReq *http.Request, req *responsesapi.Request) error {
			httpReq.Header.Set("Content-Type", "application/json")

			// Mutate request body
			return p.mutateRequestBody(httpReq)
		}),
		responsesapi.WithRequestTransform(func(ctx context.Context, req *responsesapi.Request) error {
			// Map effort: EffortMax -> xhigh
			if req.Reasoning != nil {
				if req.Reasoning.Effort == "max" {
					req.Reasoning.Effort = EffortXHigh
				}
				// Codex backend requires explicit summary to stream reasoning text
				if req.Reasoning.Summary == "" {
					req.Reasoning.Summary = "auto"
				}
			}
			return nil
		}),
	}

	if o.httpClient != nil {
		protocolOpts = append(protocolOpts, responsesapi.WithHTTPClient(o.httpClient))
	}

	// Create the low-level responses API client
	protocol := responsesapi.NewClient(protocolOpts...)

	// Client-level transforms
	clientOpts := []client.Option{
		client.WithRequestTransform(func(ctx context.Context, req *unified.Request) error {
			// Map effort based on thinking mode
			if req.Thinking.IsOn() && req.Effort.IsEmpty() {
				req.Effort = unified.EffortHigh
			}

			// Codex API requires instructions (system message)
			// Add a default if none is present
			hasSystem := false
			for _, msg := range req.Messages {
				if msg.Role == unified.RoleSystem {
					hasSystem = true
					break
				}
			}
			if !hasSystem {
				// Prepend a default system message
				systemMsg := unified.Message{
					Role: unified.RoleSystem,
					Parts: []unified.Part{{
						Type: unified.PartTypeText,
						Text: "You are a helpful assistant.",
					}},
				}
				req.Messages = append([]unified.Message{systemMsg}, req.Messages...)
			}

			return nil
		}),
	}

	// Create the high-level unified client
	p.responsesClient = client.NewResponsesClient(protocol, clientOpts...)

	return p, nil
}

// buildHeaders constructs the HTTP headers for Codex requests.
// Session-related headers (session_id, x-codex-window-id) are derived from req.PromptCacheKey,
// which is set by the conversation layer's session ID.
func (p *Provider) buildHeaders(ctx context.Context, req *responsesapi.Request) (http.Header, error) {
	h := make(http.Header)

	// Auth headers
	if err := p.auth.SetHeaders(ctx, h); err != nil {
		return nil, err
	}

	// Session tracking - derived from conversation session's PromptCacheKey
	if req.PromptCacheKey != "" {
		h.Set(HeaderSessionID, req.PromptCacheKey)
		// Window ID format: {session_id}:{window_generation}
		h.Set(HeaderCodexWindowID, req.PromptCacheKey+":0")
	}

	// Installation ID
	if p.cfg.InstallationID != "" {
		h.Set(HeaderCodexInstallationID, p.cfg.InstallationID)
	}

	// Beta features
	if p.cfg.BetaFeatures != "" {
		h.Set(HeaderCodexBetaFeatures, p.cfg.BetaFeatures)
	}

	// Sticky routing - replay turn state if we have one
	p.turnStateMu.RLock()
	if p.turnState != "" {
		h.Set(HeaderCodexTurnState, p.turnState)
	}
	p.turnStateMu.RUnlock()

	return h, nil
}

// mutateRequestBody applies Codex-specific transformations to the request body.
// Codex backend doesn't accept certain parameters that the standard OpenAI API does.
func (p *Provider) mutateRequestBody(r *http.Request) error {
	if r.Body == nil || r.Header.Get("Content-Type") != "application/json" {
		return nil
	}

	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		return err
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil // Not JSON, return as-is
	}

	// Codex requires store=false
	payload["store"] = false

	// Strip parameters that Codex backend doesn't accept
	delete(payload, "prompt_cache_retention")
	delete(payload, "max_tokens")
	delete(payload, "max_output_tokens")
	delete(payload, "temperature")
	delete(payload, "top_p")
	delete(payload, "top_k")
	delete(payload, "response_format")

	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	r.Body = io.NopCloser(bytes.NewReader(encoded))
	r.ContentLength = int64(len(encoded))

	return nil
}

// SetTurnState sets the sticky routing token for the current turn.
// This should be called when receiving the x-codex-turn-state response header.
func (p *Provider) SetTurnState(state string) {
	p.turnStateMu.Lock()
	p.turnState = state
	p.turnStateMu.Unlock()
}

// ClearTurnState clears the sticky routing token.
// Call this when starting a new turn.
func (p *Provider) ClearTurnState() {
	p.turnStateMu.Lock()
	p.turnState = ""
	p.turnStateMu.Unlock()
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
		// Codex backend does NOT support previous_response_id
		// Conversations use replay (resending full history)
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
	// Resolve model alias
	if req.Model != "" {
		model, _ := p.models.Resolve(req.Model)
		req.Model = model.ID
	} else {
		req.Model = p.cfg.Model
	}

	return p.responsesClient.Stream(ctx, req)
}

// CreateSession creates a new conversation session backed by this provider.
// The conversation layer auto-generates a session ID which flows to the provider
// via PromptCacheKey in the request, used for session_id and x-codex-window-id headers.
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

// FetchModels fetches available models from the Codex API.
func (p *Provider) FetchModels(ctx context.Context) ([]Model, error) {
	return FetchModels(ctx, p.auth, p.httpClient)
}

// FetchRawModels fetches the raw models JSON from the Codex API.
func (p *Provider) FetchRawModels(ctx context.Context) ([]byte, error) {
	return FetchRawModels(ctx, p.auth, p.httpClient)
}

// Ensure Provider implements conversation.Streamer.
var _ conversation.Streamer = (*Provider)(nil)
