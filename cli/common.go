package cli

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	llmproviders "github.com/codewandler/llmproviders"
)

// ServiceLoader is a function that loads a Service instance.
// This allows callers to inject their own Service configuration (e.g., custom overlays).
type ServiceLoader func(ctx context.Context) (*llmproviders.Service, error)

// ProviderInfo holds information about a registered provider.
type ProviderInfo struct {
	InstanceName  string
	ServiceID     string
	Order         int
	Detected      bool
	Aliases       map[string]string
	IntentAliases map[string]string
}

// IntentMapping represents an intent alias and its resolution.
type IntentMapping struct {
	Intent       string
	WireModelID  string
	InstanceName string
	ServiceID    string
}

// AliasMapping represents a provider alias and its resolution.
type AliasMapping struct {
	Alias       string
	WireModelID string
	ServiceID   string
	Shadowed    bool   // true if this alias is shadowed by a higher-priority provider
	ShadowedBy  string // the instance that shadows this alias
}

// formatTable formats data as an aligned table.
func formatTable(out io.Writer, headers []string, rows [][]string) {
	if len(rows) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	for i, h := range headers {
		if i > 0 {
			fmt.Fprint(out, "  ")
		}
		fmt.Fprintf(out, "%-*s", widths[i], h)
	}
	fmt.Fprintln(out)

	// Print separator
	for i, w := range widths {
		if i > 0 {
			fmt.Fprint(out, "  ")
		}
		fmt.Fprint(out, strings.Repeat("-", w))
	}
	fmt.Fprintln(out)

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				fmt.Fprint(out, "  ")
			}
			if i < len(widths) {
				fmt.Fprintf(out, "%-*s", widths[i], cell)
			} else {
				fmt.Fprint(out, cell)
			}
		}
		fmt.Fprintln(out)
	}
}

// sortedMapKeys returns the keys of a map sorted alphabetically.
func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// checkMark returns a check mark or X based on the boolean.
func checkMark(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// checkSymbol returns a symbol based on the boolean.
func checkSymbol(b bool) string {
	if b {
		return "[x]"
	}
	return "[ ]"
}
