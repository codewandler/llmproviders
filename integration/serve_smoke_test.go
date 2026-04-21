package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/codewandler/agentapis/api/responses"
	llmproviders "github.com/codewandler/llmproviders"
	"github.com/codewandler/llmproviders/internal/serve"
	"github.com/codewandler/llmproviders/providers/anthropic"
	"github.com/codewandler/llmproviders/providers/openrouter"
	"github.com/codewandler/llmproviders/registry"
)

// ptrInt returns a pointer to an int.
func ptrInt(n int) *int { return &n }

// serveModel describes a model variant for table-driven serve tests.
type serveModel struct {
	name           string // sub-test name
	model          string // model string sent in the request
	maxOutputTokens int   // max_output_tokens for the request
	skip           func(t *testing.T)
}

// serveTestModels returns the model variants to test.
// Each entry exercises a different code path through the proxy:
//
//   - openrouter/passthrough: OpenRouter → Responses API passthrough (no event translation)
//   - claude/translated:      Claude OAuth → Messages API → synthetic Responses events (full event translation)
func serveTestModels() []serveModel {
	return []serveModel{
		{
			name:           "openrouter/passthrough",
			model:          "openrouter/anthropic/claude-3-5-haiku",
			maxOutputTokens: 64,
			skip: func(t *testing.T) {
				t.Helper()
				if openrouter.EnvAPIKeyValue() == "" {
					t.Skip(openrouter.EnvAPIKey + " not set")
				}
			},
		},
		{
			name:           "claude/translated",
			model:          "claude/haiku",
			maxOutputTokens: 2048, // Anthropic thinking mode requires budget_tokens >= 1024
			skip: func(t *testing.T) {
				t.Helper()
				if !anthropic.LocalTokenStoreAvailable() {
					t.Skip("Claude OAuth credentials not available (~/.claude/.credentials.json)")
				}
			},
		},
	}
}

// startServeServerAll spins up the serve handler backed by a Service that has
// both OpenRouter and Claude OAuth providers registered (if credentials are
// available). It returns the base URL of the test server and a cleanup function.
func startServeServerAll(t *testing.T) (baseURL string, cleanup func()) {
	t.Helper()

	reg := registry.New()

	// Register OpenRouter if API key is available.
	if apiKey := openrouter.EnvAPIKeyValue(); apiKey != "" {
		reg.Register(registry.Registration{
			InstanceName:  openrouter.ProviderName,
			ServiceID:     openrouter.ServiceID,
			Order:         openrouter.OrderPriority,
			Aliases:       openrouter.ProviderAliases,
			IntentAliases: openrouter.IntentAliases,
			Detect:        func(ctx context.Context) (bool, error) { return true, nil },
			Build: func(ctx context.Context, cfg registry.BuildConfig) (registry.Provider, error) {
				return openrouter.New(openrouter.WithAPIKey(apiKey))
			},
		})
	}

	// Register Claude OAuth if credentials are available.
	if anthropic.LocalTokenStoreAvailable() {
		reg.Register(anthropic.ClaudeRegister)
	}

	svc, err := llmproviders.NewService(llmproviders.WithRegistry(reg))
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	logger := slog.Default()
	handler := serve.NewHandler(svc, logger)

	ts := httptest.NewServer(handler)
	return ts.URL, ts.Close
}

// newResponsesClient creates a responses.Client pointed at the test server.
func newResponsesClient(baseURL string) *responses.Client {
	return responses.NewClient(
		responses.WithBaseURL(baseURL),
		responses.WithAPIKey("unused"), // serve handler doesn't check auth
	)
}

// --------------------------------------------------------------------------
// Table-driven tests
// --------------------------------------------------------------------------

// TestServeBasicStream sends a simple streaming request through the serve
// proxy and verifies we get text back plus lifecycle events.
func TestServeBasicStream(t *testing.T) {
	requireIntegration(t)

	baseURL, cleanup := startServeServerAll(t)
	defer cleanup()

	for _, m := range serveTestModels() {
		t.Run(m.name, func(t *testing.T) {
			m.skip(t)

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			client := newResponsesClient(baseURL)

			req := responses.Request{
				Model: m.model,
				Input: responses.InputItems([]responses.InputItem{
					responses.InputItemFromMessage(responses.NewEasyInputMessage(
						"user",
						responses.EasyInputContentText("Say 'hello world' and nothing else."),
					)),
				}),
				MaxOutputTokens: ptrInt(m.maxOutputTokens),
			}

			stream, err := client.Stream(ctx, req)
			if err != nil {
				t.Fatalf("Stream() error: %v", err)
			}

			var (
				text        string
				hasCreated  bool
				hasComplete bool
			)
			for result := range stream {
				if result.Err != nil {
					t.Fatalf("stream error: %v", result.Err)
				}
				switch ev := result.Event.(type) {
				case *responses.ResponseCreatedEvent:
					hasCreated = true
				case *responses.OutputTextDeltaEvent:
					text += ev.Delta
				case *responses.ResponseCompletedEvent:
					hasComplete = true
				case *responses.APIErrorEvent:
					t.Fatalf("API error event: %s", ev.Message)
				}
			}

			if !hasCreated {
				t.Error("expected response.created event")
			}
			if !hasComplete {
				t.Error("expected response.completed event")
			}
			if text == "" {
				t.Fatal("expected text output, got empty")
			}
			t.Logf("Response text: %s", truncate(text, 200))
		})
	}
}

// TestServeStructuredInput sends a multi-message input array (instructions + user)
// through the proxy.
func TestServeStructuredInput(t *testing.T) {
	requireIntegration(t)

	baseURL, cleanup := startServeServerAll(t)
	defer cleanup()

	for _, m := range serveTestModels() {
		t.Run(m.name, func(t *testing.T) {
			m.skip(t)

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			client := newResponsesClient(baseURL)

			instr := "You are a helpful assistant. Always reply in exactly one sentence."
			req := responses.Request{
				Model:        m.model,
				Instructions: &instr,
				Input: responses.InputItems([]responses.InputItem{
					responses.InputItemFromMessage(responses.NewEasyInputMessage(
						"user",
						responses.EasyInputContentText("What is 2+2?"),
					)),
				}),
				MaxOutputTokens: ptrInt(m.maxOutputTokens),
			}

			stream, err := client.Stream(ctx, req)
			if err != nil {
				t.Fatalf("Stream() error: %v", err)
			}

			var text string
			for result := range stream {
				if result.Err != nil {
					t.Fatalf("stream error: %v", result.Err)
				}
				switch ev := result.Event.(type) {
				case *responses.OutputTextDeltaEvent:
					text += ev.Delta
				case *responses.APIErrorEvent:
					t.Fatalf("API error event: %s", ev.Message)
				}
			}

			if text == "" {
				t.Fatal("expected text output, got empty")
			}
			t.Logf("Response text: %s", truncate(text, 200))
		})
	}
}

// TestServeNonStreaming sends a non-streaming request. The proxy internally
// still streams from the upstream provider but collects and returns a single
// JSON response.
func TestServeNonStreaming(t *testing.T) {
	requireIntegration(t)

	baseURL, cleanup := startServeServerAll(t)
	defer cleanup()

	for _, m := range serveTestModels() {
		t.Run(m.name, func(t *testing.T) {
			m.skip(t)

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			body := fmt.Sprintf(`{
				"model": %q,
				"input": [{"type":"message","role":"user","content":"Say 'hello' and nothing else."}],
				"max_output_tokens": %d,
				"stream": false
			}`, m.model, m.maxOutputTokens)

			httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, strings.NewReader(body))
			if err != nil {
				t.Fatalf("NewRequest: %v", err)
			}
			httpReq.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(httpReq)
			if err != nil {
				t.Fatalf("HTTP request error: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("unexpected status: %d", resp.StatusCode)
			}

			ct := resp.Header.Get("Content-Type")
			if !strings.Contains(ct, "application/json") {
				t.Fatalf("expected application/json content-type, got %q", ct)
			}

			var result map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			output, ok := result["output"]
			if !ok {
				t.Fatalf("response missing 'output' field: %v", result)
			}
			t.Logf("Non-streaming response output: %v", truncate(fmt.Sprintf("%v", output), 300))
		})
	}
}

// --------------------------------------------------------------------------
// Error tests (not parameterised — provider-independent)
// --------------------------------------------------------------------------

// TestServeInvalidRequest verifies that the proxy returns a proper error
// for malformed requests (missing input).
func TestServeInvalidRequest(t *testing.T) {
	requireIntegration(t)

	baseURL, cleanup := startServeServerAll(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	body := `{"model": "some-model"}`

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, strings.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	var errResp map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	t.Logf("Error response: %v", errResp)
}

// TestServeUnknownModel verifies that the proxy returns 404 for unknown models.
func TestServeUnknownModel(t *testing.T) {
	requireIntegration(t)

	baseURL, cleanup := startServeServerAll(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	body := `{"model": "nonexistent/model-that-does-not-exist", "input": "hello"}`

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, strings.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	t.Logf("Got expected 404 for unknown model")
}
