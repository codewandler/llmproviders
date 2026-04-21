package ollama

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbeAvailable(t *testing.T) {
	t.Run("returns true on 200 tags endpoint", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/tags" {
				t.Fatalf("path = %q, want %q", r.URL.Path, "/api/tags")
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"models":[]}`))
		}))
		defer ts.Close()

		t.Setenv(EnvOllamaURL, ts.URL)

		if !ProbeAvailable(context.Background()) {
			t.Fatal("ProbeAvailable() = false, want true")
		}
	})

	t.Run("returns false on non-200", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer ts.Close()

		t.Setenv(EnvOllamaURL, ts.URL)

		if ProbeAvailable(context.Background()) {
			t.Fatal("ProbeAvailable() = true, want false")
		}
	})
}

func TestProviderFetchModels(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/api/tags")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"name":"qwen2.5:0.5b","model":"qwen2.5:0.5b","size":123},{"name":"","model":"llama3.2:1b","size":456}]}`))
	}))
	defer ts.Close()

	p, err := New(WithBaseURL(ts.URL), WithHTTPClient(ts.Client()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	models, err := p.FetchModels(context.Background())
	if err != nil {
		t.Fatalf("FetchModels() error = %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("len(models) = %d, want 2", len(models))
	}
	if models[0].ID != "qwen2.5:0.5b" || models[0].Name != "qwen2.5:0.5b" || models[0].Size != 123 {
		t.Fatalf("models[0] = %+v, want ID/Name qwen2.5:0.5b and Size 123", models[0])
	}
	if models[1].ID != "" || models[1].Name != "llama3.2:1b" || models[1].Size != 456 {
		t.Fatalf("models[1] = %+v, want ID empty, fallback name llama3.2:1b and Size 456", models[1])
	}
}

func TestDownloadModel(t *testing.T) {
	t.Run("returns nil when pull stream reports success", func(t *testing.T) {
		var sawPull bool
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/tags":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"models":[]}`))
			case "/api/pull":
				sawPull = true
				w.Header().Set("Content-Type", "application/x-ndjson")
				_, _ = w.Write([]byte("{\"status\":\"pulling\"}\n{\"status\":\"success\"}\n"))
			default:
				t.Fatalf("unexpected path %q", r.URL.Path)
			}
		}))
		defer ts.Close()

		p, err := New(WithBaseURL(ts.URL), WithHTTPClient(ts.Client()))
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		if err := p.Download(context.Background(), "qwen2.5:0.5b"); err != nil {
			t.Fatalf("Download() error = %v", err)
		}
		if !sawPull {
			t.Fatal("expected pull endpoint to be called")
		}
	})

	t.Run("surfaces pull stream error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/tags":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"models":[]}`))
			case "/api/pull":
				w.Header().Set("Content-Type", "application/x-ndjson")
				_, _ = w.Write([]byte("{\"error\":\"disk full\"}\n"))
			default:
				t.Fatalf("unexpected path %q", r.URL.Path)
			}
		}))
		defer ts.Close()

		p, err := New(WithBaseURL(ts.URL), WithHTTPClient(ts.Client()))
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		err = p.Download(context.Background(), "qwen2.5:0.5b")
		if err == nil || fmt.Sprint(err) != "download qwen2.5:0.5b: pull error: disk full" {
			t.Fatalf("Download() error = %v, want pull error", err)
		}
	})
}
