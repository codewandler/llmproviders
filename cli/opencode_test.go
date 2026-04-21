package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOpencodeConfigureCreatesConfig(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	ioCfg := IO{Out: &out, Err: &out}

	svc := testService(t)

	err := configureOpencode(ioCfg, svc, ":8080", dir, "")
	if err != nil {
		t.Fatalf("configureOpencode: %v", err)
	}

	cfgPath := filepath.Join(dir, "opencode.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse config: %v", err)
	}

	providers, ok := cfg["provider"].(map[string]any)
	if !ok {
		t.Fatal("missing provider key")
	}
	entry, ok := providers["llmproviders"].(map[string]any)
	if !ok {
		t.Fatal("missing llmproviders provider")
	}

	opts, _ := entry["options"].(map[string]any)
	if got := opts["baseURL"]; got != "http://localhost:8080/v1" {
		t.Errorf("baseURL = %v, want http://localhost:8080/v1", got)
	}
	if got := opts["apiKey"]; got != "unused" {
		t.Errorf("apiKey = %v, want unused", got)
	}

	models, ok := entry["models"].(map[string]any)
	if !ok || len(models) == 0 {
		t.Fatal("expected models to be populated from aliases")
	}
}

func TestOpencodeConfigurePreservesExisting(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "opencode.json")

	existing := map[string]any{
		"$schema": "https://opencode.ai/config.json",
		"plugin":  []any{"@different-ai/opencode-browser"},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	if err := os.WriteFile(cfgPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	ioCfg := IO{Out: &out, Err: &out}

	svc := testService(t)

	if err := configureOpencode(ioCfg, svc, ":3000", dir, "llmproviders/sonnet"); err != nil {
		t.Fatalf("configureOpencode: %v", err)
	}

	data, _ = os.ReadFile(cfgPath)
	var cfg map[string]any
	json.Unmarshal(data, &cfg)

	// Plugin preserved.
	if _, ok := cfg["plugin"]; !ok {
		t.Error("existing plugin key was lost")
	}

	// Model set.
	if got := cfg["model"]; got != "llmproviders/sonnet" {
		t.Errorf("model = %v, want llmproviders/sonnet", got)
	}

	// Custom port.
	providers := cfg["provider"].(map[string]any)
	entry := providers["llmproviders"].(map[string]any)
	opts := entry["options"].(map[string]any)
	if got := opts["baseURL"]; got != "http://localhost:3000/v1" {
		t.Errorf("baseURL = %v, want http://localhost:3000/v1", got)
	}
}

func TestOpencodeConfigureRemove(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "opencode.json")

	existing := map[string]any{
		"$schema": "https://opencode.ai/config.json",
		"model":   "llmproviders/sonnet",
		"provider": map[string]any{
			"llmproviders": map[string]any{
				"npm": "@ai-sdk/openai-compatible",
			},
			"other": map[string]any{
				"npm": "something-else",
			},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	os.WriteFile(cfgPath, data, 0644)

	var out bytes.Buffer
	ioCfg := IO{Out: &out, Err: &out}

	if err := removeOpencodeProvider(ioCfg, dir); err != nil {
		t.Fatalf("removeOpencodeProvider: %v", err)
	}

	data, _ = os.ReadFile(cfgPath)
	var cfg map[string]any
	json.Unmarshal(data, &cfg)

	providers := cfg["provider"].(map[string]any)
	if _, exists := providers["llmproviders"]; exists {
		t.Error("llmproviders provider should have been removed")
	}
	if _, exists := providers["other"]; !exists {
		t.Error("other provider should be preserved")
	}
	if _, exists := cfg["model"]; exists {
		t.Error("model pointing to llmproviders should have been removed")
	}
}

func TestOpencodeConfigureRemoveNoop(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "opencode.json")

	existing := map[string]any{"$schema": "https://opencode.ai/config.json"}
	data, _ := json.MarshalIndent(existing, "", "  ")
	os.WriteFile(cfgPath, data, 0644)

	var out bytes.Buffer
	ioCfg := IO{Out: &out, Err: &out}

	if err := removeOpencodeProvider(ioCfg, dir); err != nil {
		t.Fatalf("removeOpencodeProvider: %v", err)
	}

	if !bytes.Contains(out.Bytes(), []byte("nothing to remove")) {
		t.Errorf("expected noop message, got: %s", out.String())
	}
}

func TestSplitHostPort(t *testing.T) {
	tests := []struct {
		addr     string
		wantHost string
		wantPort string
		wantOK   bool
	}{
		{":8080", "", "8080", true},
		{"0.0.0.0:3000", "0.0.0.0", "3000", true},
		{"localhost:9090", "localhost", "9090", true},
		{"noport", "", "", false},
	}
	for _, tt := range tests {
		h, p, ok := splitHostPort(tt.addr)
		if h != tt.wantHost || p != tt.wantPort || ok != tt.wantOK {
			t.Errorf("splitHostPort(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tt.addr, h, p, ok, tt.wantHost, tt.wantPort, tt.wantOK)
		}
	}
}
