// llmcli is a command-line tool for exploring LLM providers and models.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	llmproviders "github.com/codewandler/llmproviders"
	"github.com/codewandler/llmproviders/cli"
	"github.com/codewandler/llmproviders/registry/auto"
	"github.com/codewandler/modeldb"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	loadService := func(ctx context.Context) (*llmproviders.Service, error) {
		reg := auto.NewAutoDetectRegistry()
		return llmproviders.NewService(llmproviders.WithRegistry(reg))
	}

	loadCatalog := func(ctx context.Context) (modeldb.Catalog, error) {
		return modeldb.LoadBuiltIn()
	}

	rootCmd := cli.NewLLMCommand(cli.LLMCommandOptions{
		Use:         "llmcli",
		Short:       "LLM provider CLI tool",
		LoadService: loadService,
		LoadCatalog: loadCatalog,
	})
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	return rootCmd.ExecuteContext(ctx)
}
