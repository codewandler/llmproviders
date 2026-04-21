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

// ProvidersCommandOptions configures the providers command.
type ProvidersCommandOptions struct {
	IO          IO
	LoadService ServiceLoader
	// LoadRegistry returns the registry for showing all providers (detected or not).
	// If nil, only detected providers from the Service are shown.
	LoadRegistry func(ctx context.Context) (*registry.Registry, error)
}

// NewProvidersCommand creates the "providers" command that lists providers.
func NewProvidersCommand(opts ProvidersCommandOptions) *cobra.Command {
	ioCfg := opts.IO.WithDefaults()

	var (
		detectedOnly bool
		jsonOut      bool
	)

	cmd := &cobra.Command{
		Use:   "providers",
		Short: "List registered providers and their detection status",
		Long: `Lists all registered LLM providers and their detection status.

Providers are detected based on environment variables or local availability.
The order value determines priority for intent alias resolution (lower = higher priority).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := opts.LoadService(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to load service: %w", err)
			}

			if jsonOut {
				return printProvidersJSON(ioCfg.Out, svc)
			}

			return printProviders(ioCfg.Out, svc, detectedOnly)
		},
	}

	cmd.SetOut(ioCfg.Out)
	cmd.SetErr(ioCfg.Err)
	cmd.Flags().BoolVarP(&detectedOnly, "detected", "d", false, "Show only detected providers")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")

	return cmd
}

func printProviders(w io.Writer, svc *llmproviders.Service, detectedOnly bool) error {
	instances := svc.RegisteredInstances()
	_ = svc.RegisteredServices()

	if len(instances) == 0 {
		fmt.Fprintln(w, "No providers detected.")
		return nil
	}

	fmt.Fprintln(w, "DETECTED PROVIDERS")
	fmt.Fprintln(w)

	headers := []string{"INSTANCE", "SERVICE", "STATUS"}
	var rows [][]string

	for _, inst := range instances {
		serviceID, ok := svc.ServiceIDForInstance(inst)
		if !ok {
			serviceID = "unknown"
		}
		rows = append(rows, []string{inst, serviceID, "detected"})
	}

	// Sort by instance name
	sort.Slice(rows, func(i, j int) bool {
		return rows[i][0] < rows[j][0]
	})

	formatTable(w, headers, rows)

	fmt.Fprintln(w)
	fmt.Fprintf(w, "%d providers detected\n", len(instances))

	return nil
}

type providersJSONOutput struct {
	Providers []providerJSONInfo `json:"providers"`
}

type providerJSONInfo struct {
	Instance string `json:"instance"`
	Service  string `json:"service"`
	Detected bool   `json:"detected"`
}

func printProvidersJSON(w io.Writer, svc *llmproviders.Service) error {
	instances := svc.RegisteredInstances()

	output := providersJSONOutput{
		Providers: make([]providerJSONInfo, 0, len(instances)),
	}

	for _, inst := range instances {
		serviceID, ok := svc.ServiceIDForInstance(inst)
		if !ok {
			serviceID = "unknown"
		}
		output.Providers = append(output.Providers, providerJSONInfo{
			Instance: inst,
			Service:  serviceID,
			Detected: true,
		})
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}
