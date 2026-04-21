package serve

import (
	"fmt"
	"net/http"
)

// SSEWriter writes Server-Sent Events to an http.ResponseWriter.
type SSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// NewSSEWriter creates an SSEWriter, returning an error if the ResponseWriter
// does not support flushing.
func NewSSEWriter(w http.ResponseWriter) (*SSEWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}
	return &SSEWriter{w: w, flusher: flusher}, nil
}

// SetHeaders writes the SSE response headers. Must be called before WriteEvent.
func (s *SSEWriter) SetHeaders() {
	s.w.Header().Set("Content-Type", "text/event-stream")
	s.w.Header().Set("Cache-Control", "no-cache")
	s.w.Header().Set("Connection", "keep-alive")
	s.w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering
}

// SSEEvent is one Server-Sent Event ready to write.
type SSEEvent struct {
	Name string // SSE event name, e.g. "response.output_text.delta"
	Data []byte // JSON payload
}

// WriteEvent writes a single SSE event and flushes.
func (s *SSEWriter) WriteEvent(ev SSEEvent) error {
	if ev.Name != "" {
		if _, err := fmt.Fprintf(s.w, "event: %s\n", ev.Name); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", ev.Data); err != nil {
		return err
	}
	s.flusher.Flush()
	return nil
}
