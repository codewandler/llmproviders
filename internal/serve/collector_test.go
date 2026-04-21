package serve

import (
	"fmt"
	"testing"

	"github.com/codewandler/agentapis/api/unified"
)

func TestResponseCollector_BasicText(t *testing.T) {
	c := NewResponseCollector("test-model")

	c.Add(unified.StreamEvent{
		Type:    unified.StreamEventStarted,
		Started: &unified.Started{RequestID: "req_1", Model: "test-model"},
	})
	c.Add(unified.StreamEvent{
		Type: unified.StreamEventContentDelta,
		ContentDelta: &unified.ContentDelta{
			ContentBase: unified.ContentBase{
				Kind: unified.ContentKindText,
				Data: "Hello ",
			},
		},
	})
	c.Add(unified.StreamEvent{
		Type: unified.StreamEventContentDelta,
		ContentDelta: &unified.ContentDelta{
			ContentBase: unified.ContentBase{
				Kind: unified.ContentKindText,
				Data: "world!",
			},
		},
	})
	c.Add(unified.StreamEvent{
		Type:      unified.StreamEventCompleted,
		Completed: &unified.Completed{StopReason: unified.StopReasonEndTurn},
	})

	payload := c.Finish()

	if payload.ID != "req_1" {
		t.Errorf("ID = %q, want %q", payload.ID, "req_1")
	}
	if payload.Model != "test-model" {
		t.Errorf("Model = %q, want %q", payload.Model, "test-model")
	}
	if payload.Status != "completed" {
		t.Errorf("Status = %q, want %q", payload.Status, "completed")
	}
	if len(payload.Output) == 0 {
		t.Fatal("expected at least one output item")
	}
	if payload.Output[0].Type != "message" {
		t.Errorf("output[0].Type = %q, want %q", payload.Output[0].Type, "message")
	}
	if len(payload.Output[0].Content) == 0 {
		t.Fatal("expected at least one content part")
	}
	if payload.Output[0].Content[0].Text != "Hello world!" {
		t.Errorf("text = %q, want %q", payload.Output[0].Content[0].Text, "Hello world!")
	}
}

func TestResponseCollector_ContentDoneOverrides(t *testing.T) {
	c := NewResponseCollector("test-model")

	c.Add(unified.StreamEvent{
		Type:    unified.StreamEventStarted,
		Started: &unified.Started{},
	})
	c.Add(unified.StreamEvent{
		Type: unified.StreamEventContentDelta,
		ContentDelta: &unified.ContentDelta{
			ContentBase: unified.ContentBase{Kind: unified.ContentKindText, Data: "partial"},
		},
	})
	// Content done should override the delta accumulation.
	c.Add(unified.StreamEvent{
		Type: unified.StreamEventContent,
		StreamContent: &unified.StreamContent{
			ContentBase: unified.ContentBase{Kind: unified.ContentKindText, Data: "final text"},
		},
	})
	c.Add(unified.StreamEvent{
		Type:      unified.StreamEventCompleted,
		Completed: &unified.Completed{StopReason: unified.StopReasonEndTurn},
	})

	payload := c.Finish()

	if len(payload.Output) == 0 || len(payload.Output[0].Content) == 0 {
		t.Fatal("expected output content")
	}
	if payload.Output[0].Content[0].Text != "final text" {
		t.Errorf("text = %q, want %q", payload.Output[0].Content[0].Text, "final text")
	}
}

func TestResponseCollector_MaxTokensIncomplete(t *testing.T) {
	c := NewResponseCollector("test-model")

	c.Add(unified.StreamEvent{
		Type:    unified.StreamEventStarted,
		Started: &unified.Started{},
	})
	c.Add(unified.StreamEvent{
		Type:      unified.StreamEventCompleted,
		Completed: &unified.Completed{StopReason: unified.StopReasonMaxTokens},
	})

	payload := c.Finish()

	if payload.Status != "incomplete" {
		t.Errorf("Status = %q, want %q", payload.Status, "incomplete")
	}
	if payload.IncompleteDetails == nil {
		t.Fatal("expected IncompleteDetails")
	}
	if payload.IncompleteDetails.Reason != "max_output_tokens" {
		t.Errorf("Reason = %q, want %q", payload.IncompleteDetails.Reason, "max_output_tokens")
	}
}

func TestResponseCollector_ErrorSetsFailedStatus(t *testing.T) {
	c := NewResponseCollector("test-model")

	c.Add(unified.StreamEvent{
		Type:    unified.StreamEventStarted,
		Started: &unified.Started{},
	})
	c.AddError(errTest)
	c.Add(unified.StreamEvent{
		Type:      unified.StreamEventCompleted,
		Completed: &unified.Completed{StopReason: unified.StopReasonEndTurn},
	})

	payload := c.Finish()

	if payload.Status != "failed" {
		t.Errorf("Status = %q, want %q", payload.Status, "failed")
	}
	if payload.Error == nil {
		t.Fatal("expected Error")
	}
	if payload.Error.Code != "upstream_error" {
		t.Errorf("Error.Code = %q, want %q", payload.Error.Code, "upstream_error")
	}
}

func TestResponseCollector_Usage(t *testing.T) {
	c := NewResponseCollector("test-model")

	c.Add(unified.StreamEvent{
		Type:    unified.StreamEventStarted,
		Started: &unified.Started{},
	})
	c.Add(unified.StreamEvent{
		Type: unified.StreamEventUsage,
		Usage: &unified.StreamUsage{
			Input:  unified.InputTokens{Total: 10},
			Output: unified.OutputTokens{Total: 20},
		},
	})
	c.Add(unified.StreamEvent{
		Type:      unified.StreamEventCompleted,
		Completed: &unified.Completed{StopReason: unified.StopReasonEndTurn},
	})

	payload := c.Finish()

	if payload.Usage == nil {
		t.Fatal("expected Usage")
	}
	if payload.Usage.InputTokens != 10 {
		t.Errorf("InputTokens = %d, want %d", payload.Usage.InputTokens, 10)
	}
	if payload.Usage.OutputTokens != 20 {
		t.Errorf("OutputTokens = %d, want %d", payload.Usage.OutputTokens, 20)
	}
}

func TestResponseCollector_ToolCall(t *testing.T) {
	c := NewResponseCollector("test-model")

	c.Add(unified.StreamEvent{
		Type:    unified.StreamEventStarted,
		Started: &unified.Started{},
	})
	c.Add(unified.StreamEvent{
		Type: unified.StreamEventToolCall,
		ToolCall: &unified.ToolCall{
			ID:   "call_123",
			Name: "get_weather",
			Args: map[string]any{"city": "Berlin"},
		},
	})
	c.Add(unified.StreamEvent{
		Type:      unified.StreamEventCompleted,
		Completed: &unified.Completed{StopReason: unified.StopReasonEndTurn},
	})

	payload := c.Finish()

	// Should have the initial message item + the tool call.
	found := false
	for _, item := range payload.Output {
		if item.Type == "function_call" {
			found = true
			if item.Name != "get_weather" {
				t.Errorf("Name = %q, want %q", item.Name, "get_weather")
			}
			if item.CallID != "call_123" {
				t.Errorf("CallID = %q, want %q", item.CallID, "call_123")
			}
		}
	}
	if !found {
		t.Error("expected a function_call output item")
	}
}

var errTest = fmt.Errorf("test error")
