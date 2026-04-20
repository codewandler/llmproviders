package qualification

import "testing"

func TestCodexEntries(t *testing.T) {
	entries := CodexEntries()
	if len(entries) == 0 {
		t.Fatal("expected codex qualification entries")
	}
}

func TestRenderMarkdown(t *testing.T) {
	md := RenderMarkdown(CodexEntries())
	if md == "" {
		t.Fatal("expected markdown output")
	}
}
