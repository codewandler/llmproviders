package serve

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/codewandler/agentapis/api/responses"
	"github.com/codewandler/agentapis/api/unified"
)

func TestEmitter_RawPassthrough(t *testing.T) {
	em := NewEmitter("gpt-4o")
	rawJSON := []byte(`{"type":"response.output_text.delta","delta":"hi"}`)
	ev := unified.StreamEvent{
		Type: unified.StreamEventContentDelta,
		Extras: unified.EventExtras{
			RawEventName: responses.EventOutputTextDelta,
			RawJSON:      rawJSON,
		},
	}

	events := em.Emit(ev)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Name != responses.EventOutputTextDelta {
		t.Errorf("event name = %q, want %q", events[0].Name, responses.EventOutputTextDelta)
	}
	if string(events[0].Data) != string(rawJSON) {
		t.Errorf("event data = %q, want %q", string(events[0].Data), string(rawJSON))
	}
}

func TestEmitter_SyntheticStarted(t *testing.T) {
	em := NewEmitter("claude-sonnet-4-6")
	ev := unified.StreamEvent{
		Type:    unified.StreamEventStarted,
		Started: &unified.Started{RequestID: "req_123", Model: "claude-sonnet-4-6"},
	}

	events := em.Emit(ev)
	// Should emit: response.created, response.in_progress, output_item.added, content_part.added
	if len(events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(events))
	}

	assertEventName(t, events[0], responses.EventResponseCreated)
	assertEventName(t, events[1], responses.EventResponseInProgress)
	assertEventName(t, events[2], responses.EventOutputItemAdded)
	assertEventName(t, events[3], responses.EventContentPartAdded)

	// Verify response.created has the right response ID.
	var created map[string]any
	mustUnmarshal(t, events[0].Data, &created)
	resp, _ := created["response"].(map[string]any)
	if resp["id"] != "req_123" {
		t.Errorf("response id = %v, want %q", resp["id"], "req_123")
	}
}

func TestEmitter_SyntheticTextDelta(t *testing.T) {
	em := NewEmitter("test-model")
	idx := uint32(0)
	ev := unified.StreamEvent{
		Type: unified.StreamEventContentDelta,
		ContentDelta: &unified.ContentDelta{
			ContentBase: unified.ContentBase{
				Ref:     unified.StreamRef{ItemIndex: &idx},
				Kind:    unified.ContentKindText,
				Variant: unified.ContentVariantPrimary,
				Data:    "Hello world",
			},
		},
	}

	events := em.Emit(ev)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	assertEventName(t, events[0], responses.EventOutputTextDelta)

	var payload map[string]any
	mustUnmarshal(t, events[0].Data, &payload)
	if payload["delta"] != "Hello world" {
		t.Errorf("delta = %v, want %q", payload["delta"], "Hello world")
	}
}

func TestEmitter_SyntheticReasoningDelta(t *testing.T) {
	em := NewEmitter("test-model")
	idx := uint32(0)
	ev := unified.StreamEvent{
		Type: unified.StreamEventContentDelta,
		ContentDelta: &unified.ContentDelta{
			ContentBase: unified.ContentBase{
				Ref:     unified.StreamRef{ItemIndex: &idx},
				Kind:    unified.ContentKindReasoning,
				Variant: unified.ContentVariantRaw,
				Data:    "thinking...",
			},
		},
	}

	events := em.Emit(ev)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	assertEventName(t, events[0], responses.EventReasoningTextDelta)
}

func TestEmitter_SyntheticReasoningSummaryDelta(t *testing.T) {
	em := NewEmitter("test-model")
	idx := uint32(0)
	ev := unified.StreamEvent{
		Type: unified.StreamEventContentDelta,
		ContentDelta: &unified.ContentDelta{
			ContentBase: unified.ContentBase{
				Ref:     unified.StreamRef{ItemIndex: &idx},
				Kind:    unified.ContentKindReasoning,
				Variant: unified.ContentVariantSummary,
				Data:    "summary...",
			},
		},
	}

	events := em.Emit(ev)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	assertEventName(t, events[0], responses.EventReasoningSummaryTextDelta)
}

func TestEmitter_SyntheticCompleted(t *testing.T) {
	em := NewEmitter("test-model")
	em.lastUsage = &unified.StreamUsage{
		Input:  unified.InputTokens{Total: 10, New: 10},
		Output: unified.OutputTokens{Total: 20},
		Tokens: unified.TokenItems{
			{Kind: unified.TokenKindInputNew, Count: 10},
			{Kind: unified.TokenKindOutput, Count: 20},
		},
	}

	ev := unified.StreamEvent{
		Type:      unified.StreamEventCompleted,
		Completed: &unified.Completed{StopReason: unified.StopReasonEndTurn},
	}

	events := em.Emit(ev)
	// Should emit: content_part.done, output_item.done, response.completed
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	assertEventName(t, events[0], responses.EventContentPartDone)
	assertEventName(t, events[1], responses.EventOutputItemDone)
	assertEventName(t, events[2], responses.EventResponseCompleted)

	// Verify usage in completed event.
	var completed map[string]any
	mustUnmarshal(t, events[2].Data, &completed)
	resp, _ := completed["response"].(map[string]any)
	usage, _ := resp["usage"].(map[string]any)
	if usage == nil {
		t.Fatal("expected usage in completed response")
	}
	if usage["input_tokens"] != float64(10) {
		t.Errorf("input_tokens = %v, want 10", usage["input_tokens"])
	}
	if usage["output_tokens"] != float64(20) {
		t.Errorf("output_tokens = %v, want 20", usage["output_tokens"])
	}
}

func TestEmitter_SyntheticError(t *testing.T) {
	em := NewEmitter("test-model")
	ev := unified.StreamEvent{
		Type: unified.StreamEventError,
		Error: &unified.StreamError{
			Err: fmt.Errorf("rate limited"),
		},
	}

	events := em.Emit(ev)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	assertEventName(t, events[0], responses.EventAPIError)
}

func TestEmitter_UsageHeldUntilCompleted(t *testing.T) {
	em := NewEmitter("test-model")

	// Emit usage event — should produce no SSE events.
	usageEv := unified.StreamEvent{
		Type: unified.StreamEventUsage,
		Usage: &unified.StreamUsage{
			Input:  unified.InputTokens{Total: 5, New: 5},
			Output: unified.OutputTokens{Total: 15},
		},
	}
	events := em.Emit(usageEv)
	if len(events) != 0 {
		t.Fatalf("expected 0 events for usage, got %d", len(events))
	}

	// Completed should include the usage.
	completedEv := unified.StreamEvent{
		Type:      unified.StreamEventCompleted,
		Completed: &unified.Completed{StopReason: unified.StopReasonEndTurn},
	}
	events = em.Emit(completedEv)
	// Find the response.completed event.
	var found bool
	for _, ev := range events {
		if ev.Name == responses.EventResponseCompleted {
			var payload map[string]any
			mustUnmarshal(t, ev.Data, &payload)
			resp, _ := payload["response"].(map[string]any)
			if resp["usage"] != nil {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected usage in response.completed event")
	}
}

func TestEmitter_ToolCallSimple(t *testing.T) {
	em := NewEmitter("test-model")
	ev := unified.StreamEvent{
		Type:     unified.StreamEventToolCall,
		ToolCall: &unified.ToolCall{ID: "call_1", Name: "get_weather", Args: map[string]any{"city": "Berlin"}},
	}

	events := em.Emit(ev)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	assertEventName(t, events[0], responses.EventFunctionCallArgumentsDone)

	var payload map[string]any
	mustUnmarshal(t, events[0].Data, &payload)
	if payload["name"] != "get_weather" {
		t.Errorf("name = %v, want %q", payload["name"], "get_weather")
	}
}

func TestEmitter_SequenceNumbers(t *testing.T) {
	em := NewEmitter("test-model")
	ev := unified.StreamEvent{
		Type:    unified.StreamEventStarted,
		Started: &unified.Started{RequestID: "req_1"},
	}
	events := em.Emit(ev)

	// All synthetic events should have incrementing sequence numbers.
	for i, sse := range events {
		var payload map[string]any
		mustUnmarshal(t, sse.Data, &payload)
		seq, ok := payload["sequence_number"]
		if !ok {
			continue
		}
		if seq.(float64) != float64(i+1) {
			t.Errorf("event %d: sequence_number = %v, want %d", i, seq, i+1)
		}
	}
}

func TestEmitter_SyntheticContentDoneText(t *testing.T) {
	em := NewEmitter("test-model")
	idx := uint32(0)
	ev := unified.StreamEvent{
		Type: unified.StreamEventContent,
		StreamContent: &unified.StreamContent{
			ContentBase: unified.ContentBase{
				Ref:  unified.StreamRef{ItemIndex: &idx},
				Kind: unified.ContentKindText,
				Data: "Complete text",
			},
		},
	}

	events := em.Emit(ev)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	assertEventName(t, events[0], responses.EventOutputTextDone)

	var payload map[string]any
	mustUnmarshal(t, events[0].Data, &payload)
	if payload["text"] != "Complete text" {
		t.Errorf("text = %v, want %q", payload["text"], "Complete text")
	}
}

func TestEmitter_SyntheticContentDoneReasoning(t *testing.T) {
	em := NewEmitter("test-model")
	idx := uint32(0)
	ev := unified.StreamEvent{
		Type: unified.StreamEventContent,
		StreamContent: &unified.StreamContent{
			ContentBase: unified.ContentBase{
				Ref:     unified.StreamRef{ItemIndex: &idx},
				Kind:    unified.ContentKindReasoning,
				Variant: unified.ContentVariantRaw,
				Data:    "done thinking",
			},
		},
	}

	events := em.Emit(ev)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	assertEventName(t, events[0], responses.EventReasoningTextDone)
}

func TestEmitter_SyntheticContentDoneReasoningSummary(t *testing.T) {
	em := NewEmitter("test-model")
	idx := uint32(0)
	ev := unified.StreamEvent{
		Type: unified.StreamEventContent,
		StreamContent: &unified.StreamContent{
			ContentBase: unified.ContentBase{
				Ref:     unified.StreamRef{ItemIndex: &idx},
				Kind:    unified.ContentKindReasoning,
				Variant: unified.ContentVariantSummary,
				Data:    "summary done",
			},
		},
	}

	events := em.Emit(ev)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	assertEventName(t, events[0], responses.EventReasoningSummaryTextDone)
}

func TestEmitter_SyntheticToolDelta(t *testing.T) {
	em := NewEmitter("test-model")
	idx := uint32(1)
	ev := unified.StreamEvent{
		Type: unified.StreamEventToolDelta,
		ToolDelta: &unified.ToolDelta{
			Ref:  unified.StreamRef{ItemIndex: &idx, ItemID: "call_abc"},
			Data: `{"city":`,
		},
	}

	events := em.Emit(ev)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	assertEventName(t, events[0], responses.EventFunctionCallArgumentsDelta)

	var payload map[string]any
	mustUnmarshal(t, events[0].Data, &payload)
	if payload["delta"] != `{"city":` {
		t.Errorf("delta = %v, want %q", payload["delta"], `{"city":`)
	}
	if payload["item_id"] != "call_abc" {
		t.Errorf("item_id = %v, want %q", payload["item_id"], "call_abc")
	}
	if payload["output_index"] != float64(1) {
		t.Errorf("output_index = %v, want 1", payload["output_index"])
	}
}

func TestEmitter_SyntheticStreamToolCall(t *testing.T) {
	em := NewEmitter("test-model")
	idx := uint32(2)
	ev := unified.StreamEvent{
		Type: unified.StreamEventToolCall,
		StreamToolCall: &unified.StreamToolCall{
			Ref:      unified.StreamRef{ItemIndex: &idx, ItemID: "call_xyz"},
			Name:     "read_file",
			RawInput: `{"path":"/tmp/x"}`,
		},
	}

	events := em.Emit(ev)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	assertEventName(t, events[0], responses.EventFunctionCallArgumentsDone)

	var payload map[string]any
	mustUnmarshal(t, events[0].Data, &payload)
	if payload["name"] != "read_file" {
		t.Errorf("name = %v, want %q", payload["name"], "read_file")
	}
	if payload["arguments"] != `{"path":"/tmp/x"}` {
		t.Errorf("arguments = %v, want %q", payload["arguments"], `{"path":"/tmp/x"}`)
	}
}

func TestEmitter_LifecycleItemAdded(t *testing.T) {
	em := NewEmitter("test-model")
	idx := uint32(1)
	ev := unified.StreamEvent{
		Type: unified.StreamEventLifecycle,
		Lifecycle: &unified.Lifecycle{
			Scope:    unified.LifecycleScopeItem,
			State:    unified.LifecycleStateAdded,
			Ref:      unified.StreamRef{ItemIndex: &idx, ItemID: "item_1"},
			ItemType: "function_call",
		},
	}

	events := em.Emit(ev)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	assertEventName(t, events[0], responses.EventOutputItemAdded)

	var payload map[string]any
	mustUnmarshal(t, events[0].Data, &payload)
	if payload["output_index"] != float64(1) {
		t.Errorf("output_index = %v, want 1", payload["output_index"])
	}
}

func TestEmitter_LifecycleItemDone(t *testing.T) {
	em := NewEmitter("test-model")
	idx := uint32(0)
	ev := unified.StreamEvent{
		Type: unified.StreamEventLifecycle,
		Lifecycle: &unified.Lifecycle{
			Scope:    unified.LifecycleScopeItem,
			State:    unified.LifecycleStateDone,
			Ref:      unified.StreamRef{ItemIndex: &idx, ItemID: "item_2"},
			ItemType: "message",
		},
	}

	events := em.Emit(ev)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	assertEventName(t, events[0], responses.EventOutputItemDone)
}

func TestEmitter_LifecycleSegmentAdded(t *testing.T) {
	em := NewEmitter("test-model")
	idx := uint32(0)
	seg := uint32(1)
	ev := unified.StreamEvent{
		Type: unified.StreamEventLifecycle,
		Lifecycle: &unified.Lifecycle{
			Scope: unified.LifecycleScopeSegment,
			State: unified.LifecycleStateAdded,
			Ref:   unified.StreamRef{ItemIndex: &idx, SegmentIndex: &seg},
		},
	}

	events := em.Emit(ev)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	assertEventName(t, events[0], responses.EventContentPartAdded)

	var payload map[string]any
	mustUnmarshal(t, events[0].Data, &payload)
	if payload["content_index"] != float64(1) {
		t.Errorf("content_index = %v, want 1", payload["content_index"])
	}
}

func TestEmitter_LifecycleSegmentDone(t *testing.T) {
	em := NewEmitter("test-model")
	idx := uint32(0)
	seg := uint32(0)
	ev := unified.StreamEvent{
		Type: unified.StreamEventLifecycle,
		Lifecycle: &unified.Lifecycle{
			Scope: unified.LifecycleScopeSegment,
			State: unified.LifecycleStateDone,
			Ref:   unified.StreamRef{ItemIndex: &idx, SegmentIndex: &seg},
		},
	}

	events := em.Emit(ev)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	assertEventName(t, events[0], responses.EventContentPartDone)
}

func TestEmitter_CompletedMaxTokens(t *testing.T) {
	em := NewEmitter("test-model")
	ev := unified.StreamEvent{
		Type:      unified.StreamEventCompleted,
		Completed: &unified.Completed{StopReason: unified.StopReasonMaxTokens},
	}

	events := em.Emit(ev)
	// Find response.completed
	for _, ev := range events {
		if ev.Name == responses.EventResponseCompleted {
			var payload map[string]any
			mustUnmarshal(t, ev.Data, &payload)
			resp, _ := payload["response"].(map[string]any)
			if resp["status"] != "incomplete" {
				t.Errorf("status = %v, want %q", resp["status"], "incomplete")
			}
			return
		}
	}
	t.Error("missing response.completed event")
}

func TestEmitter_CompletedError(t *testing.T) {
	em := NewEmitter("test-model")
	ev := unified.StreamEvent{
		Type:      unified.StreamEventCompleted,
		Completed: &unified.Completed{StopReason: unified.StopReasonError},
	}

	events := em.Emit(ev)
	for _, ev := range events {
		if ev.Name == responses.EventResponseCompleted {
			var payload map[string]any
			mustUnmarshal(t, ev.Data, &payload)
			resp, _ := payload["response"].(map[string]any)
			if resp["status"] != "failed" {
				t.Errorf("status = %v, want %q", resp["status"], "failed")
			}
			return
		}
	}
	t.Error("missing response.completed event")
}

func TestEmitter_UsageWithCacheAndReasoning(t *testing.T) {
	em := NewEmitter("test-model")
	em.lastUsage = &unified.StreamUsage{
		Input:  unified.InputTokens{Total: 100, CacheRead: 50},
		Output: unified.OutputTokens{Total: 200, Reasoning: 80},
	}

	ev := unified.StreamEvent{
		Type:      unified.StreamEventCompleted,
		Completed: &unified.Completed{StopReason: unified.StopReasonEndTurn},
	}

	events := em.Emit(ev)
	for _, ev := range events {
		if ev.Name == responses.EventResponseCompleted {
			var payload map[string]any
			mustUnmarshal(t, ev.Data, &payload)
			resp, _ := payload["response"].(map[string]any)
			usage, _ := resp["usage"].(map[string]any)
			if usage == nil {
				t.Fatal("expected usage")
			}
			if usage["input_tokens"] != float64(100) {
				t.Errorf("input_tokens = %v, want 100", usage["input_tokens"])
			}
			if usage["output_tokens"] != float64(200) {
				t.Errorf("output_tokens = %v, want 200", usage["output_tokens"])
			}
			// Check details
			inputDetails, _ := usage["input_tokens_details"].(map[string]any)
			if inputDetails == nil || inputDetails["cached_tokens"] != float64(50) {
				t.Errorf("cached_tokens = %v, want 50", inputDetails)
			}
			outputDetails, _ := usage["output_tokens_details"].(map[string]any)
			if outputDetails == nil || outputDetails["reasoning_tokens"] != float64(80) {
				t.Errorf("reasoning_tokens = %v, want 80", outputDetails)
			}
			return
		}
	}
	t.Error("missing response.completed event")
}

func TestEmitter_ContentDoneUnknownKind(t *testing.T) {
	em := NewEmitter("test-model")
	ev := unified.StreamEvent{
		Type: unified.StreamEventContent,
		StreamContent: &unified.StreamContent{
			ContentBase: unified.ContentBase{
				Kind: "unknown_kind",
				Data: "data",
			},
		},
	}

	events := em.Emit(ev)
	if len(events) != 0 {
		t.Errorf("expected 0 events for unknown kind, got %d", len(events))
	}
}

func TestEmitter_ContentDeltaUnknownKind(t *testing.T) {
	em := NewEmitter("test-model")
	ev := unified.StreamEvent{
		Type: unified.StreamEventContentDelta,
		ContentDelta: &unified.ContentDelta{
			ContentBase: unified.ContentBase{
				Kind: "unknown_kind",
				Data: "data",
			},
		},
	}

	events := em.Emit(ev)
	if len(events) != 0 {
		t.Errorf("expected 0 events for unknown kind, got %d", len(events))
	}
}

// Helpers

func assertEventName(t *testing.T, ev SSEEvent, expected string) {
	t.Helper()
	if ev.Name != expected {
		t.Errorf("event name = %q, want %q", ev.Name, expected)
	}
}

func mustUnmarshal(t *testing.T, data []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("unmarshal: %v (data: %s)", err, string(data))
	}
}

func TestEmitter_SyntheticEventsValidateAgainstOpenAPI(t *testing.T) {
	validator, err := NewOpenAPIEventValidator()
	if err != nil {
		t.Fatalf("NewOpenAPIEventValidator: %v", err)
	}

	t.Run("response.created", func(t *testing.T) {
		em := NewEmitter("claude-sonnet-4-6")
		events := em.Emit(unified.StreamEvent{
			Type:    unified.StreamEventStarted,
			Started: &unified.Started{RequestID: "req_123", Model: "claude-sonnet-4-6"},
		})
		for _, ev := range events {
			if err := validator.ValidateSSEEvent(ev); err != nil {
				t.Fatalf("ValidateSSEEvent(%s): %v\npayload=%s", ev.Name, err, string(ev.Data))
			}
		}
	})

	t.Run("response.output_text.delta", func(t *testing.T) {
		em := NewEmitter("test-model")
		idx := uint32(0)
		events := em.Emit(unified.StreamEvent{
			Type: unified.StreamEventContentDelta,
			ContentDelta: &unified.ContentDelta{
				ContentBase: unified.ContentBase{
					Ref:     unified.StreamRef{ItemIndex: &idx},
					Kind:    unified.ContentKindText,
					Variant: unified.ContentVariantPrimary,
					Data:    "Hello world",
				},
			},
		})
		for _, ev := range events {
			if err := validator.ValidateSSEEvent(ev); err != nil {
				t.Fatalf("ValidateSSEEvent(%s): %v\npayload=%s", ev.Name, err, string(ev.Data))
			}
		}
	})

	t.Run("response.completed", func(t *testing.T) {
		em := NewEmitter("test-model")
		em.lastUsage = &unified.StreamUsage{
			Input:  unified.InputTokens{Total: 10, New: 10},
			Output: unified.OutputTokens{Total: 20},
		}
		events := em.Emit(unified.StreamEvent{
			Type:      unified.StreamEventCompleted,
			Completed: &unified.Completed{StopReason: unified.StopReasonEndTurn},
		})
		for _, ev := range events {
			if err := validator.ValidateSSEEvent(ev); err != nil {
				t.Fatalf("ValidateSSEEvent(%s): %v\npayload=%s", ev.Name, err, string(ev.Data))
			}
		}
	})
}
