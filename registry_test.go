package llmproviders

import (
	"context"
	"os"
	"testing"

	"github.com/codewandler/agentapis/conversation"
	"github.com/codewandler/llmproviders/registry"
)

func TestRegistryDetect(t *testing.T) {
	ctx := context.Background()

	t.Run("no providers registered", func(t *testing.T) {
		reg := registry.New()
		detected, err := reg.Detect(ctx)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if len(detected) != 0 {
			t.Errorf("Detect() = %v, want empty", detected)
		}
	})

	t.Run("openai detected when env var set", func(t *testing.T) {
		reg := registry.New()
		reg.Register(openaiRegisterForTest)

		t.Setenv("OPENAI_API_KEY", "test-key")

		detected, err := reg.Detect(ctx)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if len(detected) == 0 {
			t.Fatal("Detect() returned empty, expected openai provider")
		}
		if detected[0].ServiceID != "openai" {
			t.Errorf("Detect()[0].ServiceID = %q, want openai", detected[0].ServiceID)
		}
	})

	t.Run("openai not detected when env var not set", func(t *testing.T) {
		reg := registry.New()
		reg.Register(openaiRegisterForTest)

		t.Setenv("OPENAI_API_KEY", "")
		t.Setenv("OPENAI_KEY", "")

		detected, err := reg.Detect(ctx)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if len(detected) != 0 {
			t.Errorf("Detect() = %v, want empty", detected)
		}
	})
}

var openaiRegisterForTest = registry.Registration{
	InstanceName: "openai",
	ServiceID:    "openai",
	Order:        40,
	Detect: func(ctx context.Context) (bool, error) {
		return os.Getenv("OPENAI_API_KEY") != "" || os.Getenv("OPENAI_KEY") != "", nil
	},
	Build: func(ctx context.Context, cfg registry.BuildConfig) (registry.Provider, error) {
		return &testProvider{name: "openai"}, nil
	},
}

type testProvider struct {
	name string
}

func (p *testProvider) Name() string {
	return p.name
}

func (p *testProvider) CreateSession(opts ...conversation.Option) *conversation.Session {
	return nil
}
