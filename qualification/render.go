package qualification

import (
	"encoding/json"
	"fmt"
	"strings"
)

func RenderJSON(entries []Entry) ([]byte, error) {
	return json.MarshalIndent(entries, "", "  ")
}

func RenderMarkdown(entries []Entry) string {
	var b strings.Builder
	b.WriteString("# Codex qualification matrix\n\n")
	b.WriteString("| Provider | Service | Model | Status | Tools | Caching | Pricing | Thinking | Conversation | Coding Loop | Usage | Notes |\n")
	b.WriteString("|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---|\n")
	for _, e := range entries {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %t | %t | %t | %t | %t | %t | %t | %s |\n",
			e.Provider, e.Service, e.Model, e.Status,
			e.Dimensions.Tools, e.Dimensions.Caching, e.Dimensions.Pricing, e.Dimensions.Thinking,
			e.Dimensions.Conversation, e.Dimensions.CodingLoop, e.Dimensions.Usage,
			strings.Join(e.Notes, "; "))
	}
	return b.String()
}
