package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	llmproviders "github.com/codewandler/llmproviders"
	"github.com/codewandler/llmproviders/registry"
	"github.com/spf13/cobra"
)

// IntentsCommandOptions configures the intents command.
type IntentsCommandOptions struct {
	IO          IO
	LoadService ServiceLoader
	// LoadRegistry returns the registry for showing all providers (detected or not).
	// If nil, only detected providers from the Service are shown.
	LoadRegistry func(ctx context.Context) (*registry.Registry, error)
}

// NewIntentsCommand creates the "intents" command that shows intent alias resolution.
func NewIntentsCommand(opts IntentsCommandOptions) *cobra.Command {
	ioCfg := opts.IO.WithDefaults()

	var (
		showAll bool
		jsonOut bool
	)

	cmd := &cobra.Command{
		Use:   "intents",
		Short: "Show intent alias resolution (fast, default, powerful)",
		Long: `Shows how intent aliases resolve to specific models.

Intent aliases provide semantic shortcuts for model selection:
  - fast:     Fastest available model (e.g., Haiku, GPT-4o-mini)
  - default:  Balanced model for general use (e.g., Sonnet, GPT-4o)
  - powerful: Most capable model (e.g., Opus, o3)

Resolution is based on provider priority (lower order = higher priority).
The first detected provider wins for each intent.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := opts.LoadService(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to load service: %w", err)
			}

			if jsonOut {
				return printIntentsJSON(ioCfg.Out, svc, showAll, opts.LoadRegistry, cmd.Context())
			}

			if showAll {
				return printIntentsAll(ioCfg.Out, svc, opts.LoadRegistry, cmd.Context())
			}

			return printIntentsActive(ioCfg.Out, svc)
		},
	}

	cmd.SetOut(ioCfg.Out)
	cmd.SetErr(ioCfg.Err)
	cmd.Flags().BoolVarP(&showAll, "all", "a", false, "Show intent mappings for all providers")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")

	return cmd
}

func printIntentsActive(w io.Writer, svc *llmproviders.Service) error {

	aliases := svc.IntentAliases()
	if len(aliases) == 0 {
		fmt.Fprintln(w, "No intent aliases configured.")
		return nil
	}

	fmt.Fprintln(w, "ACTIVE INTENT ALIASES (based on detected providers)")
	fmt.Fprintln(w)

	// Find the provider with highest priority (for the note)
	var primaryProvider string
	for _, intent := range llmproviders.AllIntents() {
		if ref, ok := aliases[intent]; ok {
			primaryProvider = ref.InstanceName
			break
		}
	}

	headers := []string{"INTENT", "MODEL", "PROVIDER"}
	var rows [][]string

	for _, intent := range llmproviders.AllIntents() {
		if ref, ok := aliases[intent]; ok {
			rows = append(rows, []string{intent, ref.WireModelID, ref.InstanceName})
		}
	}

	formatTable(w, headers, rows)

	if primaryProvider != "" {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Note: %s has highest priority among detected providers.\n", primaryProvider)
	}

	return nil
}

func printIntentsAll(w io.Writer, svc *llmproviders.Service, loadRegistry func(context.Context) (*registry.Registry, error), ctx context.Context) error {

	fmt.Fprintln(w, "INTENT ALIAS MAPPINGS BY PROVIDER")
	fmt.Fprintln(w)

	// Get all instances from the service (detected providers)
	instances := svc.RegisteredInstances()
	detectedSet := make(map[string]bool)
	for _, inst := range instances {
		detectedSet[inst] = true
	}

	// Get active intent aliases to show which provider wins
	activeAliases := svc.IntentAliases()

	// Build provider info - we need to access the registry for full info
	// For now, show what we have from the Service
	headers := []string{"PROVIDER", "ORDER", "DETECTED", "FAST", "DEFAULT", "POWERFUL"}
	var rows [][]string

	// Get intent aliases per instance from the service
	// Note: The Service doesn't expose per-instance intent aliases directly,
	// so we show the merged view with annotations

	type providerRow struct {
		instance string
		order    int
		detected bool
		fast     string
		def      string
		powerful string
	}

	// We need to get this info from somewhere - for now, show the active mappings
	// and note that --all requires registry access for full breakdown

	for _, intent := range llmproviders.AllIntents() {
		if ref, ok := activeAliases[intent]; ok {
			// Find or create row for this provider
			found := false
			for i, row := range rows {
				if row[0] == ref.InstanceName {
					found = true
					switch intent {
					case llmproviders.IntentFast:
						rows[i][3] = ref.WireModelID
					case llmproviders.IntentDefault:
						rows[i][4] = ref.WireModelID
					case llmproviders.IntentPowerful:
						rows[i][5] = ref.WireModelID
					}
					break
				}
			}
			if !found {
				row := []string{ref.InstanceName, "-", "yes", "", "", ""}
				switch intent {
				case llmproviders.IntentFast:
					row[3] = ref.WireModelID
				case llmproviders.IntentDefault:
					row[4] = ref.WireModelID
				case llmproviders.IntentPowerful:
					row[5] = ref.WireModelID
				}
				rows = append(rows, row)
			}
		}
	}

	if len(rows) == 0 {
		fmt.Fprintln(w, "No providers detected.")
		return nil
	}

	formatTable(w, headers, rows)

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Note: Only the highest-priority provider's intents are active.")
	fmt.Fprintln(w, "      Use `llmcli providers` to see all registered providers and their order.")

	return nil
}

type intentsJSONOutput struct {
	Active     map[string]intentsJSONRef `json:"active"`
	ByProvider map[string]intentsJSONRef `json:"by_provider,omitempty"`
}

type intentsJSONRef struct {
	Model    string `json:"model"`
	Provider string `json:"provider"`
	Service  string `json:"service"`
}

func printIntentsJSON(w io.Writer, svc *llmproviders.Service, showAll bool, loadRegistry func(context.Context) (*registry.Registry, error), ctx context.Context) error {

	aliases := svc.IntentAliases()

	output := intentsJSONOutput{
		Active: make(map[string]intentsJSONRef),
	}

	for intent, ref := range aliases {
		output.Active[intent] = intentsJSONRef{
			Model:    ref.WireModelID,
			Provider: ref.InstanceName,
			Service:  ref.ServiceID,
		}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

// GetProviderIntentMappings returns intent mappings grouped by provider.
// This is useful for CLI and programmatic access.
func GetProviderIntentMappings(svc *llmproviders.Service) []IntentMapping {
	aliases := svc.IntentAliases()
	var mappings []IntentMapping

	for _, intent := range llmproviders.AllIntents() {
		if ref, ok := aliases[intent]; ok {
			mappings = append(mappings, IntentMapping{
				Intent:       intent,
				WireModelID:  ref.WireModelID,
				InstanceName: ref.InstanceName,
				ServiceID:    ref.ServiceID,
			})
		}
	}

	sort.Slice(mappings, func(i, j int) bool {
		return mappings[i].Intent < mappings[j].Intent
	})

	return mappings
}
