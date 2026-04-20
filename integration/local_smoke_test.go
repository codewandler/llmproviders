package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/codewandler/agentapis/api/unified"
	"github.com/codewandler/agentapis/conversation"
	"github.com/codewandler/llmproviders/providers/dockermr"
	"github.com/codewandler/llmproviders/providers/ollama"
)

// skipUnlessLocalIntegration skips the test unless TEST_INTEGRATION_LOCAL is set.
// This allows running local provider tests only when the local services are available.
func skipUnlessLocalIntegration(t *testing.T) {
	if os.Getenv("TEST_INTEGRATION_LOCAL") == "" {
		t.Skip("TEST_INTEGRATION_LOCAL not set, skipping local provider tests")
	}
}

// =============================================================================
// Ollama Tests
// =============================================================================

func TestOllamaBasicStream(t *testing.T) {
	skipUnlessLocalIntegration(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	provider, err := ollama.New()
	if err != nil {
		t.Fatalf("Failed to create Ollama provider: %v", err)
	}

	session := provider.Session()

	events, err := session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "Say 'hello' and nothing else.",
		}},
	})
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	var response strings.Builder
	var gotUsage bool

	for ev := range events {
		switch e := ev.(type) {
		case conversation.TextDeltaEvent:
			response.WriteString(e.Text)
		case conversation.UsageEvent:
			gotUsage = true
			t.Logf("Usage: input=%d, output=%d", e.Usage.Input.Total, e.Usage.Output.Total)
		case conversation.ErrorEvent:
			t.Fatalf("Error event: %v", e.Err)
		}
	}

	text := response.String()
	if text == "" {
		t.Error("Expected non-empty response")
	}
	t.Logf("Response: %s", text)

	if !gotUsage {
		t.Log("Warning: No usage event received")
	}
}

func TestOllamaFetchModels(t *testing.T) {
	skipUnlessLocalIntegration(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	provider, err := ollama.New()
	if err != nil {
		t.Fatalf("Failed to create Ollama provider: %v", err)
	}

	models, err := provider.FetchModels(ctx)
	if err != nil {
		t.Fatalf("FetchModels failed: %v", err)
	}

	t.Logf("Found %d installed models", len(models))
	for _, m := range models {
		t.Logf("  - %s (%s)", m.ID, m.Name)
	}
}

func TestOllamaMultiTurn(t *testing.T) {
	skipUnlessLocalIntegration(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	provider, err := ollama.New()
	if err != nil {
		t.Fatalf("Failed to create Ollama provider: %v", err)
	}

	session := provider.Session()

	// Turn 1: Set context
	t.Log("Turn 1: Setting context...")
	events, err := session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "My favorite number is 42. Remember this.",
		}},
	})
	if err != nil {
		t.Fatalf("Turn 1 failed: %v", err)
	}
	drainEvents(t, events)

	// Turn 2: Test memory
	t.Log("Turn 2: Testing memory...")
	events, err = session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "What is my favorite number?",
		}},
	})
	if err != nil {
		t.Fatalf("Turn 2 failed: %v", err)
	}

	var response strings.Builder
	for ev := range events {
		switch e := ev.(type) {
		case conversation.TextDeltaEvent:
			response.WriteString(e.Text)
		case conversation.ErrorEvent:
			t.Fatalf("Error event: %v", e.Err)
		}
	}

	text := strings.ToLower(response.String())
	if !strings.Contains(text, "42") {
		t.Errorf("Expected response to contain '42', got: %s", response.String())
	}
	t.Logf("Response: %s", response.String())
}

// =============================================================================
// DockerMR Tests
// =============================================================================

func TestDockerMRBasicStream(t *testing.T) {
	skipUnlessLocalIntegration(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	provider, err := dockermr.New()
	if err != nil {
		t.Fatalf("Failed to create DockerMR provider: %v", err)
	}

	session := provider.Session()

	events, err := session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "Say 'hello' and nothing else.",
		}},
	})
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	var response strings.Builder
	var gotUsage bool

	for ev := range events {
		switch e := ev.(type) {
		case conversation.TextDeltaEvent:
			response.WriteString(e.Text)
		case conversation.UsageEvent:
			gotUsage = true
			t.Logf("Usage: input=%d, output=%d", e.Usage.Input.Total, e.Usage.Output.Total)
		case conversation.ErrorEvent:
			t.Fatalf("Error event: %v", e.Err)
		}
	}

	text := response.String()
	if text == "" {
		t.Error("Expected non-empty response")
	}
	t.Logf("Response: %s", text)

	if !gotUsage {
		t.Log("Warning: No usage event received")
	}
}

func TestDockerMRFetchModels(t *testing.T) {
	skipUnlessLocalIntegration(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	provider, err := dockermr.New()
	if err != nil {
		t.Fatalf("Failed to create DockerMR provider: %v", err)
	}

	models, err := provider.FetchModels(ctx)
	if err != nil {
		t.Fatalf("FetchModels failed: %v", err)
	}

	t.Logf("Found %d available models", len(models))
	for _, m := range models {
		t.Logf("  - %s (%s)", m.ID, m.Name)
	}
}

func TestDockerMRMultiTurn(t *testing.T) {
	skipUnlessLocalIntegration(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	provider, err := dockermr.New()
	if err != nil {
		t.Fatalf("Failed to create DockerMR provider: %v", err)
	}

	session := provider.Session()

	// Turn 1: Set context
	t.Log("Turn 1: Setting context...")
	events, err := session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "My favorite color is blue. Remember this.",
		}},
	})
	if err != nil {
		t.Fatalf("Turn 1 failed: %v", err)
	}
	drainEvents(t, events)

	// Turn 2: Test memory
	t.Log("Turn 2: Testing memory...")
	events, err = session.Request(ctx, conversation.Request{
		Inputs: []conversation.Input{{
			Role: unified.RoleUser,
			Text: "What is my favorite color?",
		}},
	})
	if err != nil {
		t.Fatalf("Turn 2 failed: %v", err)
	}

	var response strings.Builder
	for ev := range events {
		switch e := ev.(type) {
		case conversation.TextDeltaEvent:
			response.WriteString(e.Text)
		case conversation.ErrorEvent:
			t.Fatalf("Error event: %v", e.Err)
		}
	}

	text := strings.ToLower(response.String())
	if !strings.Contains(text, "blue") {
		t.Errorf("Expected response to contain 'blue', got: %s", response.String())
	}
	t.Logf("Response: %s", response.String())
}

// =============================================================================
// Helpers
// =============================================================================

func drainEvents(t *testing.T, events <-chan conversation.Event) {
	for ev := range events {
		switch e := ev.(type) {
		case conversation.ErrorEvent:
			t.Fatalf("Error event while draining: %v", e.Err)
		}
	}
}
