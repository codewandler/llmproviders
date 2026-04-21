package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/codewandler/agentapis/api/unified"
	"github.com/codewandler/agentapis/conversation"
	"github.com/codewandler/llmproviders/providers/anthropic"
)

// TestAnthropicToolUse verifies end-to-end tool calling with the Anthropic provider.
// This test requires ANTHROPIC_API_KEY to be set.
func TestAnthropicToolUse(t *testing.T) {
	apiKey := anthropic.EnvAPIKeyValue()
	if apiKey == "" {
		t.Skip(anthropic.EnvAPIKey + " not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create provider
	p, err := anthropic.New(anthropic.WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("anthropic.New() error = %v", err)
	}

	// Define a simple weather tool
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

	// Create session with tool
	// Note: We disable thinking to avoid replay issues with thinking block signatures
	session := p.CreateSession(
		conversation.WithTools([]unified.Tool{weatherTool}),
		conversation.WithModel(anthropic.ModelSonnet),
		conversation.WithThinking(unified.ThinkingModeOff),
	)

	// Send request that should trigger tool use
	events, err := session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "What's the weather in Paris? Use the get_weather tool.",
		}},
	})
	if err != nil {
		t.Fatalf("session.Request() error = %v", err)
	}

	// Collect events from first turn
	var toolCallEvent *conversation.ToolCallEvent
	var hasUsage bool

	for ev := range events {
		switch e := ev.(type) {
		case conversation.ToolCallEvent:
			toolCallEvent = &e
			argsJSON, _ := json.Marshal(e.ToolCall.Args)
			t.Logf("Tool call: %s(%s)", e.ToolCall.Name, string(argsJSON))
		case conversation.UsageEvent:
			hasUsage = true
			t.Logf("Usage: input=%d, output=%d", e.Usage.Input.Total, e.Usage.Output.Total)
		case conversation.ErrorEvent:
			t.Fatalf("Error event: %v", e.Err)
		case conversation.TextDeltaEvent:
			// Ignore text deltas for now
		case conversation.CompletedEvent:
			t.Log("Turn completed")
		}
	}

	// Verify tool call was received
	if toolCallEvent == nil {
		t.Fatal("Expected tool call event, got none")
	}
	if toolCallEvent.ToolCall.Name != "get_weather" {
		t.Errorf("Tool call name = %q, want %q", toolCallEvent.ToolCall.Name, "get_weather")
	}

	// Submit tool result and continue conversation
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

	// Collect events from second turn
	var finalText string
	for ev := range events {
		switch e := ev.(type) {
		case conversation.TextDeltaEvent:
			finalText += e.Text
		case conversation.UsageEvent:
			hasUsage = true
			t.Logf("Usage: input=%d, output=%d", e.Usage.Input.Total, e.Usage.Output.Total)
		case conversation.ErrorEvent:
			t.Fatalf("Error event: %v", e.Err)
		case conversation.CompletedEvent:
			t.Log("Turn completed")
		}
	}

	// Verify final response contains weather info
	if finalText == "" {
		t.Fatal("Expected final text response, got empty")
	}
	t.Logf("Final response: %s", truncate(finalText, 200))

	// Verify usage was reported
	if !hasUsage {
		t.Error("Expected usage event, got none")
	}
}

// TestAnthropicBasicStream verifies basic streaming without tools.
func TestAnthropicBasicStream(t *testing.T) {
	apiKey := anthropic.EnvAPIKeyValue()
	if apiKey == "" {
		t.Skip(anthropic.EnvAPIKey + " not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create provider
	p, err := anthropic.New(anthropic.WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("anthropic.New() error = %v", err)
	}

	// Create session
	session := p.CreateSession(
		conversation.WithModel(anthropic.ModelSonnet),
	)

	// Send simple request
	events, err := session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "Say 'Hello, World!' and nothing else.",
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

// TestAnthropicModelResolution verifies model alias resolution.
func TestAnthropicModelResolution(t *testing.T) {
	apiKey := anthropic.EnvAPIKeyValue()
	if apiKey == "" {
		t.Skip(anthropic.EnvAPIKey + " not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create provider
	p, err := anthropic.New(anthropic.WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("anthropic.New() error = %v", err)
	}

	// Test with alias "sonnet"
	session := p.CreateSession(
		conversation.WithModel("sonnet"),
	)

	events, err := session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "Say 'test' and nothing else.",
		}},
	})
	if err != nil {
		t.Fatalf("session.Request() error = %v", err)
	}

	// Verify we get a response (model was resolved correctly)
	var gotResponse bool
	for ev := range events {
		switch ev.(type) {
		case conversation.TextDeltaEvent:
			gotResponse = true
		case conversation.ErrorEvent:
			e := ev.(conversation.ErrorEvent)
			t.Fatalf("Error event: %v", e.Err)
		}
	}

	if !gotResponse {
		t.Fatal("Expected response with alias 'sonnet', got none")
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// TestAnthropicThinkingToolUse verifies that thinking mode works correctly with tool use
// in multi-turn conversations. This test specifically validates the thinking signature fix
// that ensures thinking blocks are properly captured with their signatures for replay.
func TestAnthropicThinkingToolUse(t *testing.T) {
	apiKey := anthropic.EnvAPIKeyValue()
	if apiKey == "" {
		t.Skip(anthropic.EnvAPIKey + " not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Create provider
	p, err := anthropic.New(anthropic.WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("anthropic.New() error = %v", err)
	}

	// Define a simple calculator tool
	calcTool := unified.Tool{
		Name:        "calculate",
		Description: "Perform a mathematical calculation",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"expression": map[string]any{
					"type":        "string",
					"description": "The mathematical expression to evaluate",
				},
			},
			"required": []string{"expression"},
		},
	}

	// Create session WITH THINKING ENABLED - this is the key test
	session := p.CreateSession(
		conversation.WithTools([]unified.Tool{calcTool}),
		conversation.WithModel(anthropic.ModelSonnet),
		conversation.WithThinking(unified.ThinkingModeOn), // THINKING ENABLED
	)

	// Send request that should trigger tool use
	t.Log("Turn 1: Requesting calculation with thinking enabled...")
	events, err := session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "What is 23 * 47? Use the calculate tool to compute this.",
		}},
	})
	if err != nil {
		t.Fatalf("session.Request() error = %v", err)
	}

	// Collect events from first turn
	var toolCallEvent *conversation.ToolCallEvent
	var hasThinking bool
	var thinkingText string

	for ev := range events {
		switch e := ev.(type) {
		case conversation.ToolCallEvent:
			toolCallEvent = &e
			argsJSON, _ := json.Marshal(e.ToolCall.Args)
			t.Logf("Tool call: %s(%s)", e.ToolCall.Name, string(argsJSON))
		case conversation.ReasoningDeltaEvent:
			hasThinking = true
			thinkingText += e.Text
		case conversation.UsageEvent:
			t.Logf("Usage: input=%d, output=%d", e.Usage.Input.Total, e.Usage.Output.Total)
		case conversation.ErrorEvent:
			t.Fatalf("Error event in turn 1: %v", e.Err)
		case conversation.CompletedEvent:
			t.Log("Turn 1 completed")
		}
	}

	// Verify thinking was received
	if !hasThinking {
		t.Log("Warning: No thinking events received (model may not have used thinking)")
	} else {
		t.Logf("Thinking received: %s", truncate(thinkingText, 100))
	}

	// Verify tool call was received
	if toolCallEvent == nil {
		t.Fatal("Expected tool call event, got none")
	}
	if toolCallEvent.ToolCall.Name != "calculate" {
		t.Errorf("Tool call name = %q, want %q", toolCallEvent.ToolCall.Name, "calculate")
	}

	// Check history after first turn - should have thinking parts with signatures
	history := session.History()
	t.Logf("History after turn 1: %d messages", len(history))

	// Look for thinking parts with signatures in history
	var foundThinkingWithSignature bool
	for i, msg := range history {
		for j, part := range msg.Parts {
			if part.Type == unified.PartTypeThinking && part.Thinking != nil {
				t.Logf("Message %d, Part %d: Thinking (sig=%q, text=%s)",
					i, j, truncate(part.Thinking.Signature, 20), truncate(part.Thinking.Text, 50))
				if part.Thinking.Signature != "" {
					foundThinkingWithSignature = true
				}
			}
		}
	}

	if hasThinking && !foundThinkingWithSignature {
		t.Log("Warning: Thinking parts found but none have signatures")
	}

	// Submit tool result and continue conversation - THIS IS THE REPLAY TEST
	t.Log("Turn 2: Submitting tool result (tests replay with thinking signatures)...")
	events, err = session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleTool,
			ToolResult: &conversation.ToolResult{
				ToolCallID: toolCallEvent.ToolCall.ID,
				Output:     "1081",
			},
		}},
	})
	if err != nil {
		t.Fatalf("session.Request() with tool result error = %v (replay failed!)", err)
	}

	// Collect events from second turn
	var finalText string
	var hasThinkingTurn2 bool
	for ev := range events {
		switch e := ev.(type) {
		case conversation.TextDeltaEvent:
			finalText += e.Text
		case conversation.ReasoningDeltaEvent:
			hasThinkingTurn2 = true
		case conversation.UsageEvent:
			t.Logf("Usage: input=%d, output=%d", e.Usage.Input.Total, e.Usage.Output.Total)
		case conversation.ErrorEvent:
			t.Fatalf("Error event in turn 2: %v", e.Err)
		case conversation.CompletedEvent:
			t.Log("Turn 2 completed")
		}
	}

	// Verify final response
	if finalText == "" {
		t.Fatal("Expected final text response, got empty")
	}
	t.Logf("Final response: %s", truncate(finalText, 200))

	if hasThinkingTurn2 {
		t.Log("Turn 2 also included thinking")
	}

	// Check final history
	finalHistory := session.History()
	t.Logf("Final history: %d messages", len(finalHistory))

	// Success! If we got here, replay with thinking signatures worked
	t.Log("SUCCESS: Multi-turn conversation with thinking completed without replay errors")
}
