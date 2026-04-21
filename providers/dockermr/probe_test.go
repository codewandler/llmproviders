package dockermr

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbeAvailable(t *testing.T) {
	t.Run("returns true on 200 engines endpoint", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/engines" {
				t.Fatalf("path = %q, want %q", r.URL.Path, "/engines")
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		t.Setenv(EnvDockerMRURL, ts.URL)

		if !ProbeAvailable(context.Background()) {
			t.Fatal("ProbeAvailable() = false, want true")
		}
	})

	t.Run("returns false on non-200", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
		defer ts.Close()

		t.Setenv(EnvDockerMRURL, ts.URL)

		if ProbeAvailable(context.Background()) {
			t.Fatal("ProbeAvailable() = true, want false")
		}
	})
}

func TestProviderFetchModels(t *testing.T) {
	t.Run("returns models and falls back name to id", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/engines/test-engine/v1/models" {
				t.Fatalf("path = %q, want %q", r.URL.Path, "/engines/test-engine/v1/models")
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"ai/smollm2","name":"SmolLM2"},{"id":"ai/qwen2.5","name":""}]}`))
		}))
		defer ts.Close()

		p, err := New(WithBaseURL(ts.URL), WithEngine("test-engine"), WithHTTPClient(ts.Client()))
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
		if models[0].ID != "ai/smollm2" || models[0].Name != "SmolLM2" {
			t.Fatalf("models[0] = %+v, want explicit name", models[0])
		}
		if models[1].ID != "ai/qwen2.5" || models[1].Name != "ai/qwen2.5" {
			t.Fatalf("models[1] = %+v, want fallback name to id", models[1])
		}
	})

	t.Run("returns status errors", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("upstream down"))
		}))
		defer ts.Close()

		p, err := New(WithBaseURL(ts.URL), WithEngine("test-engine"), WithHTTPClient(ts.Client()))
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		_, err = p.FetchModels(context.Background())
		if err == nil || fmt.Sprint(err) != "dockermr list models: status 502: upstream down" {
			t.Fatalf("FetchModels() error = %v, want status error", err)
		}
	})
}
