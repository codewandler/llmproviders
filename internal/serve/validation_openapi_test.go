package serve

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAPIValidator_ValidRequestPassesThrough(t *testing.T) {
	validator, err := NewOpenAPIValidator("/v1/responses")
	if err != nil {
		t.Fatalf("NewOpenAPIValidator: %v", err)
	}

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		body, err := readAllString(r)
		if err != nil {
			t.Fatalf("read body in next handler: %v", err)
		}
		if !strings.Contains(body, `"model":"gpt-5.4"`) {
			t.Fatalf("body not preserved for next handler: %s", body)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"gpt-5.4","input":"hello","stream":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	validator.Middleware(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if !called {
		t.Fatal("expected next handler to be called")
	}
}

func TestOpenAPIValidator_InvalidRequestRejected(t *testing.T) {
	validator, err := NewOpenAPIValidator("/v1/responses")
	if err != nil {
		t.Fatalf("NewOpenAPIValidator: %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"gpt-5.4","temperature":3,"input":"hello","stream":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	validator.Middleware(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"code":"invalid_request"`) {
		t.Fatalf("expected invalid_request code, got body=%s", rec.Body.String())
	}
}

func TestOpenAPIValidator_SkipsOtherPaths(t *testing.T) {
	validator, err := NewOpenAPIValidator("/v1/responses")
	if err != nil {
		t.Fatalf("NewOpenAPIValidator: %v", err)
	}

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	})

	req := httptest.NewRequest(http.MethodPost, "/health", strings.NewReader(`not json`))
	rec := httptest.NewRecorder()

	validator.Middleware(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	if !called {
		t.Fatal("expected next handler to be called")
	}
}

func readAllString(r *http.Request) (string, error) {
	if r.Body == nil {
		return "", nil
	}
	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
