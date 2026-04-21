// llmcli is a command-line tool for exploring LLM providers and models.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	llmproviders "github.com/codewandler/llmproviders"
	"github.com/codewandler/llmproviders/cli"
	"github.com/codewandler/llmproviders/registry/auto"
	"github.com/codewandler/modeldb"
	"github.com/spf13/cobra"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	rootCmd := &cobra.Command{
		Use:   "llmcli",
		Short: "LLM provider CLI tool",
		Long: `A CLI tool for exploring LLM providers and models.

Commands:
  intents    Show intent alias resolution (fast, default, powerful)
  providers  List registered providers
  aliases    Show provider-specific aliases
  models     List available models
  resolve    Explain how a model reference resolves
  catalog    Query the modeldb catalog directly

Use "llmcli <command> --help" for more information about a command.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Service loader used by all commands
	loadService := func(ctx context.Context) (*llmproviders.Service, error) {
		reg := auto.NewAutoDetectRegistry()
		return llmproviders.NewService(llmproviders.WithRegistry(reg))
	}

	// Catalog loader for catalog command
	loadCatalog := func(ctx context.Context) (modeldb.Catalog, error) {
		return modeldb.LoadBuiltIn()
	}

	// Add commands
	rootCmd.AddCommand(cli.NewIntentsCommand(cli.IntentsCommandOptions{
		LoadService: loadService,
	}))
	rootCmd.AddCommand(cli.NewProvidersCommand(cli.ProvidersCommandOptions{
		LoadService: loadService,
	}))
	rootCmd.AddCommand(cli.NewAliasesCommand(cli.AliasesCommandOptions{
		LoadService: loadService,
	}))
	rootCmd.AddCommand(cli.NewModelsCommand(cli.ModelsCommandOptions{
		LoadService: loadService,
	}))
	rootCmd.AddCommand(cli.NewResolveCommand(cli.ResolveCommandOptions{
		LoadService: loadService,
	}))
	rootCmd.AddCommand(cli.NewCatalogCommand(cli.CatalogCommandOptions{
		LoadCatalog: loadCatalog,
	}))

	return rootCmd.ExecuteContext(ctx)
}
