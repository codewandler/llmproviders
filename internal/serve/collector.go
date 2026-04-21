package serve

import (
	"encoding/json"
	"time"

	"github.com/codewandler/agentapis/api/responses"
	"github.com/codewandler/agentapis/api/unified"
	"github.com/google/uuid"
)

// ResponseCollector accumulates unified stream events into a single
// responses.ResponsePayload for non-streaming (stream:false) requests.
type ResponseCollector struct {
	responseID string
	model      string
	createdAt  int64
	status     string

	// Current output items being built.
	output []responses.ResponseOutputItem

	// Text buffer for the current message content part.
	textParts map[outputContentKey]string

	// Tool call accumulation.
	toolArgs map[int]string // output_index → accumulated arguments

	// Tracking state.
	lastUsage  *unified.StreamUsage
	lastError  error
	stopReason unified.StopReason
}

// outputContentKey identifies a content part within the output.
type outputContentKey struct {
	outputIndex  int
	contentIndex int
}

// NewResponseCollector creates a new collector for a non-streaming response.
func NewResponseCollector(model string) *ResponseCollector {
	return &ResponseCollector{
		responseID: "resp_" + uuid.New().String(),
		model:      model,
		createdAt:  time.Now().Unix(),
		status:     "in_progress",
		textParts:  make(map[outputContentKey]string),
		toolArgs:   make(map[int]string),
	}
}

// Add processes a single unified stream event.
func (c *ResponseCollector) Add(ev unified.StreamEvent) {
	switch ev.Type {
	case unified.StreamEventStarted:
		if ev.Started != nil {
			if ev.Started.Model != "" {
				c.model = ev.Started.Model
			}
			if ev.Started.RequestID != "" {
				c.responseID = ev.Started.RequestID
			}
		}
		// Ensure we have at least one message output item.
		if len(c.output) == 0 {
			c.output = append(c.output, responses.ResponseOutputItem{
				ID:      "msg_" + uuid.New().String(),
				Type:    "message",
				Status:  "in_progress",
				Role:    "assistant",
				Content: []responses.ResponseContentPart{},
			})
		}

	case unified.StreamEventContentDelta:
		if ev.ContentDelta != nil {
			c.addContentDelta(*ev.ContentDelta)
		}

	case unified.StreamEventContent:
		if ev.StreamContent != nil {
			c.addContentDone(*ev.StreamContent)
		}

	case unified.StreamEventToolDelta:
		if ev.ToolDelta != nil {
			idx := 0
			if ev.ToolDelta.Ref.ItemIndex != nil {
				idx = int(*ev.ToolDelta.Ref.ItemIndex)
			}
			c.toolArgs[idx] += ev.ToolDelta.Data
		}

	case unified.StreamEventToolCall:
		if ev.StreamToolCall != nil {
			c.addToolCall(*ev.StreamToolCall)
		} else if ev.ToolCall != nil {
			c.addSimpleToolCall(*ev.ToolCall)
		}

	case unified.StreamEventLifecycle:
		if ev.Lifecycle != nil {
			c.addLifecycle(*ev.Lifecycle)
		}

	case unified.StreamEventUsage:
		if ev.Usage != nil {
			c.lastUsage = ev.Usage
		}

	case unified.StreamEventCompleted:
		if ev.Completed != nil {
			c.stopReason = ev.Completed.StopReason
		}
		if ev.Usage != nil {
			c.lastUsage = ev.Usage
		}

	case unified.StreamEventError:
		if ev.Error != nil && ev.Error.Err != nil {
			c.lastError = ev.Error.Err
		}
	}
}

// AddError records an error from a StreamResult.
func (c *ResponseCollector) AddError(err error) {
	c.lastError = err
}

// Finish builds the final ResponsePayload from accumulated events.
func (c *ResponseCollector) Finish() responses.ResponsePayload {
	// Determine final status.
	status := "completed"
	switch c.stopReason {
	case unified.StopReasonMaxTokens:
		status = "incomplete"
	case unified.StopReasonError:
		status = "failed"
	}
	if c.lastError != nil {
		status = "failed"
	}

	// Flush accumulated text into content parts.
	c.flushTextParts()

	// Mark all message items as completed.
	for i := range c.output {
		if c.output[i].Type == "message" {
			c.output[i].Status = status
			if c.output[i].Status == "failed" {
				c.output[i].Status = "incomplete"
			}
		}
	}

	payload := responses.ResponsePayload{
		ID:        c.responseID,
		Model:     c.model,
		CreatedAt: c.createdAt,
		Status:    status,
		Output:    c.output,
	}

	if c.lastUsage != nil {
		payload.Usage = buildResponsesUsage(c.lastUsage)
	}

	if c.lastError != nil {
		payload.Error = &responses.ResponseError{
			Code:    "upstream_error",
			Message: c.lastError.Error(),
		}
	}

	if status == "incomplete" {
		reason := "max_output_tokens"
		if c.lastError != nil {
			reason = "upstream_error"
		}
		payload.IncompleteDetails = &responses.IncompleteDetails{
			Reason: reason,
		}
	}

	return payload
}

func (c *ResponseCollector) addContentDelta(delta unified.ContentDelta) {
	outIdx := 0
	if delta.Ref.ItemIndex != nil {
		outIdx = int(*delta.Ref.ItemIndex)
	}
	contentIdx := 0
	if delta.Ref.SegmentIndex != nil {
		contentIdx = int(*delta.Ref.SegmentIndex)
	}

	switch delta.Kind {
	case unified.ContentKindText:
		key := outputContentKey{outputIndex: outIdx, contentIndex: contentIdx}
		c.textParts[key] += delta.Data
	case unified.ContentKindReasoning:
		// Reasoning content is accumulated but handled via summary in output items.
		// For non-streaming, we don't include raw reasoning tokens.
	}
}

func (c *ResponseCollector) addContentDone(content unified.StreamContent) {
	outIdx := 0
	if content.Ref.ItemIndex != nil {
		outIdx = int(*content.Ref.ItemIndex)
	}
	contentIdx := 0
	if content.Ref.SegmentIndex != nil {
		contentIdx = int(*content.Ref.SegmentIndex)
	}

	switch content.Kind {
	case unified.ContentKindText:
		// The done event has the final text — use it as authoritative.
		key := outputContentKey{outputIndex: outIdx, contentIndex: contentIdx}
		c.textParts[key] = content.Data
	}
}

func (c *ResponseCollector) addToolCall(tc unified.StreamToolCall) {
	idx := 0
	if tc.Ref.ItemIndex != nil {
		idx = int(*tc.Ref.ItemIndex)
	}

	item := responses.ResponseOutputItem{
		ID:        tc.Ref.ItemID,
		Type:      "function_call",
		Status:    "completed",
		CallID:    tc.ID,
		Name:      tc.Name,
		Arguments: tc.RawInput,
	}

	c.ensureOutputSlot(idx)
	if idx < len(c.output) && c.output[idx].Type == "function_call" {
		// Update existing slot.
		c.output[idx] = item
	} else {
		c.output = append(c.output, item)
	}
}

func (c *ResponseCollector) addSimpleToolCall(tc unified.ToolCall) {
	item := responses.ResponseOutputItem{
		Type:   "function_call",
		Status: "completed",
		CallID: tc.ID,
		Name:   tc.Name,
	}

	// Marshal args to JSON string.
	if tc.Args != nil {
		if argsJSON, err := json.Marshal(tc.Args); err == nil {
			item.Arguments = string(argsJSON)
		}
	}

	c.output = append(c.output, item)
}

func (c *ResponseCollector) addLifecycle(lc unified.Lifecycle) {
	if lc.Scope == unified.LifecycleScopeItem && lc.State == unified.LifecycleStateAdded {
		idx := 0
		if lc.Ref.ItemIndex != nil {
			idx = int(*lc.Ref.ItemIndex)
		}

		c.ensureOutputSlot(idx)
		if idx < len(c.output) {
			// Only update if the slot is empty/placeholder.
			if c.output[idx].ID == "" {
				c.output[idx].ID = lc.Ref.ItemID
				c.output[idx].Type = lc.ItemType
			}
		} else {
			c.output = append(c.output, responses.ResponseOutputItem{
				ID:   lc.Ref.ItemID,
				Type: lc.ItemType,
			})
		}
	}
}

func (c *ResponseCollector) ensureOutputSlot(idx int) {
	for len(c.output) <= idx {
		c.output = append(c.output, responses.ResponseOutputItem{})
	}
}

func (c *ResponseCollector) flushTextParts() {
	for key, text := range c.textParts {
		c.ensureOutputSlot(key.outputIndex)
		item := &c.output[key.outputIndex]

		// Ensure content slice is large enough.
		for len(item.Content) <= key.contentIndex {
			item.Content = append(item.Content, responses.ResponseContentPart{})
		}

		item.Content[key.contentIndex] = responses.ResponseContentPart{
			Type: "output_text",
			Text: text,
		}
	}
}
