package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	llmproviders "github.com/codewandler/llmproviders"
	"github.com/spf13/cobra"
)

// ModelsCommandOptions configures the models command.
type ModelsCommandOptions struct {
	IO          IO
	LoadService ServiceLoader
}

// NewModelsCommand creates the "models" command that lists available models.
func NewModelsCommand(opts ModelsCommandOptions) *cobra.Command {
	ioCfg := opts.IO.WithDefaults()

	var (
		serviceFilter string
		query         string
		showIntents   bool
	)

	cmd := &cobra.Command{
		Use:   "models",
		Short: "List available models from detected providers",
		Long: `Lists models available from detected providers.

Models are prefixed with their service ID (e.g., "anthropic/claude-sonnet-4-6").
Use --service to filter by a specific provider, --query to search.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := opts.LoadService(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to load service: %w", err)
			}

			return printModels(ioCfg.Out, svc, serviceFilter, query, showIntents)
		},
	}

	cmd.SetOut(ioCfg.Out)
	cmd.SetErr(ioCfg.Err)
	cmd.Flags().StringVarP(&serviceFilter, "service", "s", "", "Filter by service (e.g., anthropic, openai)")
	cmd.Flags().StringVarP(&query, "query", "q", "", "Search models by substring")
	cmd.Flags().BoolVar(&showIntents, "show-intents", false, "Mark models that are intent alias targets")

	// Register completions
	_ = cmd.RegisterFlagCompletionFunc("service", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		svc, err := opts.LoadService(cmd.Context())
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return svc.RegisteredServices(), cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func printModels(w io.Writer, svc *llmproviders.Service, serviceFilter, query string, showIntents bool) error {
	models := svc.Models(serviceFilter)

	if len(models) == 0 {
		if serviceFilter != "" {
			fmt.Fprintf(w, "No models found for service %q.\n", serviceFilter)
		} else {
			fmt.Fprintln(w, "No models available.")
		}
		return nil
	}

	// Filter by query
	if query != "" {
		query = strings.ToLower(query)
		filtered := make([]string, 0, len(models))
		for _, m := range models {
			if strings.Contains(strings.ToLower(m), query) {
				filtered = append(filtered, m)
			}
		}
		models = filtered
	}

	if len(models) == 0 {
		fmt.Fprintf(w, "No models found matching %q.\n", query)
		return nil
	}

	// Build intent markers if requested
	intentMarkers := make(map[string]string)
	if showIntents {
		intentAliases := svc.IntentAliases()
		for intent, ref := range intentAliases {
			// The model in Models() is prefixed with serviceID
			prefixed := ref.ServiceID + "/" + ref.WireModelID
			if existing, ok := intentMarkers[prefixed]; ok {
				intentMarkers[prefixed] = existing + ", " + intent
			} else {
				intentMarkers[prefixed] = intent
			}
		}
	}

	// Group by service for better display
	byService := make(map[string][]string)
	for _, m := range models {
		parts := strings.SplitN(m, "/", 2)
		if len(parts) == 2 {
			byService[parts[0]] = append(byService[parts[0]], m)
		} else {
			byService[""] = append(byService[""], m)
		}
	}

	title := "AVAILABLE MODELS"
	if serviceFilter != "" {
		title = fmt.Sprintf("MODELS FOR %s", strings.ToUpper(serviceFilter))
	}
	fmt.Fprintln(w, title)
	fmt.Fprintln(w)

	services := sortedMapKeys(byService)
	for _, svcID := range services {
		svcModels := byService[svcID]
		sort.Strings(svcModels)

		for _, m := range svcModels {
			if showIntents {
				if marker, ok := intentMarkers[m]; ok {
					fmt.Fprintf(w, "  %s  [%s]\n", m, marker)
					continue
				}
			}
			fmt.Fprintf(w, "  %s\n", m)
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "%d models available\n", len(models))

	if showIntents {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Models marked with [intent] are targets for intent aliases (fast, default, powerful).")
	}

	return nil
}
