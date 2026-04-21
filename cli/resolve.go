package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	llmproviders "github.com/codewandler/llmproviders"
	"github.com/spf13/cobra"
)

// ResolveCommandOptions configures the resolve command.
type ResolveCommandOptions struct {
	IO          IO
	LoadService ServiceLoader
}

// NewResolveCommand creates the "resolve" command that explains model resolution.
func NewResolveCommand(opts ResolveCommandOptions) *cobra.Command {
	ioCfg := opts.IO.WithDefaults()

	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "resolve <model-ref>",
		Short: "Explain how a model reference resolves",
		Long: `Resolves a model reference and shows the resolution path.

This command is useful for debugging routing issues, like understanding
why "anthropic/claude-3-5-haiku" routes to OpenRouter instead of native Anthropic.

Examples:
  llmcli resolve fast
  llmcli resolve sonnet
  llmcli resolve anthropic/claude-3-5-haiku
  llmcli resolve anthropic/claude-sonnet-4-6`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := opts.LoadService(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to load service: %w", err)
			}

			modelRef := args[0]

			if jsonOut {
				return printResolveJSON(ioCfg.Out, svc, modelRef)
			}

			return printResolve(ioCfg.Out, svc, modelRef)
		},
	}

	cmd.SetOut(ioCfg.Out)
	cmd.SetErr(ioCfg.Err)
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")

	// Register positional argument completion
	cmd.ValidArgsFunction = completeModelRefs(opts.LoadService)

	return cmd
}

// ResolutionResult holds the result of resolving a model reference.
type ResolutionResult struct {
	Input        string
	Type         string // "intent_alias", "provider_alias", "catalog_wire_model", "service_model", "bare_model"
	WireModelID  string
	ProviderName string
	ServiceID    string
	Steps        []ResolutionStep
	Error        string
}

// ResolutionStep describes one step in the resolution process.
type ResolutionStep struct {
	Step    int
	Check   string
	Matched bool
	Detail  string
}

func printResolve(w io.Writer, svc *llmproviders.Service, modelRef string) error {
	result := resolveWithDetails(svc, modelRef)

	fmt.Fprintf(w, "Resolving: %s\n", result.Input)
	fmt.Fprintln(w)

	if result.Error != "" {
		fmt.Fprintf(w, "  Error: %s\n", result.Error)
		return nil
	}

	fmt.Fprintf(w, "  Type:       %s\n", formatResolutionType(result.Type))
	fmt.Fprintf(w, "  Resolved:   %s\n", result.WireModelID)
	fmt.Fprintf(w, "  Provider:   %s\n", result.ProviderName)
	fmt.Fprintf(w, "  Service:    %s\n", result.ServiceID)
	fmt.Fprintf(w, "  Wire model: %s\n", result.WireModelID)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "  Resolution path:")
	for _, step := range result.Steps {
		marker := "x"
		if step.Matched {
			marker = "v"
		}
		fmt.Fprintf(w, "    %d. [%s] %s", step.Step, marker, step.Check)
		if step.Detail != "" {
			fmt.Fprintf(w, " -> %s", step.Detail)
		}
		fmt.Fprintln(w)
	}

	return nil
}

func formatResolutionType(t string) string {
	switch t {
	case "intent_alias":
		return "Intent alias"
	case "provider_alias":
		return "Provider alias"
	case "catalog_wire_model":
		return "Wire model (catalog lookup)"
	case "service_model":
		return "Service/model reference"
	case "bare_model":
		return "Bare model search"
	default:
		return t
	}
}

func resolveWithDetails(svc *llmproviders.Service, modelRef string) ResolutionResult {
	result := ResolutionResult{
		Input: modelRef,
		Steps: []ResolutionStep{},
	}

	// Step 1: Check intent aliases
	intentAliases := svc.IntentAliases()
	step1 := ResolutionStep{Step: 1, Check: "Checked intent aliases"}
	if ref, ok := intentAliases[modelRef]; ok {
		step1.Matched = true
		step1.Detail = fmt.Sprintf("found in %s", ref.InstanceName)
		result.Steps = append(result.Steps, step1)
		result.Type = "intent_alias"
		result.WireModelID = ref.WireModelID
		result.ProviderName = ref.InstanceName
		result.ServiceID = ref.ServiceID
		return result
	}
	result.Steps = append(result.Steps, step1)

	// Step 2: Check provider aliases
	providerAliases := svc.ProviderAliases()
	step2 := ResolutionStep{Step: 2, Check: "Checked provider aliases"}
	if target, ok := providerAliases[modelRef]; ok {
		step2.Matched = true
		step2.Detail = fmt.Sprintf("found in %s", target.InstanceName)
		result.Steps = append(result.Steps, step2)
		result.Type = "provider_alias"
		result.ProviderName = target.InstanceName
		result.ServiceID = target.ServiceID
		result.WireModelID = target.WireModelID
		return result
	}
	result.Steps = append(result.Steps, step2)

	// Step 3: Try actual resolution to get final result
	provider, wireModel, err := svc.ProviderFor(modelRef)
	if err != nil {
		result.Error = err.Error()

		// Add remaining steps as not matched
		step3 := ResolutionStep{Step: 3, Check: "Checked catalog wire models"}
		result.Steps = append(result.Steps, step3)

		step4 := ResolutionStep{Step: 4, Check: "Parsed as service/model"}
		result.Steps = append(result.Steps, step4)

		step5 := ResolutionStep{Step: 5, Check: "Searched all services"}
		result.Steps = append(result.Steps, step5)

		return result
	}

	result.WireModelID = wireModel
	result.ProviderName = provider.Name()
	result.ServiceID = provider.Name() // Instance typically matches service

	// Determine which step matched
	if strings.Contains(modelRef, "/") {
		// Could be catalog wire model or service/model
		// If the wire model matches the input exactly, it's likely catalog lookup
		if wireModel == modelRef {
			step3 := ResolutionStep{
				Step:    3,
				Check:   "Checked catalog wire models",
				Matched: true,
				Detail:  fmt.Sprintf("found in %s catalog", provider.Name()),
			}
			result.Steps = append(result.Steps, step3)
			result.Type = "catalog_wire_model"
		} else {
			step3 := ResolutionStep{Step: 3, Check: "Checked catalog wire models"}
			result.Steps = append(result.Steps, step3)

			step4 := ResolutionStep{
				Step:    4,
				Check:   "Parsed as service/model",
				Matched: true,
				Detail:  fmt.Sprintf("service=%s, model=%s", provider.Name(), wireModel),
			}
			result.Steps = append(result.Steps, step4)
			result.Type = "service_model"
		}
	} else {
		step3 := ResolutionStep{Step: 3, Check: "Checked catalog wire models"}
		result.Steps = append(result.Steps, step3)

		step4 := ResolutionStep{Step: 4, Check: "Parsed as service/model"}
		result.Steps = append(result.Steps, step4)

		step5 := ResolutionStep{
			Step:    5,
			Check:   "Searched all services",
			Matched: true,
			Detail:  fmt.Sprintf("found in %s", provider.Name()),
		}
		result.Steps = append(result.Steps, step5)
		result.Type = "bare_model"
	}

	return result
}

type resolveJSONOutput struct {
	Input       string            `json:"input"`
	Type        string            `json:"type"`
	WireModelID string            `json:"wire_model_id"`
	Provider    string            `json:"provider"`
	Service     string            `json:"service"`
	Steps       []resolveJSONStep `json:"steps"`
	Error       string            `json:"error,omitempty"`
}

type resolveJSONStep struct {
	Step    int    `json:"step"`
	Check   string `json:"check"`
	Matched bool   `json:"matched"`
	Detail  string `json:"detail,omitempty"`
}

func printResolveJSON(w io.Writer, svc *llmproviders.Service, modelRef string) error {
	result := resolveWithDetails(svc, modelRef)

	output := resolveJSONOutput{
		Input:       result.Input,
		Type:        result.Type,
		WireModelID: result.WireModelID,
		Provider:    result.ProviderName,
		Service:     result.ServiceID,
		Error:       result.Error,
		Steps:       make([]resolveJSONStep, len(result.Steps)),
	}

	for i, step := range result.Steps {
		output.Steps[i] = resolveJSONStep{
			Step:    step.Step,
			Check:   step.Check,
			Matched: step.Matched,
			Detail:  step.Detail,
		}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}
