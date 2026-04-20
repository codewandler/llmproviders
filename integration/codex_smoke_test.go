package integration

import (
	"context"
	"testing"
	"time"

	"github.com/codewandler/agentapis/api/unified"
	"github.com/codewandler/llmproviders/providers/codex"
)

func TestCodexSmoke(t *testing.T) {
	if codex.APIKeyFromEnv() == "" {
		t.Skip("CODEX_API_KEY or OPENAI_API_KEY not configured")
	}
	p := codex.New()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	stream, err := p.Stream(ctx, unified.Request{
		Model: codex.DefaultModelID(),
		Messages: []unified.Message{{
			Role: unified.RoleUser,
			Parts: []unified.Part{{Type: unified.PartTypeText, Text: "Reply with exactly: codex-ok"}},
		}},
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	seenEvent := false
	for item := range stream {
		if item.Err != nil {
			t.Fatalf("stream item error: %v", item.Err)
		}
		seenEvent = true
	}
	if !seenEvent {
		t.Fatal("expected at least one stream event")
	}
}
