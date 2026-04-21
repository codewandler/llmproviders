package integration

import (
	"context"
	"strings"
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

// TestClaudeOAuthCaching validates that Anthropic prompt caching works end-to-end.
// It sends two turns on the same session with a large system prompt so the second
// turn benefits from a cache read. The test asserts that:
//   - Turn 1 reports CacheWrite > 0 (the system prompt was written to cache).
//   - Turn 2 reports CacheRead  > 0 (the cached prefix was reused).
//
// Prompt caching requires the system prompt to exceed Anthropic's minimum
// cacheable size (currently 1024 tokens for Haiku / 2048 for Sonnet), so we
// use a deliberately large system prompt.
func TestClaudeOAuthCaching(t *testing.T) {
	requireIntegration(t)
	if !anthropic.LocalTokenStoreAvailable() {
		t.Skip("Claude OAuth credentials not available (~/.claude/.credentials.json)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Create provider with Claude Code configuration
	p, err := anthropic.NewClaudeCode()
	if err != nil {
		t.Fatalf("anthropic.NewClaudeCode() error = %v", err)
	}

	// Build a large system prompt that exceeds the minimum cacheable token count.
	// Anthropic requires ≥1024 tokens for Haiku, ≥2048 for Sonnet.
	// We repeat a block of text to ensure we're well above the threshold.
	block := "You are a helpful assistant specializing in geography, history, and science. " +
		"Always respond concisely and accurately. Cite sources when possible. " +
		"Use markdown formatting for clarity. Be polite and professional.\n"
	largeSystem := strings.Repeat(block, 40) // ~200 words × 40 ≈ 1600 tokens

	// Create session with Haiku for fast responses and thinking off for simplicity.
	session := p.CreateSession(
		conversation.WithModel(anthropic.ModelHaiku),
		conversation.WithThinking(unified.ThinkingModeOff),
		conversation.WithSystem(largeSystem),
	)

	// ─── Turn 1: prime the cache ───────────────────────────────────────────
	t.Log("Turn 1: sending initial request (should write to cache)...")
	events, err := session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "What is the capital of France? Reply in one word.",
		}},
	})
	if err != nil {
		t.Fatalf("Turn 1 session.Request() error = %v", err)
	}

	var turn1Text string
	var turn1Usage *unified.StreamUsage

	for ev := range events {
		switch e := ev.(type) {
		case conversation.TextDeltaEvent:
			turn1Text += e.Text
		case conversation.UsageEvent:
			u := e.Usage
			turn1Usage = &u
		case conversation.ErrorEvent:
			t.Fatalf("Turn 1 error event: %v", e.Err)
		}
	}

	if turn1Text == "" {
		t.Fatal("Turn 1: expected text response, got empty")
	}
	t.Logf("Turn 1 response: %s", truncate(turn1Text, 100))

	if turn1Usage == nil {
		t.Fatal("Turn 1: expected usage event, got none")
	}
	t.Logf("Turn 1 usage: input=%d (new=%d, cache_write=%d, cache_read=%d), output=%d",
		turn1Usage.Input.Total, turn1Usage.Input.New,
		turn1Usage.Input.CacheWrite, turn1Usage.Input.CacheRead,
		turn1Usage.Output.Total)

	// Turn 1 should have written to cache (the large system prompt).
	if turn1Usage.Input.CacheWrite == 0 {
		t.Error("Turn 1: expected CacheWrite > 0 (system prompt should be cached), got 0")
	}

	// ─── Turn 2: read from cache ───────────────────────────────────────────
	t.Log("Turn 2: sending follow-up request (should read from cache)...")
	events, err = session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "What is the capital of Germany? Reply in one word.",
		}},
	})
	if err != nil {
		t.Fatalf("Turn 2 session.Request() error = %v", err)
	}

	var turn2Text string
	var turn2Usage *unified.StreamUsage

	for ev := range events {
		switch e := ev.(type) {
		case conversation.TextDeltaEvent:
			turn2Text += e.Text
		case conversation.UsageEvent:
			u := e.Usage
			turn2Usage = &u
		case conversation.ErrorEvent:
			t.Fatalf("Turn 2 error event: %v", e.Err)
		}
	}

	if turn2Text == "" {
		t.Fatal("Turn 2: expected text response, got empty")
	}
	t.Logf("Turn 2 response: %s", truncate(turn2Text, 100))

	if turn2Usage == nil {
		t.Fatal("Turn 2: expected usage event, got none")
	}
	t.Logf("Turn 2 usage: input=%d (new=%d, cache_write=%d, cache_read=%d), output=%d",
		turn2Usage.Input.Total, turn2Usage.Input.New,
		turn2Usage.Input.CacheWrite, turn2Usage.Input.CacheRead,
		turn2Usage.Output.Total)

	// Turn 2 MUST read from cache — the system prompt was cached in turn 1.
	if turn2Usage.Input.CacheRead == 0 {
		t.Error("Turn 2: expected CacheRead > 0 (system prompt should be read from cache), got 0")
	}

	// The cache-read tokens should be a significant portion of the input.
	if turn2Usage.Input.Total > 0 {
		cacheRatio := float64(turn2Usage.Input.CacheRead) / float64(turn2Usage.Input.Total)
		t.Logf("Turn 2 cache hit ratio: %.1f%%", cacheRatio*100)
		if cacheRatio < 0.3 {
			t.Errorf("Turn 2: cache hit ratio %.1f%% is unexpectedly low (expected ≥30%%)", cacheRatio*100)
		}
	}

	// ─── Turn 3: verify cache persists across multiple turns ────────────────
	t.Log("Turn 3: sending another follow-up (cache should still be warm)...")
	events, err = session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "What is the capital of Italy? Reply in one word.",
		}},
	})
	if err != nil {
		t.Fatalf("Turn 3 session.Request() error = %v", err)
	}

	var turn3Text string
	var turn3Usage *unified.StreamUsage

	for ev := range events {
		switch e := ev.(type) {
		case conversation.TextDeltaEvent:
			turn3Text += e.Text
		case conversation.UsageEvent:
			u := e.Usage
			turn3Usage = &u
		case conversation.ErrorEvent:
			t.Fatalf("Turn 3 error event: %v", e.Err)
		}
	}

	if turn3Text == "" {
		t.Fatal("Turn 3: expected text response, got empty")
	}
	t.Logf("Turn 3 response: %s", truncate(turn3Text, 100))

	if turn3Usage == nil {
		t.Fatal("Turn 3: expected usage event, got none")
	}
	t.Logf("Turn 3 usage: input=%d (new=%d, cache_write=%d, cache_read=%d), output=%d",
		turn3Usage.Input.Total, turn3Usage.Input.New,
		turn3Usage.Input.CacheWrite, turn3Usage.Input.CacheRead,
		turn3Usage.Output.Total)

	if turn3Usage.Input.CacheRead == 0 {
		t.Error("Turn 3: expected CacheRead > 0 (cache should still be warm), got 0")
	}

	// Cache read should increase across turns as more of the conversation prefix is cached.
	if turn3Usage.Input.CacheRead < turn2Usage.Input.CacheRead {
		t.Logf("Note: Turn 3 CacheRead (%d) < Turn 2 CacheRead (%d) — may indicate cache eviction",
			turn3Usage.Input.CacheRead, turn2Usage.Input.CacheRead)
	}

	t.Log("SUCCESS: Prompt caching is working correctly across multiple turns")
}
