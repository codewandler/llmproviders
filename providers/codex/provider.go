package codex

import (
	"context"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/codewandler/agentapis/api/unified"
	responsesapi "github.com/codewandler/agentapis/api/responses"
	"github.com/codewandler/agentapis/client"
	lpprovider "github.com/codewandler/llmproviders/provider"
)

const defaultResponsesPath = "/codex/responses"

type Provider struct {
	cfg    Config
	client *client.ResponsesClient
}

type Option func(*Config)

func WithBaseURL(v string) Option { return func(c *Config) { c.BaseURL = v } }
func WithAPIKey(v string) Option  { return func(c *Config) { c.APIKey = v } }
func WithModel(v string) Option   { return func(c *Config) { c.Model = v } }
func WithTimeout(v time.Duration) Option {
	return func(c *Config) { c.Timeout = v }
}

func New(opts ...Option) *Provider {
	cfg := FromEnv()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	protocol := responsesapi.NewClient(
		responsesapi.WithBaseURL(cfg.BaseURL),
		responsesapi.WithPath(defaultResponsesPath),
		responsesapi.WithHTTPClient(&http.Client{Timeout: cfg.Timeout}),
		responsesapi.WithHeaderFunc(func(ctx context.Context, req *responsesapi.Request) (http.Header, error) {
			h := make(http.Header)
			if err := ApplyHeaders(h, cfg); err != nil {
				return nil, err
			}
			return h, nil
		}),
		responsesapi.WithRequestTransform(func(ctx context.Context, req *responsesapi.Request) error {
			if req.Model == "" {
				req.Model = cfg.Model
			}
			if req.Reasoning == nil {
				req.Reasoning = &responsesapi.Reasoning{}
			}
			if req.Reasoning.Effort == string(unified.EffortMax) {
				req.Reasoning.Effort = "xhigh"
			}
			if req.Reasoning.Summary == "" {
				req.Reasoning.Summary = "auto"
			}
			req.Store = false
			return nil
		}),
		responsesapi.WithHTTPRequestMutator(func(ctx context.Context, httpReq *http.Request, req *responsesapi.Request) error {
			if httpReq.Body == nil || httpReq.Header.Get("Content-Type") != "application/json" {
				return nil
			}
			body, err := io.ReadAll(httpReq.Body)
			if err != nil {
				return nil
			}
			_ = httpReq.Body.Close()
						var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
								return nil
			}
			delete(payload, "prompt_cache_retention")
			delete(payload, "max_tokens")
			delete(payload, "max_output_tokens")
			delete(payload, "temperature")
			delete(payload, "top_p")
			delete(payload, "top_k")
			delete(payload, "response_format")
			payload["store"] = false
			encoded, err := json.Marshal(payload)
			if err != nil {
				return nil
			}
						httpReq.ContentLength = int64(len(encoded))
			httpReq.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(io.NopCloser(nil)), nil }
			return rewriteBody(httpReq, encoded)
		}),
	)
	return &Provider{cfg: cfg, client: client.NewResponsesClient(protocol)}
}

func rewriteBody(req *http.Request, body []byte) error {
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
	req.ContentLength = int64(len(body))
	return nil
}

func (p *Provider) Name() string { return "codex" }

func (p *Provider) Capabilities() lpprovider.Capabilities { return staticCapabilities() }

func (p *Provider) Stream(ctx context.Context, req unified.Request) (<-chan client.StreamResult, error) {
	if err := p.cfg.Validate(); err != nil {
		return nil, err
	}
	working := req
	if working.Model == "" {
		working.Model = p.cfg.Model
	}
	if working.Model == "" {
		return nil, fmt.Errorf("codex: model is required")
	}
	return p.client.Stream(ctx, working)
}
