package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	llmproviders "github.com/codewandler/llmproviders"
	"github.com/spf13/cobra"
)

// AliasesCommandOptions configures the aliases command.
type AliasesCommandOptions struct {
	IO          IO
	LoadService ServiceLoader
}

// NewAliasesCommand creates the "aliases" command that shows provider aliases.
func NewAliasesCommand(opts AliasesCommandOptions) *cobra.Command {
	ioCfg := opts.IO.WithDefaults()

	var (
		byProvider bool
		jsonOut    bool
	)

	cmd := &cobra.Command{
		Use:   "aliases",
		Short: "Show provider-specific aliases (sonnet, opus, mini, etc.)",
		Long: `Shows provider-specific model aliases.

Aliases are short names registered by each provider:
  - sonnet  -> claude-sonnet-4-6 (anthropic)
  - opus    -> claude-opus-4-6 (anthropic)
  - mini    -> gpt-5.4-mini (openai)
  - nano    -> gpt-5.4-nano (openai)

When multiple providers register the same alias, the highest-priority
provider (lowest order) wins. Shadowed aliases are shown with --by-provider.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := opts.LoadService(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to load service: %w", err)
			}

			if jsonOut {
				return printAliasesJSON(ioCfg.Out, svc)
			}

			if byProvider {
				return printAliasesByProvider(ioCfg.Out, svc)
			}

			return printAliasesMerged(ioCfg.Out, svc)
		},
	}

	cmd.SetOut(ioCfg.Out)
	cmd.SetErr(ioCfg.Err)
	cmd.Flags().BoolVarP(&byProvider, "by-provider", "p", false, "Show aliases grouped by provider")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")

	return cmd
}

func printAliasesMerged(w io.Writer, svc *llmproviders.Service) error {
	aliases := svc.ProviderAliases()

	if len(aliases) == 0 {
		fmt.Fprintln(w, "No aliases configured.")
		return nil
	}

	fmt.Fprintln(w, "MERGED ALIASES (first detected provider wins)")
	fmt.Fprintln(w)

	headers := []string{"ALIAS", "MODEL", "SERVICE"}
	var rows [][]string

	for _, alias := range sortedMapKeys(aliases) {
		target := aliases[alias]
		rows = append(rows, []string{alias, target.WireModelID, target.ServiceID})
	}

	formatTable(w, headers, rows)

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Use `llmcli aliases --by-provider` to see per-provider breakdown.")

	return nil
}

func printAliasesByProvider(w io.Writer, svc *llmproviders.Service) error {
	aliases := svc.ProviderAliases()

	if len(aliases) == 0 {
		fmt.Fprintln(w, "No aliases configured.")
		return nil
	}

	fmt.Fprintln(w, "ALIASES BY PROVIDER")
	fmt.Fprintln(w)

	// Group by service
	byService := make(map[string][]struct {
		alias string
		model string
	})

	for alias, target := range aliases {
		byService[target.ServiceID] = append(byService[target.ServiceID], struct {
			alias string
			model string
		}{alias: alias, model: target.WireModelID})
	}

	// Sort services
	services := sortedMapKeys(byService)

	for _, svcID := range services {
		items := byService[svcID]
		// Sort aliases within service
		sort.Slice(items, func(i, j int) bool {
			return items[i].alias < items[j].alias
		})

		fmt.Fprintf(w, "%s (detected)\n", svcID)
		for _, item := range items {
			fmt.Fprintf(w, "  %-12s -> %s\n", item.alias, item.model)
		}
		fmt.Fprintln(w)
	}

	return nil
}

type aliasesJSONOutput struct {
	Aliases map[string]aliasJSONTarget `json:"aliases"`
}

type aliasJSONTarget struct {
	Model   string `json:"model"`
	Service string `json:"service"`
}

func printAliasesJSON(w io.Writer, svc *llmproviders.Service) error {
	aliases := svc.ProviderAliases()

	output := aliasesJSONOutput{
		Aliases: make(map[string]aliasJSONTarget),
	}

	for alias, target := range aliases {
		output.Aliases[alias] = aliasJSONTarget{
			Model:   target.WireModelID,
			Service: target.ServiceID,
		}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}
