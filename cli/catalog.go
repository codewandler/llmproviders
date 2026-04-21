package cli

import (
	"context"

	"github.com/codewandler/modeldb"
	modeldbcli "github.com/codewandler/modeldb/cli"
	"github.com/spf13/cobra"
)

// CatalogCommandOptions configures the catalog command.
type CatalogCommandOptions struct {
	IO IO
	// LoadCatalog returns the modeldb catalog.
	// If nil, uses modeldb.LoadBuiltIn().
	LoadCatalog func(ctx context.Context) (modeldb.Catalog, error)
}

// NewCatalogCommand creates the "catalog" command that wraps the modeldb CLI.
// This provides access to the full modeldb query capabilities.
func NewCatalogCommand(opts CatalogCommandOptions) *cobra.Command {
	ioCfg := opts.IO.WithDefaults()

	loadCatalog := opts.LoadCatalog
	if loadCatalog == nil {
		loadCatalog = func(ctx context.Context) (modeldb.Catalog, error) {
			return modeldb.LoadBuiltIn()
		}
	}

	// Wrap the modeldb models command
	modelsCmd := modeldbcli.NewModelsCommand(modeldbcli.ModelsCommandOptions{
		IO: modeldbcli.IO{
			In:  ioCfg.In,
			Out: ioCfg.Out,
			Err: ioCfg.Err,
		},
		LoadBaseCatalog: loadCatalog,
	})

	// Create a parent command for catalog operations
	cmd := &cobra.Command{
		Use:   "catalog",
		Short: "Query the modeldb catalog directly",
		Long: `Provides direct access to modeldb CLI features for advanced queries.

This command wraps the modeldb CLI, giving you access to:
  - Full model catalog queries with --id, --name, --creator filters
  - Service and API type filtering
  - Detailed model information with --details
  - JSON output for scripting

Examples:
  llmcli catalog --service openai --details
  llmcli catalog --query "claude" --offerings
  llmcli catalog --id "claude-sonnet-4-6" --json`,
	}

	// The modeldb command is the default action
	cmd.RunE = modelsCmd.RunE
	cmd.Flags().AddFlagSet(modelsCmd.Flags())

	cmd.SetOut(ioCfg.Out)
	cmd.SetErr(ioCfg.Err)

	// Register service flag completion
	_ = cmd.RegisterFlagCompletionFunc("service", completeCatalogServices(loadCatalog))

	return cmd
}
