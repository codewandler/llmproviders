package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/codewandler/agentapis/api/unified"
	"github.com/codewandler/agentapis/conversation"
	"github.com/spf13/cobra"
)

// ANSI escape codes for terminal styling.
const (
	ansiDim   = "\033[2m"
	ansiReset = "\033[0m"
)

// InferCommandOptions configures the infer command.
type InferCommandOptions struct {
	IO          IO
	LoadService ServiceLoader
}

// inferParams holds the parsed parameters for inference.
type inferParams struct {
	message     string
	model       string
	system      string
	maxTokens   int
	temperature float64
	thinking    string
	effort      string
	verbose     bool
}

// NewInferCommand creates the "infer" command that sends a message to an LLM.
func NewInferCommand(opts InferCommandOptions) *cobra.Command {
	ioCfg := opts.IO.WithDefaults()

	var params inferParams
	params.model = "default"
	params.maxTokens = 8000

	cmd := &cobra.Command{
		Use:   "infer <message>",
		Short: "Send a message to an LLM and stream the response",
		Long: `Send a message to an LLM and stream the response.

Examples:
  llmcli infer "Hello, how are you?"              # Quick inference with default model
  llmcli infer -m powerful "Write a poem"         # Use most capable model
  llmcli infer -m sonnet "Explain Go channels"    # Use specific model alias
  llmcli infer -s "You are helpful" "Hello"       # With system prompt
  llmcli infer --thinking on "Complex problem"    # Enable thinking/reasoning
  llmcli infer --effort high "Hard question"      # High effort mode
  llmcli infer -v "Hello"                         # Verbose (show resolution, usage)
  llmcli infer --max-tokens 512 "Short answer"    # Limit output length
  llmcli infer --temperature 0.7 "Creative task"  # Higher randomness`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params.message = args[0]
			return runInfer(cmd.Context(), opts, ioCfg, params)
		},
	}

	cmd.SetOut(ioCfg.Out)
	cmd.SetErr(ioCfg.Err)

	f := cmd.Flags()
	f.StringVarP(&params.model, "model", "m", params.model, "Model alias or full path (default, fast, powerful, sonnet, etc.)")
	f.StringVarP(&params.system, "system", "s", "", "System prompt")
	f.IntVar(&params.maxTokens, "max-tokens", params.maxTokens, "Maximum output tokens")
	f.Float64Var(&params.temperature, "temperature", 0, "Sampling temperature 0.0-2.0 (0 = provider default)")
	f.StringVar(&params.thinking, "thinking", "", "Thinking mode: auto, on, off")
	f.StringVar(&params.effort, "effort", "", "Effort level: low, medium, high, max")
	f.BoolVarP(&params.verbose, "verbose", "v", false, "Show model resolution and usage stats")

	// Register flag completions
	_ = cmd.RegisterFlagCompletionFunc("model", completeModelRefs(opts.LoadService))
	_ = cmd.RegisterFlagCompletionFunc("thinking", completeThinking)
	_ = cmd.RegisterFlagCompletionFunc("effort", completeEffort)

	return cmd
}

func runInfer(ctx context.Context, opts InferCommandOptions, ioCfg IO, params inferParams) error {
	out := ioCfg.Out
	errOut := ioCfg.Err

	// Load service
	svc, err := opts.LoadService(ctx)
	if err != nil {
		return fmt.Errorf("failed to load service: %w", err)
	}

	// Resolve model
	provider, wireModel, err := svc.ProviderFor(params.model)
	if err != nil {
		return fmt.Errorf("failed to resolve model %q: %w", params.model, err)
	}

	if params.verbose {
		fmt.Fprintf(errOut, "%s── model resolved ──%s\n", ansiDim, ansiReset)
		fmt.Fprintf(errOut, "  input: %s\n", params.model)
		fmt.Fprintf(errOut, "  provider: %s\n", provider.Name())
		fmt.Fprintf(errOut, "  wire_model: %s\n", wireModel)
		fmt.Fprintln(errOut)
	}

	// Build session options
	sessionOpts := []conversation.Option{
		conversation.WithModel(wireModel),
		conversation.WithMaxTokens(params.maxTokens),
	}

	if params.temperature > 0 {
		sessionOpts = append(sessionOpts, conversation.WithTemperature(params.temperature))
	}

	if params.thinking != "" {
		sessionOpts = append(sessionOpts, conversation.WithThinking(unified.ThinkingMode(params.thinking)))
	}

	if params.effort != "" {
		sessionOpts = append(sessionOpts, conversation.WithEffort(unified.Effort(params.effort)))
	}

	if params.system != "" {
		sessionOpts = append(sessionOpts, conversation.WithSystem(params.system))
	}

	// Create session
	session := provider.CreateSession(sessionOpts...)

	// Build request
	req := conversation.Request{
		Inputs: []conversation.Input{
			{Role: unified.RoleUser, Text: params.message},
		},
	}

	// Stream response
	stream, err := session.Request(ctx, req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	// Process events
	var (
		inReasoning   bool
		hadTextOutput bool
		lastUsage     *conversation.UsageEvent
	)

	for ev := range stream {
		switch e := ev.(type) {
		case conversation.TextDeltaEvent:
			if inReasoning {
				fmt.Fprint(out, ansiReset)
				inReasoning = false
			}
			fmt.Fprint(out, e.Text)
			hadTextOutput = true

		case conversation.ReasoningDeltaEvent:
			if !inReasoning {
				fmt.Fprint(out, ansiDim)
				inReasoning = true
			}
			fmt.Fprint(out, e.Text)

		case conversation.UsageEvent:
			lastUsage = &e

		case conversation.ErrorEvent:
			if inReasoning {
				fmt.Fprint(out, ansiReset)
			}
			return e.Err

		case conversation.CompletedEvent:
			// Response complete
		}
	}

	// Reset styling
	if inReasoning {
		fmt.Fprint(out, ansiReset)
	}

	// Ensure newline after output
	if hadTextOutput {
		fmt.Fprintln(out)
	}

	// Print usage in verbose mode
	if params.verbose && lastUsage != nil {
		printUsage(errOut, lastUsage.Usage)
	}

	return nil
}

// printUsage prints token usage information.
func printUsage(w io.Writer, usage unified.StreamUsage) {
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s── usage ──%s\n", ansiDim, ansiReset)

	for _, item := range usage.Tokens {
		if item.Count > 0 {
			fmt.Fprintf(w, "  %s: %d\n", item.Kind, item.Count)
		}
	}

	// Print costs if available
	var totalCost float64
	for _, cost := range usage.Costs {
		totalCost += cost.Amount
	}
	if totalCost > 0 {
		fmt.Fprintf(w, "  cost: %s\n", formatCost(totalCost))
	}
}

// formatCost formats cost with appropriate precision for the amount.
func formatCost(cost float64) string {
	switch {
	case cost < 0.0001:
		return fmt.Sprintf("$%.8f", cost)
	case cost < 0.01:
		return fmt.Sprintf("$%.6f", cost)
	case cost < 1.0:
		return fmt.Sprintf("$%.4f", cost)
	default:
		return fmt.Sprintf("$%.2f", cost)
	}
}
