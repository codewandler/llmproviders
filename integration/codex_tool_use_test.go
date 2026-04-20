package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/codewandler/agentapis/api/unified"
		"github.com/codewandler/agentapis/conversation"
	"github.com/codewandler/llmproviders/pricing"
	"github.com/codewandler/llmproviders/providers/codex"
)

func TestCodexToolUseSmoke(t *testing.T) {
	if codex.APIKeyFromEnv() == "" {
		t.Skip("CODEX_API_KEY or OPENAI_API_KEY not configured")
	}

	tool := unified.Tool{
		Name:        "echo_tool",
		Description: "Returns the provided text unchanged.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"text": map[string]any{"type": "string"},
			},
			"required": []string{"text"},
		},
		Strict: true,
	}

	provider := codex.New()
	sess := conversation.New(
		provider,
		conversation.WithModel(codex.DefaultModelID()),
		conversation.WithTools([]unified.Tool{tool}),
		conversation.WithToolChoice(unified.ToolChoiceRequired{}),
		conversation.WithCapabilities(conversation.Capabilities{SupportsResponsesPreviousResponseID: provider.Capabilities().SupportsResponsesPreviousResponseID}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	first := conversation.NewRequest().
		User("Call the echo_tool with text=codex-tool-ok and do not answer directly before the tool result.").
		Build()

	firstStream, err := sess.Events(ctx, first)
	if err != nil {
		t.Fatalf("first request: %v", err)
	}
	toolCalls, usage1, raw1 := collectToolCallsAndUsage(t, firstStream)
	if len(toolCalls) == 0 {
		t.Fatalf("expected at least one tool call, raw events: %v", raw1)
	}
	call := toolCalls[0]
	if call.Name != "echo_tool" {
		t.Fatalf("expected echo_tool call, got %#v (raw events: %v)", call, raw1)
	}
	textArg, _ := call.Args["text"].(string)
	if !strings.Contains(strings.ToLower(textArg), "codex-tool-ok") {
		t.Fatalf("expected tool arg to contain codex-tool-ok, got %#v", call.Args)
	}

	second := conversation.NewRequest().
		ToolResult(call.ID, textArg).
		Build()
	secondStream, err := sess.Events(ctx, second)
	if err != nil {
		t.Fatalf("second request: %v", err)
	}
	finalText, usage2, raw2 := collectTextAndUsage(t, secondStream)
	if !strings.Contains(strings.ToLower(finalText), "codex-tool-ok") {
		t.Fatalf("expected final response to contain codex-tool-ok, got %q (raw events: %v)", finalText, raw2)
	}

	if _, ok := pricing.Lookup("codex", codex.DefaultModelID()); !ok {
		t.Fatalf("expected pricing entry for %s", codex.DefaultModelID())
	}
	_ = usage1
	_ = usage2
}

func TestCodexConversationNativeContinuationSmoke(t *testing.T) {
	if codex.APIKeyFromEnv() == "" {
		t.Skip("CODEX_API_KEY or OPENAI_API_KEY not configured")
	}

	provider := codex.New()
	sess := conversation.New(
		provider,
		conversation.WithModel(codex.DefaultModelID()),
		conversation.WithCapabilities(conversation.Capabilities{SupportsResponsesPreviousResponseID: provider.Capabilities().SupportsResponsesPreviousResponseID}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	token := "codex-continuation-token"

	first := conversation.NewRequest().
		User("Remember this exact token for the next turn: " + token + ". Reply with exactly: stored").
		Build()
	firstStream, err := sess.Events(ctx, first)
	if err != nil {
		t.Fatalf("first request: %v", err)
	}
	text1, _, raw1 := collectTextAndUsage(t, firstStream)
	if !strings.Contains(strings.ToLower(text1), "stored") {
		t.Fatalf("expected stored acknowledgement, got %q (raw events: %v)", text1, raw1)
	}

	second := conversation.NewRequest().
		User("What exact token did I ask you to remember? Reply only with that token.").
		Build()
	secondStream, err := sess.Events(ctx, second)
	if err != nil {
		t.Fatalf("second request: %v", err)
	}
	text2, _, raw2 := collectTextAndUsage(t, secondStream)
	if !strings.Contains(text2, token) {
		t.Fatalf("expected remembered token %q, got %q (raw events: %v)", token, text2, raw2)
	}
}

func collectToolCallsAndUsage(t *testing.T, stream <-chan conversation.Event) ([]unified.ToolCall, []unified.StreamUsage, []string) {
	t.Helper()
	var calls []unified.ToolCall
	var usages []unified.StreamUsage
	var raw []string
	for ev := range stream {
		switch e := ev.(type) {
		case conversation.ErrorEvent:
			t.Fatalf("stream item error: %v (events: %v)", e.Err, raw)
		case conversation.ToolCallEvent:
			raw = append(raw, "tool_call")
			calls = append(calls, e.ToolCall)
		case conversation.TransportUsageEvent:
			raw = append(raw, "usage")
			usages = append(usages, e.Usage)
		case conversation.CompletedEvent:
			raw = append(raw, "completed")
		}
	}
	return calls, usages, raw
}

func collectTextAndUsage(t *testing.T, stream <-chan conversation.Event) (string, []unified.StreamUsage, []string) {
	t.Helper()
	var text strings.Builder
	var fallback strings.Builder
	var usages []unified.StreamUsage
	var raw []string
	var sawCompleted bool
	for ev := range stream {
		switch e := ev.(type) {
		case conversation.ErrorEvent:
			t.Fatalf("stream item error: %v (events: %v)", e.Err, raw)
		case conversation.TextDeltaEvent:
			raw = append(raw, "text")
			text.WriteString(e.Text)
		case conversation.TransportUsageEvent:
			raw = append(raw, "usage")
			usages = append(usages, e.Usage)
		case conversation.CompletedEvent:
			raw = append(raw, "completed")
			sawCompleted = true
		case conversation.ReasoningDeltaEvent:
			raw = append(raw, "reasoning")
		}
	}
	if !sawCompleted {
		t.Fatalf("expected completed event, raw events: %v", raw)
	}
	if text.Len() > 0 {
		return text.String(), usages, raw
	}
	return fallback.String(), usages, raw
}
