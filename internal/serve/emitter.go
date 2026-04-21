package serve

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/codewandler/agentapis/api/responses"
	"github.com/codewandler/agentapis/api/unified"
	"github.com/google/uuid"
)

// Emitter converts unified stream events to Responses API SSE events.
// It handles both raw passthrough (for providers that already produce Responses
// wire events) and synthetic construction (for Messages/Ollama providers).
type Emitter struct {
	responseID string
	model      string
	seqNum     int
	lastUsage  *unified.StreamUsage
}

// NewEmitter creates a new Emitter for a single response stream.
func NewEmitter(model string) *Emitter {
	return &Emitter{
		responseID: "resp_" + uuid.New().String(),
		model:      model,
	}
}

// Emit converts one unified.StreamEvent into zero or more SSE events.
func (e *Emitter) Emit(ev unified.StreamEvent) []SSEEvent {
	// Raw passthrough: if the upstream provider already produced Responses wire
	// events (OpenAI, OpenRouter, Codex), forward them as-is.
	// Only pass through events with Responses API event names ("response.*" or "error").
	if ev.Extras.RawEventName != "" && len(ev.Extras.RawJSON) > 0 && isResponsesEventName(ev.Extras.RawEventName) {
		// Still capture usage for our tracking even in passthrough mode.
		if ev.Usage != nil {
			e.lastUsage = ev.Usage
		}
		return []SSEEvent{{Name: ev.Extras.RawEventName, Data: ev.Extras.RawJSON}}
	}

	// Synthetic construction for non-Responses providers (Anthropic, Ollama, etc.)
	return e.synthesize(ev)
}

func (e *Emitter) synthesize(ev unified.StreamEvent) []SSEEvent {
	var out []SSEEvent

	switch ev.Type {
	case unified.StreamEventStarted:
		out = append(out, e.emitCreated(ev)...)

	case unified.StreamEventContentDelta:
		if ev.ContentDelta != nil {
			out = append(out, e.emitContentDelta(*ev.ContentDelta)...)
		}

	case unified.StreamEventContent:
		if ev.StreamContent != nil {
			out = append(out, e.emitContentDone(*ev.StreamContent)...)
		}

	case unified.StreamEventToolDelta:
		if ev.ToolDelta != nil {
			out = append(out, e.emitToolDelta(*ev.ToolDelta)...)
		}

	case unified.StreamEventToolCall:
		if ev.StreamToolCall != nil {
			out = append(out, e.emitToolCall(*ev.StreamToolCall)...)
		} else if ev.ToolCall != nil {
			out = append(out, e.emitToolCallFromSimple(*ev.ToolCall)...)
		}

	case unified.StreamEventLifecycle:
		if ev.Lifecycle != nil {
			out = append(out, e.emitLifecycle(*ev.Lifecycle)...)
		}

	case unified.StreamEventUsage:
		// Hold usage until completed event.
		if ev.Usage != nil {
			e.lastUsage = ev.Usage
		}

	case unified.StreamEventCompleted:
		out = append(out, e.emitCompleted(ev)...)

	case unified.StreamEventError:
		out = append(out, e.emitError(ev)...)
	}

	return out
}

func (e *Emitter) nextSeq() int {
	e.seqNum++
	return e.seqNum
}

func (e *Emitter) emitCreated(ev unified.StreamEvent) []SSEEvent {
	model := e.model
	if ev.Started != nil && ev.Started.Model != "" {
		model = ev.Started.Model
	}
	respID := e.responseID
	if ev.Started != nil && ev.Started.RequestID != "" {
		respID = ev.Started.RequestID
		e.responseID = respID
	}

	payload := responses.ResponsePayload{
		ID:        respID,
		Model:     model,
		CreatedAt: time.Now().Unix(),
		Status:    "in_progress",
		Output:    []responses.ResponseOutputItem{},
	}

	var out []SSEEvent
	out = append(out, e.marshalEvent(responses.EventResponseCreated, map[string]any{
		"type":            responses.EventResponseCreated,
		"sequence_number": e.nextSeq(),
		"response":        payload,
	}))
	out = append(out, e.marshalEvent(responses.EventResponseInProgress, map[string]any{
		"type":            responses.EventResponseInProgress,
		"sequence_number": e.nextSeq(),
		"response":        payload,
	}))

	// Emit output_item.added for the first message item.
	out = append(out, e.marshalEvent(responses.EventOutputItemAdded, map[string]any{
		"type":            responses.EventOutputItemAdded,
		"sequence_number": e.nextSeq(),
		"output_index":    0,
		"item": map[string]any{
			"id":      "msg_" + uuid.New().String(),
			"type":    "message",
			"status":  "in_progress",
			"role":    "assistant",
			"content": []any{},
		},
	}))

	// Emit content_part.added for the first text content part.
	out = append(out, e.marshalEvent(responses.EventContentPartAdded, map[string]any{
		"type":            responses.EventContentPartAdded,
		"sequence_number": e.nextSeq(),
		"output_index":    0,
		"content_index":   0,
		"part": map[string]any{
			"type": "output_text",
			"text": "",
		},
	}))

	return out
}

func (e *Emitter) emitContentDelta(delta unified.ContentDelta) []SSEEvent {
	switch delta.Kind {
	case unified.ContentKindText:
		return []SSEEvent{e.marshalEvent(responses.EventOutputTextDelta, map[string]any{
			"type":            responses.EventOutputTextDelta,
			"sequence_number": e.nextSeq(),
			"output_index":    refIndex(delta.Ref.ItemIndex, 0),
			"content_index":   refIndex(delta.Ref.SegmentIndex, 0),
			"delta":           delta.Data,
		})}

	case unified.ContentKindReasoning:
		if delta.Variant == unified.ContentVariantSummary {
			return []SSEEvent{e.marshalEvent(responses.EventReasoningSummaryTextDelta, map[string]any{
				"type":            responses.EventReasoningSummaryTextDelta,
				"sequence_number": e.nextSeq(),
				"output_index":    refIndex(delta.Ref.ItemIndex, 0),
				"item_id":         delta.Ref.ItemID,
				"delta":           delta.Data,
			})}
		}
		return []SSEEvent{e.marshalEvent(responses.EventReasoningTextDelta, map[string]any{
			"type":            responses.EventReasoningTextDelta,
			"sequence_number": e.nextSeq(),
			"output_index":    refIndex(delta.Ref.ItemIndex, 0),
			"item_id":         delta.Ref.ItemID,
			"delta":           delta.Data,
		})}

	default:
		return nil
	}
}

func (e *Emitter) emitContentDone(content unified.StreamContent) []SSEEvent {
	switch content.Kind {
	case unified.ContentKindText:
		return []SSEEvent{e.marshalEvent(responses.EventOutputTextDone, map[string]any{
			"type":            responses.EventOutputTextDone,
			"sequence_number": e.nextSeq(),
			"output_index":    refIndex(content.Ref.ItemIndex, 0),
			"content_index":   refIndex(content.Ref.SegmentIndex, 0),
			"text":            content.Data,
		})}
	case unified.ContentKindReasoning:
		if content.Variant == unified.ContentVariantSummary {
			return []SSEEvent{e.marshalEvent(responses.EventReasoningSummaryTextDone, map[string]any{
				"type":            responses.EventReasoningSummaryTextDone,
				"sequence_number": e.nextSeq(),
				"output_index":    refIndex(content.Ref.ItemIndex, 0),
				"item_id":         content.Ref.ItemID,
				"text":            content.Data,
			})}
		}
		return []SSEEvent{e.marshalEvent(responses.EventReasoningTextDone, map[string]any{
			"type":            responses.EventReasoningTextDone,
			"sequence_number": e.nextSeq(),
			"output_index":    refIndex(content.Ref.ItemIndex, 0),
			"item_id":         content.Ref.ItemID,
			"text":            content.Data,
		})}
	default:
		return nil
	}
}

func (e *Emitter) emitToolDelta(delta unified.ToolDelta) []SSEEvent {
	return []SSEEvent{e.marshalEvent(responses.EventFunctionCallArgumentsDelta, map[string]any{
		"type":            responses.EventFunctionCallArgumentsDelta,
		"sequence_number": e.nextSeq(),
		"output_index":    refIndex(delta.Ref.ItemIndex, 0),
		"item_id":         delta.Ref.ItemID,
		"delta":           delta.Data,
	})}
}

func (e *Emitter) emitToolCall(tc unified.StreamToolCall) []SSEEvent {
	return []SSEEvent{e.marshalEvent(responses.EventFunctionCallArgumentsDone, map[string]any{
		"type":            responses.EventFunctionCallArgumentsDone,
		"sequence_number": e.nextSeq(),
		"output_index":    refIndex(tc.Ref.ItemIndex, 0),
		"item_id":         tc.Ref.ItemID,
		"name":            tc.Name,
		"arguments":       tc.RawInput,
	})}
}

func (e *Emitter) emitToolCallFromSimple(tc unified.ToolCall) []SSEEvent {
	argsJSON, _ := json.Marshal(tc.Args)
	return []SSEEvent{e.marshalEvent(responses.EventFunctionCallArgumentsDone, map[string]any{
		"type":            responses.EventFunctionCallArgumentsDone,
		"sequence_number": e.nextSeq(),
		"output_index":    0,
		"name":            tc.Name,
		"call_id":         tc.ID,
		"arguments":       string(argsJSON),
	})}
}

func (e *Emitter) emitLifecycle(lc unified.Lifecycle) []SSEEvent {
	switch lc.Scope {
	case unified.LifecycleScopeItem:
		switch lc.State {
		case unified.LifecycleStateAdded:
			return []SSEEvent{e.marshalEvent(responses.EventOutputItemAdded, map[string]any{
				"type":            responses.EventOutputItemAdded,
				"sequence_number": e.nextSeq(),
				"output_index":    refIndex(lc.Ref.ItemIndex, 0),
				"item": map[string]any{
					"id":   lc.Ref.ItemID,
					"type": lc.ItemType,
				},
			})}
		case unified.LifecycleStateDone:
			return []SSEEvent{e.marshalEvent(responses.EventOutputItemDone, map[string]any{
				"type":            responses.EventOutputItemDone,
				"sequence_number": e.nextSeq(),
				"output_index":    refIndex(lc.Ref.ItemIndex, 0),
				"item": map[string]any{
					"id":   lc.Ref.ItemID,
					"type": lc.ItemType,
				},
			})}
		}

	case unified.LifecycleScopeSegment:
		switch lc.State {
		case unified.LifecycleStateAdded:
			return []SSEEvent{e.marshalEvent(responses.EventContentPartAdded, map[string]any{
				"type":            responses.EventContentPartAdded,
				"sequence_number": e.nextSeq(),
				"output_index":    refIndex(lc.Ref.ItemIndex, 0),
				"content_index":   refIndex(lc.Ref.SegmentIndex, 0),
				"part":            map[string]any{"type": "output_text", "text": ""},
			})}
		case unified.LifecycleStateDone:
			return []SSEEvent{e.marshalEvent(responses.EventContentPartDone, map[string]any{
				"type":            responses.EventContentPartDone,
				"sequence_number": e.nextSeq(),
				"output_index":    refIndex(lc.Ref.ItemIndex, 0),
				"content_index":   refIndex(lc.Ref.SegmentIndex, 0),
				"part":            map[string]any{"type": "output_text", "text": ""},
			})}
		}
	}

	return nil
}

func (e *Emitter) emitCompleted(ev unified.StreamEvent) []SSEEvent {
	var out []SSEEvent

	// Emit content_part.done and output_item.done before response.completed.
	out = append(out, e.marshalEvent(responses.EventContentPartDone, map[string]any{
		"type":            responses.EventContentPartDone,
		"sequence_number": e.nextSeq(),
		"output_index":    0,
		"content_index":   0,
		"part":            map[string]any{"type": "output_text", "text": ""},
	}))
	out = append(out, e.marshalEvent(responses.EventOutputItemDone, map[string]any{
		"type":            responses.EventOutputItemDone,
		"sequence_number": e.nextSeq(),
		"output_index":    0,
		"item": map[string]any{
			"type":   "message",
			"status": "completed",
			"role":   "assistant",
		},
	}))

	status := "completed"
	if ev.Completed != nil {
		switch ev.Completed.StopReason {
		case unified.StopReasonMaxTokens:
			status = "incomplete"
		case unified.StopReasonError:
			status = "failed"
		}
	}

	completedResp := map[string]any{
		"id":     e.responseID,
		"model":  e.model,
		"status": status,
	}
	if e.lastUsage != nil {
		completedResp["usage"] = buildResponsesUsage(e.lastUsage)
	}

	out = append(out, e.marshalEvent(responses.EventResponseCompleted, map[string]any{
		"type":            responses.EventResponseCompleted,
		"sequence_number": e.nextSeq(),
		"response":        completedResp,
	}))

	return out
}

func (e *Emitter) emitError(ev unified.StreamEvent) []SSEEvent {
	msg := "unknown error"
	if ev.Error != nil && ev.Error.Err != nil {
		msg = ev.Error.Err.Error()
	}
	return []SSEEvent{e.marshalEvent(responses.EventAPIError, map[string]any{
		"type":    responses.EventAPIError,
		"code":    "upstream_error",
		"message": msg,
	})}
}

func (e *Emitter) marshalEvent(name string, payload any) SSEEvent {
	data, err := json.Marshal(payload)
	if err != nil {
		data = []byte(fmt.Sprintf(`{"type":"error","message":"marshal error: %s"}`, err.Error()))
		name = responses.EventAPIError
	}
	return SSEEvent{Name: name, Data: data}
}

func buildResponsesUsage(u *unified.StreamUsage) *responses.ResponseUsage {
	if u == nil {
		return nil
	}
	ru := &responses.ResponseUsage{
		InputTokens:  u.Input.Total,
		OutputTokens: u.Output.Total,
	}
	if u.Input.CacheRead > 0 {
		ru.InputTokensDetails = &struct {
			CachedTokens int `json:"cached_tokens"`
		}{CachedTokens: u.Input.CacheRead}
	}
	if u.Output.Reasoning > 0 {
		ru.OutputTokensDetails = &struct {
			ReasoningTokens int `json:"reasoning_tokens"`
		}{ReasoningTokens: u.Output.Reasoning}
	}
	return ru
}

func refIndex(p *uint32, fallback int) int {
	if p != nil {
		return int(*p)
	}
	return fallback
}

// isResponsesEventName returns true if the event name is a valid Responses API
// SSE event name. This is used to distinguish Responses wire events (from OpenAI,
// OpenRouter, Codex) from other formats (Anthropic Messages API, Ollama, etc.).
func isResponsesEventName(name string) bool {
	return strings.HasPrefix(name, "response.") || name == "error"
}
