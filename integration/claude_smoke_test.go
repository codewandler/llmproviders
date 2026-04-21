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
	requireIntegration(t)
	requireIntegration(t)
	if !anthropic.LocalTokenStoreAvailable() {
		t.Skip("Claude OAuth credentials not available (~/.claude/.credentials.json)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create provider with Claude Code configuration
	p, err := anthropic.NewClaudeCode()
	if err != nil {
		t.Fatalf("anthropic.NewClaudeCode() error = %v", err)
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
	requireIntegration(t)
	requireIntegration(t)
	if !anthropic.LocalTokenStoreAvailable() {
		t.Skip("Claude OAuth credentials not available (~/.claude/.credentials.json)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create provider with Claude Code configuration
	p, err := anthropic.NewClaudeCode()
	if err != nil {
		t.Fatalf("anthropic.NewClaudeCode() error = %v", err)
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

// TestClaudeOAuthToolUse verifies end-to-end tool calling with the Claude Code configured provider.
func TestClaudeOAuthToolUse(t *testing.T) {
	requireIntegration(t)
	if !anthropic.LocalTokenStoreAvailable() {
		t.Skip("Claude OAuth credentials not available (~/.claude/.credentials.json)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	p, err := anthropic.NewClaudeCode()
	if err != nil {
		t.Fatalf("anthropic.NewClaudeCode() error = %v", err)
	}

	weatherTool := unified.Tool{
		Name:        "get_weather",
		Description: "Get the current weather for a location",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"location": map[string]any{
					"type":        "string",
					"description": "The city name",
				},
			},
			"required": []string{"location"},
		},
	}

	session := p.CreateSession(
		conversation.WithTools([]unified.Tool{weatherTool}),
		conversation.WithModel(anthropic.ModelSonnet),
		conversation.WithThinking(unified.ThinkingModeOff),
	)

	events, err := session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "What's the weather in Paris? Use the get_weather tool.",
		}},
	})
	if err != nil {
		t.Fatalf("session.Request() error = %v", err)
	}

	var toolCallEvent *conversation.ToolCallEvent
	for ev := range events {
		switch e := ev.(type) {
		case conversation.ToolCallEvent:
			toolCallEvent = &e
			t.Logf("Tool call: %s", e.ToolCall.Name)
		case conversation.ErrorEvent:
			t.Fatalf("Error event: %v", e.Err)
		}
	}

	if toolCallEvent == nil {
		t.Fatal("Expected tool call event, got none")
	}
	if toolCallEvent.ToolCall.Name != "get_weather" {
		t.Fatalf("Tool call name = %q, want get_weather", toolCallEvent.ToolCall.Name)
	}

	events, err = session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleTool,
			ToolResult: &conversation.ToolResult{
				ToolCallID: toolCallEvent.ToolCall.ID,
				Output:     "The weather in Paris is sunny, 22°C",
			},
		}},
	})
	if err != nil {
		t.Fatalf("session.Request() with tool result error = %v", err)
	}

	var finalText string
	for ev := range events {
		switch e := ev.(type) {
		case conversation.TextDeltaEvent:
			finalText += e.Text
		case conversation.ErrorEvent:
			t.Fatalf("Error event: %v", e.Err)
		}
	}

	if finalText == "" {
		t.Fatal("Expected final text response, got empty")
	}
	t.Logf("Final response: %s", truncate(finalText, 200))
}
