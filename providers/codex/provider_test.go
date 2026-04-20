package codex

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/codewandler/agentapis/api/unified"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.BaseURL == "" {
		t.Fatal("expected base URL")
	}
	if cfg.Model == "" {
		t.Fatal("expected default model")
	}
	if cfg.Timeout <= 0 {
		t.Fatal("expected positive timeout")
	}
}

func TestCapabilities(t *testing.T) {
	caps := staticCapabilities()
	if !caps.SupportsTools {
		t.Fatal("expected tools support")
	}
	if !caps.SupportsResponsesPreviousResponseID {
		t.Fatal("expected previous_response_id support")
	}
}

func TestNewAppliesOptions(t *testing.T) {
	p := New(WithAPIKey("x"), WithBaseURL("https://example.com"), WithModel("codex-test"), WithTimeout(5*time.Second))
	if p.cfg.APIKey != "x" {
		t.Fatal("expected API key option to apply")
	}
	if p.cfg.BaseURL != "https://example.com" {
		t.Fatal("expected base URL option to apply")
	}
	if p.cfg.Model != "codex-test" {
		t.Fatal("expected model option to apply")
	}
}

func TestCatalogModels(t *testing.T) {
	models := CatalogModels()
	if len(models) == 0 {
		t.Fatal("expected embedded catalog models")
	}
}

func TestStreamMutatesResponsesBodyForCodex(t *testing.T) {
	var gotBody map[string]any
	var gotHeader http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Clone()
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "event: response.created\ndata: {\"response\":{\"id\":\"resp_1\",\"model\":\"gpt-5.4\"}}\n\n")
		_, _ = io.WriteString(w, "event: response.completed\ndata: {\"response\":{\"id\":\"resp_1\",\"model\":\"gpt-5.4\",\"status\":\"completed\",\"usage\":{\"input_tokens\":10,\"output_tokens\":5}}}\n\n")
	}))
	defer server.Close()

	p := New(WithAPIKey("test-token"), WithBaseURL(server.URL), WithModel("gpt-5.4"), WithTimeout(5*time.Second))
	stream, err := p.Stream(context.Background(), unified.Request{
		Model:    "gpt-5.4",
		Thinking: unified.ThinkingModeOn,
		Effort:   unified.EffortMax,
		Messages: []unified.Message{{Role: unified.RoleUser, Parts: []unified.Part{{Type: unified.PartTypeText, Text: "Hello"}}}},
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	seen := 0
	for item := range stream {
		if item.Err != nil {
			t.Fatalf("stream item error: %v", item.Err)
		}
		seen++
	}
	if seen == 0 {
		t.Fatal("expected stream events")
	}
	store, ok := gotBody["store"].(bool)
	if !ok || store != false {
		t.Fatalf("expected store=false, got %#v", gotBody["store"])
	}
	if _, ok := gotBody["max_tokens"]; ok {
		t.Fatal("expected max_tokens stripped")
	}
	if _, ok := gotBody["max_output_tokens"]; ok {
		t.Fatal("expected max_output_tokens stripped")
	}
	reasoning, _ := gotBody["reasoning"].(map[string]any)
	if reasoning["effort"] != "xhigh" {
		t.Fatalf("expected xhigh reasoning effort, got %#v", reasoning["effort"])
	}
	if reasoning["summary"] != "auto" {
		t.Fatalf("expected auto reasoning summary, got %#v", reasoning["summary"])
	}
	if gotHeader.Get("Authorization") != "Bearer test-token" {
		t.Fatalf("unexpected auth header: %q", gotHeader.Get("Authorization"))
	}
	if gotHeader.Get(codexBetaHeader) != codexBetaValue {
		t.Fatalf("unexpected beta header: %q", gotHeader.Get(codexBetaHeader))
	}
}
