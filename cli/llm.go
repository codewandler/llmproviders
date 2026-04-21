package cli

import "github.com/spf13/cobra"

// LLMCommandOptions configures the LLM command.
type LLMCommandOptions struct {
	// LoadService is required - loads the provider service.
	LoadService ServiceLoader

	// LoadCatalog is optional - if provided, adds the catalog subcommand.
	LoadCatalog CatalogLoader

	// IO configures input/output streams. Uses defaults if zero.
	IO IO

	// Use overrides the command name. Defaults to "llm".
	Use string

	// Short overrides the short description.
	Short string
}

// NewLLMCommand creates a command with all LLM-related subcommands.
// Can be used as a root command (llmcli) or as a subcommand (miniagent llm).
func NewLLMCommand(opts LLMCommandOptions) *cobra.Command {
	use := opts.Use
	if use == "" {
		use = "llm"
	}
	short := opts.Short
	if short == "" {
		short = "LLM provider commands"
	}

	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long: `LLM provider CLI for inference, model discovery, and provider management.

Commands:
  infer      Send a message to an LLM and stream the response
  models     List available models from detected providers
  intents    Show intent alias resolution (fast, default, powerful)
  providers  List registered providers and their detection status
  aliases    Show provider-specific aliases (sonnet, opus, mini, etc.)
  resolve    Explain how a model reference resolves
  skill      Show or install the llmcli skill`,
	}

	// Add subcommands
	cmd.AddCommand(NewInferCommand(InferCommandOptions{
		IO:          opts.IO,
		LoadService: opts.LoadService,
	}))
	cmd.AddCommand(NewModelsCommand(ModelsCommandOptions{
		IO:          opts.IO,
		LoadService: opts.LoadService,
	}))
	cmd.AddCommand(NewIntentsCommand(IntentsCommandOptions{
		IO:          opts.IO,
		LoadService: opts.LoadService,
	}))
	cmd.AddCommand(NewProvidersCommand(ProvidersCommandOptions{
		IO:          opts.IO,
		LoadService: opts.LoadService,
	}))
	cmd.AddCommand(NewAliasesCommand(AliasesCommandOptions{
		IO:          opts.IO,
		LoadService: opts.LoadService,
	}))
	cmd.AddCommand(NewResolveCommand(ResolveCommandOptions{
		IO:          opts.IO,
		LoadService: opts.LoadService,
	}))
	cmd.AddCommand(NewSkillCommand(SkillCommandOptions{
		IO: opts.IO,
	}))

	// Catalog is optional - only add if LoadCatalog is provided
	if opts.LoadCatalog != nil {
		cmd.AddCommand(NewCatalogCommand(CatalogCommandOptions{
			IO:          opts.IO,
			LoadCatalog: opts.LoadCatalog,
		}))
	}

	return cmd
}
