package serve

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/codewandler/agentapis/adapt"
	"github.com/codewandler/agentapis/api/responses"
	"github.com/codewandler/agentapis/api/unified"
	"github.com/codewandler/agentapis/client"
	"github.com/codewandler/agentapis/conversation"
	llmproviders "github.com/codewandler/llmproviders"
)

// Handler serves the OpenAI Responses API endpoint.
// It accepts POST requests with a responses.Request body, resolves the model
// through llmproviders.Service, streams the response from the upstream provider,
// and emits Responses API SSE events back to the client.
type Handler struct {
	service *llmproviders.Service
	logger  *slog.Logger
}

// NewHandler creates a new serve Handler.
func NewHandler(svc *llmproviders.Service, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{service: svc, logger: logger}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("handler received request", "method", r.Method, "path", r.URL.Path)
	
	if r.Method != http.MethodPost {
		h.logger.Debug("method not allowed", "method", r.Method)
		WriteJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST is allowed")
		return
	}

	// Read the entire body into a buffer so we can log it and then decode it.
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		h.logger.Debug("failed to read request body", "err", err)
		WriteJSONError(w, http.StatusBadRequest, "read_error", fmt.Sprintf("Failed to read body: %v", err))
		return
	}

	// Log the raw incoming request body at DEBUG level in pretty structured form when possible.
	logJSONPayload(h.logger, slog.LevelDebug, "request body", "raw", body)

	// Decode and validate the Responses API request.
	req, err := responses.DecodeRequest(body)
	if err != nil {
		var valErr *responses.RequestValidationError
		if errors.As(err, &valErr) {
			h.logger.Debug("request validation failed", "err", err)
			WriteJSONError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		h.logger.Debug("failed to decode request body", "err", err)
		WriteJSONError(w, http.StatusBadRequest, "invalid_json", fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	if req.Model == "" {
		h.logger.Debug("missing model field in request")
		WriteJSONError(w, http.StatusBadRequest, "missing_model", "The 'model' field is required")
		return
	}

	// Resolve model → provider + wire model ID.
	provider, wireModelID, err := h.service.ProviderFor(req.Model)
	if err != nil {
		if errors.Is(err, llmproviders.ErrModelNotFound) || errors.Is(err, llmproviders.ErrProviderNotFound) {
			h.logger.Debug("model not found", "model", req.Model)
			WriteJSONError(w, http.StatusNotFound, "model_not_found",
				fmt.Sprintf("Model %q not found", req.Model))
			return
		}
		if errors.Is(err, llmproviders.ErrAmbiguousModel) {
			h.logger.Debug("ambiguous model", "model", req.Model, "err", err)
			WriteJSONError(w, http.StatusBadRequest, "ambiguous_model", err.Error())
			return
		}
		h.logger.Debug("model resolution error", "model", req.Model, "err", err)
		WriteJSONError(w, http.StatusInternalServerError, "resolve_error",
			fmt.Sprintf("Failed to resolve model: %v", err))
		return
	}

	h.logger.Info("request",
		"client_model", req.Model,
		"provider", provider.Name(),
		"wire_model", wireModelID,
	)

	// Convert the responses.Request → unified.Request.
	// Overwrite the model with the resolved wire model ID before conversion.
	req.Model = wireModelID
	unifiedReq, err := adapt.RequestFromResponses(req)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request",
			fmt.Sprintf("Failed to convert request: %v", err))
		return
	}

	h.logger.Info("converted request", "unified_request", unifiedReq)

	// Stream via the provider.
	// We call provider.Stream() directly (the conversation.Streamer interface)
	// to get the raw unified event stream without session state management.
	ctx := r.Context()
	streamer, ok := provider.(conversation.Streamer)
	if !ok {
		WriteJSONError(w, http.StatusInternalServerError, "provider_error",
			"Provider does not support streaming")
		return
	}

	stream, err := streamer.Stream(ctx, unifiedReq)
	if err != nil {
		WriteJSONError(w, http.StatusBadGateway, "upstream_error",
			fmt.Sprintf("Upstream request failed: %v", err))
		return
	}

	if req.Stream != nil && *req.Stream {
		h.serveStreaming(w, ctx, wireModelID, stream)
	} else {
		h.serveNonStreaming(w, ctx, wireModelID, stream)
	}
}

// serveStreaming sets up SSE and streams events to the client.
func (h *Handler) serveStreaming(
	w http.ResponseWriter,
	ctx context.Context,
	wireModelID string,
	stream <-chan client.StreamResult,
) {
	// Set up SSE writer.
	sse, err := NewSSEWriter(w)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "streaming_error", err.Error())
		return
	}
	sse.SetHeaders()

	// Emit SSE events.
	emitter := NewEmitter(wireModelID)
	h.streamEvents(ctx, sse, emitter, stream)
}

// serveNonStreaming collects all stream events and returns a single JSON response.
func (h *Handler) serveNonStreaming(
	w http.ResponseWriter,
	ctx context.Context,
	wireModelID string,
	stream <-chan client.StreamResult,
) {
	collector := NewResponseCollector(wireModelID)

	for {
		select {
		case <-ctx.Done():
			go func() {
				for range stream {
				}
			}()
			WriteJSONError(w, http.StatusGatewayTimeout, "timeout", "Request cancelled")
			return

		case result, ok := <-stream:
			if !ok {
				// Stream complete — write the collected response.
				payload := collector.Finish()
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(payload); err != nil {
					h.logger.Debug("failed to write response", "err", err)
				}
				return
			}

			if result.Err != nil {
				collector.AddError(result.Err)
				continue
			}

			collector.Add(result.Event)
		}
	}
}

// streamEvents reads from the upstream channel and writes SSE events.
func (h *Handler) streamEvents(
	ctx interface{ Done() <-chan struct{} },
	sse *SSEWriter,
	emitter *Emitter,
	stream <-chan client.StreamResult,
) {
	for {
		select {
		case <-ctx.Done():
			// Client disconnected — drain and exit.
			go func() {
				for range stream {
				}
			}()
			return

		case result, ok := <-stream:
			if !ok {
				return // Stream complete.
			}

			if result.Err != nil {
				// Emit the error as a unified error event.
				errEv := unified.StreamEvent{
					Type:  unified.StreamEventError,
					Error: &unified.StreamError{Err: result.Err},
				}
				for _, ev := range emitter.Emit(errEv) {
					if err := validateAndLogSSEEvent(h.logger, ev); err != nil {
						h.logger.Error("invalid SSE event", "event_name", ev.Name, "err", err)
					}
					if writeErr := sse.WriteEvent(ev); writeErr != nil {
						h.logger.Debug("write error", "err", writeErr)
						return
					}
				}
				continue
			}

			for _, ev := range emitter.Emit(result.Event) {
				if err := validateAndLogSSEEvent(h.logger, ev); err != nil {
					h.logger.Error("invalid SSE event", "event_name", ev.Name, "err", err)
				}
				if writeErr := sse.WriteEvent(ev); writeErr != nil {
					h.logger.Debug("write error", "err", writeErr)
					// Client gone — drain upstream in background.
					go func() {
						for range stream {
						}
					}()
					return
				}
			}
		}
	}
}

func validateAndLogSSEEvent(logger *slog.Logger, ev SSEEvent) error {
	logSSEEvent(logger, ev)
	validator, err := defaultOpenAPIEventValidator()
	if err != nil {
		return fmt.Errorf("initialize OpenAPI event validator: %w", err)
	}
	if err := validator.ValidateSSEEvent(ev); err != nil {
		return fmt.Errorf("validate SSE event %q: %w", ev.Name, err)
	}
	return nil
}

func logJSONPayload(logger *slog.Logger, level slog.Level, msg, key string, data []byte) {
	if logger == nil {
		return
	}
	trimmed := trimJSONWhitespace(data)
	if len(trimmed) == 0 {
		logger.Log(context.Background(), level, msg, key, "")
		return
	}

	var payload any
	if err := json.Unmarshal(trimmed, &payload); err != nil {
		logger.Log(context.Background(), level, msg, key, string(data))
		return
	}
	logger.Log(context.Background(), level, msg, key, payload)
}

func logSSEEvent(logger *slog.Logger, ev SSEEvent) {
	if logger == nil {
		return
	}
	trimmed := trimJSONWhitespace(ev.Data)
	if len(trimmed) == 0 {
		logger.Debug("sending SSE event", "event_name", ev.Name)
		return
	}

	var payload any
	if err := json.Unmarshal(trimmed, &payload); err != nil {
		logger.Debug("sending SSE event", "event_name", ev.Name, "raw", string(ev.Data))
		return
	}
	logger.Debug("sending SSE event", "event_name", ev.Name, "payload", payload)
}

func trimJSONWhitespace(b []byte) []byte {
	start := 0
	for start < len(b) {
		switch b[start] {
		case ' ', '\t', '\n', '\r':
			start++
		default:
			goto endStart
		}
	}

endStart:
	end := len(b)
	for end > start {
		switch b[end-1] {
		case ' ', '\t', '\n', '\r':
			end--
		default:
			return b[start:end]
		}
	}
	return b[start:end]
}
