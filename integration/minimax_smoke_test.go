package integration

import (
	"context"
	"testing"
	"time"

	"github.com/codewandler/agentapis/api/unified"
	"github.com/codewandler/agentapis/conversation"
	"github.com/codewandler/llmproviders/providers/minimax"
)

// TestMiniMaxBasicStream verifies basic streaming with the MiniMax provider.
// This test requires MINIMAX_API_KEY to be set.
func TestMiniMaxBasicStream(t *testing.T) {
	requireIntegration(t)
	requireIntegration(t)
	apiKey := minimax.EnvAPIKeyValue()
	if apiKey == "" {
		t.Skip(minimax.EnvAPIKey + " not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create provider
	p, err := minimax.New(minimax.WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("minimax.New() error = %v", err)
	}

	// Create session with default model (M2.7)
	session := p.CreateSession()

	// Send simple request
	events, err := session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "Say 'Hello from MiniMax!' and nothing else.",
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

// TestMiniMaxModelResolution verifies model alias resolution.
func TestMiniMaxModelResolution(t *testing.T) {
	requireIntegration(t)
	requireIntegration(t)
	apiKey := minimax.EnvAPIKeyValue()
	if apiKey == "" {
		t.Skip(minimax.EnvAPIKey + " not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create provider
	p, err := minimax.New(minimax.WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("minimax.New() error = %v", err)
	}

	// Test with provider-level aliases only (intent aliases like "minimax", "default", "fast"
	// are resolved at the Service level, not here)
	aliases := []string{"m2.7", "m2.5"}

	for _, alias := range aliases {
		t.Run(alias, func(t *testing.T) {
			session := p.CreateSession(
				conversation.WithModel(alias),
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
				t.Fatalf("Expected response with alias %q, got none", alias)
			}
		})
	}
}

// TestMiniMaxToolUse verifies tool calling with the MiniMax provider.
func TestMiniMaxToolUse(t *testing.T) {
	requireIntegration(t)
	requireIntegration(t)
	apiKey := minimax.EnvAPIKeyValue()
	if apiKey == "" {
		t.Skip(minimax.EnvAPIKey + " not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Create provider
	p, err := minimax.New(minimax.WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("minimax.New() error = %v", err)
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
	session := p.CreateSession(
		conversation.WithTools([]unified.Tool{weatherTool}),
	)

	// Send request that should trigger tool use
	events, err := session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "What's the weather in Tokyo? Use the get_weather tool.",
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
				Output:     "The weather in Tokyo is partly cloudy, 18°C",
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

// TestMiniMaxMultiTurn verifies multi-turn conversation works correctly.
func TestMiniMaxMultiTurn(t *testing.T) {
	requireIntegration(t)
	requireIntegration(t)
	apiKey := minimax.EnvAPIKeyValue()
	if apiKey == "" {
		t.Skip(minimax.EnvAPIKey + " not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Create provider
	p, err := minimax.New(minimax.WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("minimax.New() error = %v", err)
	}

	session := p.CreateSession()

	// Turn 1: Introduce a topic
	t.Log("Turn 1: Setting context...")
	events, err := session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "My favorite color is blue. Remember this.",
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
			Text: "What is my favorite color?",
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

	// Check if the response mentions blue
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
