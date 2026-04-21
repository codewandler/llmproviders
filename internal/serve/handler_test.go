package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/codewandler/agentapis/api/unified"
	"github.com/codewandler/agentapis/client"
	"github.com/codewandler/agentapis/conversation"
	llmproviders "github.com/codewandler/llmproviders"
	"github.com/codewandler/llmproviders/registry"
)

// mockProvider implements registry.Provider and conversation.Streamer.
type mockProvider struct {
	name   string
	events []client.StreamResult
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) CreateSession(opts ...conversation.Option) *conversation.Session {
	return conversation.New(m, opts...)
}
func (m *mockProvider) Stream(_ context.Context, _ unified.Request) (<-chan client.StreamResult, error) {
	ch := make(chan client.StreamResult, len(m.events))
	for _, ev := range m.events {
		ch <- ev
	}
	close(ch)
	return ch, nil
}

var _ conversation.Streamer = (*mockProvider)(nil)
var _ registry.Provider = (*mockProvider)(nil)

// mockNonStreamer implements registry.Provider but NOT conversation.Streamer.
type mockNonStreamer struct {
	name string
}

func (m *mockNonStreamer) Name() string { return m.name }
func (m *mockNonStreamer) CreateSession(opts ...conversation.Option) *conversation.Session {
	return nil
}
func (m *mockNonStreamer) Stream(ctx context.Context, req unified.Request) (<-chan client.StreamResult, error) {
	return nil, fmt.Errorf("streaming not supported")
}

var _ registry.Provider = (*mockNonStreamer)(nil)

// errorStreamer implements conversation.Streamer but always returns an error.
type errorStreamer struct {
	name string
}

func (m *errorStreamer) Name() string { return m.name }
func (m *errorStreamer) CreateSession(opts ...conversation.Option) *conversation.Session {
	return nil
}
func (m *errorStreamer) Stream(_ context.Context, _ unified.Request) (<-chan client.StreamResult, error) {
	return nil, fmt.Errorf("upstream connection refused")
}

var _ conversation.Streamer = (*errorStreamer)(nil)
var _ registry.Provider = (*errorStreamer)(nil)



func TestHandler_MethodNotAllowed(t *testing.T) {
	handler := NewHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/responses", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}

	var body apiError
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if body.Error.Code != "method_not_allowed" {
		t.Errorf("error code = %q, want %q", body.Error.Code, "method_not_allowed")
	}
}

func TestHandler_InvalidJSON(t *testing.T) {
	svc := mustBuildEmptyService(t)
	handler := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/responses",
		strings.NewReader(`{invalid json`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandler_MissingModel(t *testing.T) {
	svc := mustBuildEmptyService(t)
	handler := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/responses",
		strings.NewReader(`{"input":[{"role":"user","content":"Hi"}],"stream":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var body apiError
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if body.Error.Code != "invalid_request" {
		t.Errorf("error code = %q, want %q", body.Error.Code, "invalid_request")
	}
}

func TestHandler_ModelNotFound(t *testing.T) {
	svc := mustBuildEmptyService(t)
	handler := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/responses",
		strings.NewReader(`{"model":"nonexistent","input":[{"role":"user","content":"Hi"}],"stream":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandler_StreamingHappyPath(t *testing.T) {
	mock := &mockProvider{
		name: "test",
		events: []client.StreamResult{
			{Event: unified.StreamEvent{
				Type:    unified.StreamEventStarted,
				Started: &unified.Started{RequestID: "req_1", Model: "test-model"},
			}},
			{Event: unified.StreamEvent{
				Type: unified.StreamEventContentDelta,
				ContentDelta: &unified.ContentDelta{
					ContentBase: unified.ContentBase{
						Kind: unified.ContentKindText,
						Data: "Hello!",
					},
				},
			}},
			{Event: unified.StreamEvent{
				Type:      unified.StreamEventCompleted,
				Completed: &unified.Completed{StopReason: unified.StopReasonEndTurn},
			}},
		},
	}

	svc := mustBuildServiceWithProvider(t, mock)
	handler := NewHandler(svc, nil)

	body := `{"model":"test-alias","input":[{"role":"user","content":"Hi"}],"stream":true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	// Verify it's SSE.
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("content-type = %q, want %q", ct, "text/event-stream")
	}

	// Verify we got some events.
	output := rec.Body.String()
	if !strings.Contains(output, "event: response.created") {
		t.Error("missing response.created event")
	}
	if !strings.Contains(output, "event: response.output_text.delta") {
		t.Error("missing response.output_text.delta event")
	}
	if !strings.Contains(output, "event: response.completed") {
		t.Error("missing response.completed event")
	}
	if !strings.Contains(output, "Hello!") {
		t.Error("missing text delta content")
	}
}

// mustBuildEmptyService builds a service with no providers registered.
// It overrides the registry to avoid ErrNoProviders.
func mustBuildEmptyService(t *testing.T) *llmproviders.Service {
	t.Helper()
	reg := registry.New()
	// Register a dummy that never detects.
	reg.Register(registry.Registration{
		InstanceName: "__dummy",
		ServiceID:    "__dummy",
		Order:        9999,
		Detect:       func(context.Context) (bool, error) { return false, nil },
		Build:        func(context.Context, registry.BuildConfig) (registry.Provider, error) { return nil, nil },
	})
	svc, err := llmproviders.NewService(llmproviders.WithRegistry(reg))
	if err != nil {
		// ErrNoProviders is expected — create without registry.
		svc, err = llmproviders.NewService()
		if err != nil {
			t.Fatalf("failed to create service: %v", err)
		}
	}
	return svc
}

// mustBuildServiceWithProvider builds a service with a single mock provider.
func mustBuildServiceWithProvider(t *testing.T, mock registry.Provider) *llmproviders.Service {
	t.Helper()
	reg := registry.New()
	reg.Register(registry.Registration{
		InstanceName: mock.Name(),
		ServiceID:    mock.Name(),
		Order:        10,
		Aliases:      map[string]string{"test-alias": "test-model"},
		IntentAliases: map[string]string{"default": "test-model"},
		Detect:       func(context.Context) (bool, error) { return true, nil },
		Build: func(context.Context, registry.BuildConfig) (registry.Provider, error) {
			return mock, nil
		},
	})
	svc, err := llmproviders.NewService(llmproviders.WithRegistry(reg))
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	return svc
}

func TestHandler_UpstreamError(t *testing.T) {
	mock := &errorStreamer{name: "test"}
	svc := mustBuildServiceWithProvider(t, mock)
	handler := NewHandler(svc, nil)

	body := `{"model":"test-alias","input":[{"role":"user","content":"Hi"}],"stream":true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}

	var body2 apiError
	_ = json.NewDecoder(rec.Body).Decode(&body2)
	if body2.Error.Code != "upstream_error" {
		t.Errorf("error code = %q, want %q", body2.Error.Code, "upstream_error")
	}
}

func TestHandler_StreamWithError(t *testing.T) {
	mock := &mockProvider{
		name: "test",
		events: []client.StreamResult{
			{Event: unified.StreamEvent{
				Type:    unified.StreamEventStarted,
				Started: &unified.Started{RequestID: "req_1", Model: "test-model"},
			}},
			{Err: fmt.Errorf("rate limited")},
			{Event: unified.StreamEvent{
				Type:      unified.StreamEventCompleted,
				Completed: &unified.Completed{StopReason: unified.StopReasonEndTurn},
			}},
		},
	}

	svc := mustBuildServiceWithProvider(t, mock)
	handler := NewHandler(svc, nil)

	body := `{"model":"test-alias","input":[{"role":"user","content":"Hi"}],"stream":true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	output := rec.Body.String()
	if !strings.Contains(output, "event: response.created") {
		t.Error("missing response.created event")
	}
	if !strings.Contains(output, "error") {
		t.Error("missing error event")
	}
	if !strings.Contains(output, "rate limited") {
		t.Error("missing error message")
	}
}

func TestHandler_NonStreamingHappyPath(t *testing.T) {
	mock := &mockProvider{
		name: "test",
		events: []client.StreamResult{
			{Event: unified.StreamEvent{
				Type:    unified.StreamEventStarted,
				Started: &unified.Started{RequestID: "req_1", Model: "test-model"},
			}},
			{Event: unified.StreamEvent{
				Type: unified.StreamEventContentDelta,
				ContentDelta: &unified.ContentDelta{
					ContentBase: unified.ContentBase{
						Kind: unified.ContentKindText,
						Data: "Hello!",
					},
				},
			}},
			{Event: unified.StreamEvent{
				Type:      unified.StreamEventCompleted,
				Completed: &unified.Completed{StopReason: unified.StopReasonEndTurn},
			}},
		},
	}

	svc := mustBuildServiceWithProvider(t, mock)
	handler := NewHandler(svc, nil)

	// Note: stream:false (or absent)
	body := `{"model":"test-alias","input":[{"role":"user","content":"Hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	// Verify it's JSON, not SSE.
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q, want %q", ct, "application/json")
	}

	// Parse the response payload.
	var payload map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	if payload["status"] != "completed" {
		t.Errorf("status = %v, want %q", payload["status"], "completed")
	}
	if payload["id"] != "req_1" {
		t.Errorf("id = %v, want %q", payload["id"], "req_1")
	}

	// Verify output contains text.
	output, ok := payload["output"].([]interface{})
	if !ok || len(output) == 0 {
		t.Fatal("expected output array")
	}

	msg, ok := output[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected output[0] to be an object")
	}
	if msg["type"] != "message" {
		t.Errorf("output[0].type = %v, want %q", msg["type"], "message")
	}

	content, ok := msg["content"].([]interface{})
	if !ok || len(content) == 0 {
		t.Fatal("expected content array")
	}
	part, ok := content[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected content[0] to be an object")
	}
	if part["text"] != "Hello!" {
		t.Errorf("text = %v, want %q", part["text"], "Hello!")
	}
}

func TestHandler_NonStreamingExplicitFalse(t *testing.T) {
	mock := &mockProvider{
		name: "test",
		events: []client.StreamResult{
			{Event: unified.StreamEvent{
				Type:    unified.StreamEventStarted,
				Started: &unified.Started{RequestID: "req_2", Model: "test-model"},
			}},
			{Event: unified.StreamEvent{
				Type:      unified.StreamEventCompleted,
				Completed: &unified.Completed{StopReason: unified.StopReasonEndTurn},
			}},
		},
	}

	svc := mustBuildServiceWithProvider(t, mock)
	handler := NewHandler(svc, nil)

	// Explicitly stream:false
	body := `{"model":"test-alias","input":[{"role":"user","content":"Hi"}],"stream":false}`
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	// Must be JSON, not SSE.
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q, want %q", ct, "application/json")
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if payload["status"] != "completed" {
		t.Errorf("status = %v, want %q", payload["status"], "completed")
	}
}

func TestHandler_NonStreamingWithError(t *testing.T) {
	mock := &mockProvider{
		name: "test",
		events: []client.StreamResult{
			{Event: unified.StreamEvent{
				Type:    unified.StreamEventStarted,
				Started: &unified.Started{RequestID: "req_3", Model: "test-model"},
			}},
			{Err: fmt.Errorf("rate limited")},
			{Event: unified.StreamEvent{
				Type:      unified.StreamEventCompleted,
				Completed: &unified.Completed{StopReason: unified.StopReasonEndTurn},
			}},
		},
	}

	svc := mustBuildServiceWithProvider(t, mock)
	handler := NewHandler(svc, nil)

	body := `{"model":"test-alias","input":[{"role":"user","content":"Hi"}],"stream":false}`
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if payload["status"] != "failed" {
		t.Errorf("status = %v, want %q", payload["status"], "failed")
	}
	errObj, ok := payload["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error object")
	}
	if errObj["message"] != "rate limited" {
		t.Errorf("error message = %v, want %q", errObj["message"], "rate limited")
	}
}
