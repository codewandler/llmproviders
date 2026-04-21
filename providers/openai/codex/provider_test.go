package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	responsesapi "github.com/codewandler/agentapis/api/responses"
)

func testAuth() *Auth {
	token := testJWT(time.Now().Add(time.Hour))
	return &Auth{
		auth: authFile{Tokens: tokenStore{AccessToken: token, AccountID: "acct-test"}},
		expiry:     time.Now().Add(time.Hour),
		httpClient: http.DefaultClient,
	}
}

func TestProviderBuildHeaders(t *testing.T) {
	p := &Provider{
		cfg: Config{InstallationID: "install-1", BetaFeatures: "feat-a,feat-b"},
		auth: testAuth(),
	}
	p.SetTurnState("sticky-1")

	h, err := p.buildHeaders(context.Background(), &responsesapi.Request{PromptCacheKey: "session-123"})
	if err != nil {
		t.Fatalf("buildHeaders() error = %v", err)
	}
	if got := h.Get("Authorization"); got == "" {
		t.Fatal("Authorization header missing")
	}
	if got := h.Get(HeaderChatGPTAccountID); got != "acct-test" {
		t.Fatalf("%s = %q, want acct-test", HeaderChatGPTAccountID, got)
	}
	if got := h.Get(HeaderSessionID); got != "session-123" {
		t.Fatalf("%s = %q, want session-123", HeaderSessionID, got)
	}
	if got := h.Get(HeaderCodexWindowID); got != "session-123:0" {
		t.Fatalf("%s = %q, want session-123:0", HeaderCodexWindowID, got)
	}
	if got := h.Get(HeaderCodexInstallationID); got != "install-1" {
		t.Fatalf("%s = %q, want install-1", HeaderCodexInstallationID, got)
	}
	if got := h.Get(HeaderCodexBetaFeatures); got != "feat-a,feat-b" {
		t.Fatalf("%s = %q, want feat-a,feat-b", HeaderCodexBetaFeatures, got)
	}
	if got := h.Get(HeaderCodexTurnState); got != "sticky-1" {
		t.Fatalf("%s = %q, want sticky-1", HeaderCodexTurnState, got)
	}
}

func TestProviderTurnState(t *testing.T) {
	p := &Provider{}
	p.SetTurnState("sticky")
	if p.turnState != "sticky" {
		t.Fatalf("turnState = %q, want sticky", p.turnState)
	}
	p.ClearTurnState()
	if p.turnState != "" {
		t.Fatalf("turnState = %q, want empty", p.turnState)
	}
}

func TestMutateRequestBody(t *testing.T) {
	payload := map[string]any{
		"prompt_cache_retention": "ephemeral",
		"max_tokens":             999,
		"max_output_tokens":      888,
		"temperature":            0.2,
		"top_p":                  0.9,
		"top_k":                  40,
		"response_format":        map[string]any{"type": "json_schema"},
		"input":                  "hello",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, "http://example.test", bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	p := &Provider{}
	if err := p.mutateRequestBody(req); err != nil {
		t.Fatalf("mutateRequestBody() error = %v", err)
	}
	mutated, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(mutated, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got["store"] != false {
		t.Fatalf("store = %#v, want false", got["store"])
	}
	for _, key := range []string{"prompt_cache_retention", "max_tokens", "max_output_tokens", "temperature", "top_p", "top_k", "response_format"} {
		if _, ok := got[key]; ok {
			t.Fatalf("key %q still present after mutation: %+v", key, got)
		}
	}
	if got["input"] != "hello" {
		t.Fatalf("input = %#v, want hello", got["input"])
	}
}

func TestMutateRequestBodySkipsNonJSON(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "http://example.test", bytes.NewBufferString("not-json"))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	p := &Provider{}
	if err := p.mutateRequestBody(req); err != nil {
		t.Fatalf("mutateRequestBody() error = %v", err)
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(body) != "not-json" {
		t.Fatalf("body = %q, want unchanged not-json", string(body))
	}
}
