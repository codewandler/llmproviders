package integration

import (
	"context"
	"testing"
	"time"

	"github.com/codewandler/agentapis/api/unified"
	"github.com/codewandler/agentapis/conversation"
	"github.com/codewandler/llmproviders/providers/openrouter"
)

// TestOpenRouterBasicStream verifies basic streaming with the OpenRouter provider.
// This test requires OPENROUTER_API_KEY to be set.
func TestOpenRouterBasicStream(t *testing.T) {
	apiKey := openrouter.EnvAPIKeyValue()
	if apiKey == "" {
		t.Skip(openrouter.EnvAPIKey + " not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create provider
	p, err := openrouter.New(openrouter.WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("openrouter.New() error = %v", err)
	}

	// Create session with auto model
	session := p.Session()

	// Send simple request
	events, err := session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "Say 'Hello from OpenRouter!' and nothing else.",
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
			t.Logf("Usage: input=%d, output=%d", e.Usage.Input.Total, e.Usage.Output.Total)
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

// TestOpenRouterToolUse verifies tool calling with the OpenRouter provider.
func TestOpenRouterToolUse(t *testing.T) {
	apiKey := openrouter.EnvAPIKeyValue()
	if apiKey == "" {
		t.Skip(openrouter.EnvAPIKey + " not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Create provider
	p, err := openrouter.New(openrouter.WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("openrouter.New() error = %v", err)
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
	session := p.Session(
		conversation.WithTools([]unified.Tool{weatherTool}),
	)

	// Send request that should trigger tool use
	events, err := session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "What's the weather in Berlin? Use the get_weather tool.",
		}},
	})
	if err != nil {
		t.Fatalf("session.Request() error = %v", err)
	}

	// Collect events from first turn
	var toolCallEvent *conversation.ToolCallEvent

	for ev := range events {
		switch e := ev.(type) {
		case conversation.ToolCallEvent:
			toolCallEvent = &e
			t.Logf("Tool call: %s", e.ToolCall.Name)
		case conversation.ErrorEvent:
			t.Fatalf("Error event: %v", e.Err)
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
				Output:     "The weather in Berlin is cloudy, 8°C",
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
}

// TestOpenRouterAnthropicModel tests using an Anthropic model through OpenRouter.
// This routes through the Messages API.
func TestOpenRouterAnthropicModel(t *testing.T) {
	apiKey := openrouter.EnvAPIKeyValue()
	if apiKey == "" {
		t.Skip(openrouter.EnvAPIKey + " not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create provider
	p, err := openrouter.New(openrouter.WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("openrouter.New() error = %v", err)
	}

	// Create session with Anthropic model (uses Messages API)
	// Using claude-3-haiku which is typically free/cheap on OpenRouter
	session := p.Session(
		conversation.WithModel("anthropic/claude-3-haiku"),
	)

	// Send simple request
	events, err := session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "Say 'Hello from Claude via OpenRouter!' and nothing else.",
		}},
	})
	if err != nil {
		t.Fatalf("session.Request() error = %v", err)
	}

	// Collect response
	var text string

	for ev := range events {
		switch e := ev.(type) {
		case conversation.TextDeltaEvent:
			text += e.Text
		case conversation.ErrorEvent:
			t.Fatalf("Error event: %v", e.Err)
		}
	}

	if text == "" {
		t.Fatal("Expected text response, got empty")
	}
	t.Logf("Response: %s", text)
}

// TestOpenRouterMultiTurn verifies multi-turn conversation works correctly.
func TestOpenRouterMultiTurn(t *testing.T) {
	apiKey := openrouter.EnvAPIKeyValue()
	if apiKey == "" {
		t.Skip(openrouter.EnvAPIKey + " not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Create provider
	p, err := openrouter.New(openrouter.WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("openrouter.New() error = %v", err)
	}

	session := p.Session()

	// Turn 1: Introduce a topic
	t.Log("Turn 1: Setting context...")
	events, err := session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "My pet's name is Max. Remember this.",
		}},
	})
	if err != nil {
		t.Fatalf("Turn 1 error = %v", err)
	}

	// Drain events
	for ev := range events {
		if e, ok := ev.(conversation.ErrorEvent); ok {
			t.Fatalf("Turn 1 error event: %v", e.Err)
		}
	}

	// Turn 2: Ask about the context
	t.Log("Turn 2: Testing memory...")
	events, err = session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "What is my pet's name?",
		}},
	})
	if err != nil {
		t.Fatalf("Turn 2 error = %v", err)
	}

	var response string
	for ev := range events {
		switch e := ev.(type) {
		case conversation.TextDeltaEvent:
			response += e.Text
		case conversation.ErrorEvent:
			t.Fatalf("Turn 2 error event: %v", e.Err)
		}
	}

	t.Logf("Response: %s", truncate(response, 200))

	if response == "" {
		t.Fatal("Expected response, got empty")
	}

	// Verify history is being tracked
	history := session.History()
	if len(history) < 4 { // 2 user + 2 assistant messages
		t.Errorf("Expected at least 4 messages in history, got %d", len(history))
	}
	t.Logf("History contains %d messages", len(history))
}
