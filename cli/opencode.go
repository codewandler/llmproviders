package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	llmproviders "github.com/codewandler/llmproviders"
	"github.com/codewandler/modeldb"
	"github.com/spf13/cobra"
)

// OpencodeCommandOptions configures the opencode command.
type OpencodeCommandOptions struct {
	IO          IO
	LoadService ServiceLoader
}

// NewOpencodeCommand creates the "opencode" command group.
func NewOpencodeCommand(opts OpencodeCommandOptions) *cobra.Command {
	ioCfg := opts.IO.WithDefaults()

	cmd := &cobra.Command{
		Use:   "opencode",
		Short: "Manage OpenCode integration",
		Long: `Commands for integrating llmcli with OpenCode.

Commands:
  configure  Inject llmproviders proxy as an OpenCode provider`,
	}

	cmd.SetOut(ioCfg.Out)
	cmd.SetErr(ioCfg.Err)
	cmd.AddCommand(newOpencodeConfigureCommand(opts))

	return cmd
}

func newOpencodeConfigureCommand(opts OpencodeCommandOptions) *cobra.Command {
	ioCfg := opts.IO.WithDefaults()

	var (
		addr      string
		configDir string
		setModel  string
		remove    bool
	)

	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Inject llmproviders proxy into OpenCode config",
		Long: `Add an OpenAI-compatible provider entry to the global OpenCode config
that routes through the llmcli serve proxy.

All detected provider aliases (sonnet, opus, gpt, etc.) are registered as
models so they appear in the OpenCode model picker.

The proxy must be running for OpenCode to use it:
  llmcli serve

Examples:
  llmcli opencode configure                           # Default: localhost:8080
  llmcli opencode configure --addr :3000              # Custom port
  llmcli opencode configure --model llmproviders/o3   # Set default model
  llmcli opencode configure --remove                  # Remove the provider entry`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if remove {
				return removeOpencodeProvider(ioCfg, configDir)
			}

			svc, err := opts.LoadService(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to load service: %w", err)
			}

			return configureOpencode(ioCfg, svc, addr, configDir, setModel)
		},
	}

	cmd.SetOut(ioCfg.Out)
	cmd.SetErr(ioCfg.Err)

	f := cmd.Flags()
	f.StringVarP(&addr, "addr", "a", ":8080", "Proxy listen address (must match llmcli serve --addr)")
	f.StringVar(&configDir, "config-dir", "", "OpenCode config directory (default: ~/.config/opencode)")
	f.StringVarP(&setModel, "model", "m", "", "Set as default model (e.g. llmproviders/sonnet)")
	f.BoolVar(&remove, "remove", false, "Remove the llmproviders provider entry")

	return cmd
}

const opencodeProviderKey = "llmproviders"

// opencodeConfig represents the parts of opencode.json we care about.
// We use ordered map operations to preserve unknown fields.
type opencodeConfig = map[string]any

func opencodeConfigPath(configDir string) (string, error) {
	if configDir != "" {
		return filepath.Join(configDir, "opencode.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "opencode", "opencode.json"), nil
}

func readOpencodeConfig(path string) (opencodeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return opencodeConfig{
				"$schema": "https://opencode.ai/config.json",
			}, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var cfg opencodeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return cfg, nil
}

func writeOpencodeConfig(path string, cfg opencodeConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func configureOpencode(ioCfg IO, svc *llmproviders.Service, addr, configDir, setModel string) error {
	cfgPath, err := opencodeConfigPath(configDir)
	if err != nil {
		return err
	}

	cfg, err := readOpencodeConfig(cfgPath)
	if err != nil {
		return err
	}

	// Build base URL from addr.
	host, port := "localhost", "8080"
	if h, p, ok := splitHostPort(addr); ok {
		if h != "" {
			host = h
		}
		port = p
	}
	baseURL := fmt.Sprintf("http://%s:%s/v1", host, port)

	// Build models map from detected aliases with catalog metadata.
	aliases := svc.ProviderAliases()
	catalog := svc.Catalog()
	models := make(map[string]any, len(aliases))
	for alias, target := range aliases {
		entry := map[string]any{
			"name": fmt.Sprintf("%s (%s via %s)", alias, target.WireModelID, target.ServiceID),
		}

		// Look up limits from catalog: first try offering override, then model record.
		var contextWindow, maxOutput int
		if offering, ok := catalog.OfferingByRef(modeldb.OfferingRef{
			ServiceID: target.ServiceID, WireModelID: target.WireModelID,
		}); ok {
			if offering.LimitsOverride != nil {
				contextWindow = offering.LimitsOverride.ContextWindow
				maxOutput = offering.LimitsOverride.MaxOutput
			}
			if contextWindow == 0 || maxOutput == 0 {
				if model, mok := catalog.ModelByKey(offering.ModelKey); mok {
					if contextWindow == 0 {
						contextWindow = model.Limits.ContextWindow
					}
					if maxOutput == 0 {
						maxOutput = model.Limits.MaxOutput
					}
				}
			}
		}

		if contextWindow > 0 || maxOutput > 0 {
			limit := map[string]any{}
			if contextWindow > 0 {
				limit["context"] = contextWindow
			}
			if maxOutput > 0 {
				limit["output"] = maxOutput
			}
			entry["limit"] = limit
		}

		models[alias] = entry
	}

	if len(models) == 0 {
		return fmt.Errorf("no provider aliases detected — is at least one provider configured?")
	}

	// Build provider entry.
	providerEntry := map[string]any{
		"npm":  "@ai-sdk/openai",
		"name": "LLM Providers Proxy",
		"options": map[string]any{
			"baseURL": baseURL,
			"apiKey":  "unused",
		},
		"models": models,
	}

	// Inject into config.
	providers, _ := cfg["provider"].(map[string]any)
	if providers == nil {
		providers = make(map[string]any)
	}
	providers[opencodeProviderKey] = providerEntry
	cfg["provider"] = providers

	// Set default model if requested.
	if setModel != "" {
		cfg["model"] = setModel
	}

	if err := writeOpencodeConfig(cfgPath, cfg); err != nil {
		return err
	}

	// Report.
	out := ioCfg.Out
	fmt.Fprintf(out, "Updated %s\n\n", cfgPath)
	fmt.Fprintf(out, "Provider:  %s\n", opencodeProviderKey)
	fmt.Fprintf(out, "Base URL:  %s\n", baseURL)
	fmt.Fprintf(out, "Models:    %d aliases\n", len(models))

	for _, alias := range sortedMapKeys(aliases) {
		target := aliases[alias]
		fmt.Fprintf(out, "  %-12s -> %s (%s)\n", alias, target.WireModelID, target.ServiceID)
	}

	if setModel != "" {
		fmt.Fprintf(out, "\nDefault model: %s\n", setModel)
	}

	fmt.Fprintf(out, "\nMake sure the proxy is running:\n  llmcli serve --addr %s\n", addr)

	return nil
}

func removeOpencodeProvider(ioCfg IO, configDir string) error {
	cfgPath, err := opencodeConfigPath(configDir)
	if err != nil {
		return err
	}

	cfg, err := readOpencodeConfig(cfgPath)
	if err != nil {
		return err
	}

	providers, _ := cfg["provider"].(map[string]any)
	if providers == nil {
		fmt.Fprintf(ioCfg.Out, "No provider section in %s — nothing to remove.\n", cfgPath)
		return nil
	}

	if _, exists := providers[opencodeProviderKey]; !exists {
		fmt.Fprintf(ioCfg.Out, "Provider %q not found in %s — nothing to remove.\n", opencodeProviderKey, cfgPath)
		return nil
	}

	delete(providers, opencodeProviderKey)
	if len(providers) == 0 {
		delete(cfg, "provider")
	}

	// If model pointed to our provider, remove it.
	if model, _ := cfg["model"].(string); len(model) > len(opencodeProviderKey)+1 && model[:len(opencodeProviderKey)+1] == opencodeProviderKey+"/" {
		delete(cfg, "model")
	}

	if err := writeOpencodeConfig(cfgPath, cfg); err != nil {
		return err
	}

	fmt.Fprintf(ioCfg.Out, "Removed %q provider from %s\n", opencodeProviderKey, cfgPath)
	return nil
}

// splitHostPort splits "host:port" without requiring net import.
func splitHostPort(addr string) (host, port string, ok bool) {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i], addr[i+1:], true
		}
	}
	return "", "", false
}
