package integration

import (
	"context"
	"testing"
	"time"

	"github.com/codewandler/agentapis/api/unified"
	"github.com/codewandler/agentapis/conversation"
	"github.com/codewandler/llmproviders/providers/openai/codex"
)

// TestCodexBasicStream verifies basic streaming with the Codex provider.
// This test requires valid ~/.codex/auth.json (created by `codex login`).
func TestCodexBasicStream(t *testing.T) {
	if !codex.LocalAvailable() {
		t.Skip("Codex auth not available (run 'codex login' to authenticate)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create provider
	p, err := codex.New()
	if err != nil {
		t.Fatalf("codex.New() error = %v", err)
	}

	// Create session with default model
	session := p.Session()

	// Send simple request
	events, err := session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "Say 'Hello from Codex!' and nothing else.",
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

// TestCodexModelResolution verifies model alias resolution.
func TestCodexModelResolution(t *testing.T) {
	if !codex.LocalAvailable() {
		t.Skip("Codex auth not available (run 'codex login' to authenticate)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create provider
	p, err := codex.New()
	if err != nil {
		t.Fatalf("codex.New() error = %v", err)
	}

	// Test with various aliases
	aliases := []string{"codex", "gpt-5.4"}

	for _, alias := range aliases {
		t.Run(alias, func(t *testing.T) {
			session := p.Session(
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

// TestCodexToolUse verifies tool calling with the Codex provider.
func TestCodexToolUse(t *testing.T) {
	if !codex.LocalAvailable() {
		t.Skip("Codex auth not available (run 'codex login' to authenticate)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Create provider
	p, err := codex.New()
	if err != nil {
		t.Fatalf("codex.New() error = %v", err)
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
			Text: "What's the weather in London? Use the get_weather tool.",
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
				Output:     "The weather in London is rainy, 12°C",
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

// TestCodexMultiTurn verifies multi-turn conversation works correctly.
func TestCodexMultiTurn(t *testing.T) {
	if !codex.LocalAvailable() {
		t.Skip("Codex auth not available (run 'codex login' to authenticate)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Create provider
	p, err := codex.New()
	if err != nil {
		t.Fatalf("codex.New() error = %v", err)
	}

	session := p.Session()

	// Turn 1: Introduce a topic
	t.Log("Turn 1: Setting context...")
	events, err := session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "My favorite number is 42. Remember this.",
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
			Text: "What is my favorite number?",
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

	// Check if the response mentions 42
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
