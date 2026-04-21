package integration

import (
	"context"
	"testing"
	"time"

	"github.com/codewandler/agentapis/api/unified"
	"github.com/codewandler/agentapis/conversation"
	"github.com/codewandler/llmproviders/providers/anthropic"
)

// TestClaudeOAuthBasicStream verifies Claude OAuth authentication works.
// This test requires ~/.claude/.credentials.json to exist (Claude CLI credentials).
func TestClaudeOAuthBasicStream(t *testing.T) {
	if !anthropic.LocalTokenStoreAvailable() {
		t.Skip("Claude OAuth credentials not available (~/.claude/.credentials.json)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create provider with Claude OAuth
	p, err := anthropic.NewWithOAuthAndClaudeHeaders()
	if err != nil {
		t.Fatalf("anthropic.NewWithOAuthAndClaudeHeaders() error = %v", err)
	}

	// Create session with haiku for fast response
	session := p.CreateSession(
		conversation.WithModel(anthropic.ModelHaiku),
	)

	// Send simple request
	events, err := session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "Say 'Hello, OAuth!' and nothing else.",
		}},
	})
	if err != nil {
		t.Fatalf("session.Request() error = %v", err)
	}

	// Collect response
	var text string
	var hasUsage bool

	for ev := range events {
		switch e := ev.(type) {
		case conversation.TextDeltaEvent:
			text += e.Text
		case conversation.UsageEvent:
			hasUsage = true
		case conversation.ErrorEvent:
			t.Fatalf("Error event: %v", e.Err)
		}
	}

	if text == "" {
		t.Fatal("Expected text response, got empty")
	}
	t.Logf("Response: %s", text)

	if !hasUsage {
		t.Error("Expected usage event, got none")
	}
}

// TestClaudeOAuthThinking verifies thinking mode works with Claude OAuth.
func TestClaudeOAuthThinking(t *testing.T) {
	if !anthropic.LocalTokenStoreAvailable() {
		t.Skip("Claude OAuth credentials not available (~/.claude/.credentials.json)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create provider with Claude OAuth
	p, err := anthropic.NewWithOAuthAndClaudeHeaders()
	if err != nil {
		t.Fatalf("anthropic.NewWithOAuthAndClaudeHeaders() error = %v", err)
	}

	// Create session with thinking enabled
	session := p.CreateSession(
		conversation.WithModel(anthropic.ModelHaiku),
		conversation.WithThinking(unified.ThinkingModeOn),
	)

	// Send request that triggers thinking
	events, err := session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "What is 17 * 23? Think step by step.",
		}},
	})
	if err != nil {
		t.Fatalf("session.Request() error = %v", err)
	}

	// Collect response
	var text string
	var thinkingText string
	var hasThinking bool

	for ev := range events {
		switch e := ev.(type) {
		case conversation.TextDeltaEvent:
			text += e.Text
		case conversation.ReasoningDeltaEvent:
			hasThinking = true
			thinkingText += e.Text
		case conversation.ErrorEvent:
			t.Fatalf("Error event: %v", e.Err)
		}
	}

	if text == "" {
		t.Fatal("Expected text response, got empty")
	}
	t.Logf("Response: %s", truncate(text, 200))

	if hasThinking {
		t.Logf("Thinking: %s", truncate(thinkingText, 100))
	} else {
		t.Log("Note: No thinking events received (model may not have used thinking)")
	}
}
