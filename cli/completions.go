package cli

import (
	"context"
	"sort"
	"strings"

	"github.com/codewandler/modeldb"
	"github.com/spf13/cobra"
)

// CatalogLoader loads a modeldb Catalog.
type CatalogLoader func(ctx context.Context) (modeldb.Catalog, error)

// completeModelRefs returns completion for model references.
// Includes intent aliases, provider aliases, and full model paths.
// Results are sorted alphabetically.
func completeModelRefs(loadService ServiceLoader) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		svc, err := loadService(cmd.Context())
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		toComplete = strings.ToLower(toComplete)
		seen := make(map[string]bool)
		var matches []string

		add := func(s string) {
			if !seen[s] && strings.Contains(strings.ToLower(s), toComplete) {
				seen[s] = true
				matches = append(matches, s)
			}
		}

		// Intent aliases (fast, default, powerful)
		for intent := range svc.IntentAliases() {
			add(intent)
		}
		// Provider aliases (sonnet, opus, haiku, etc.)
		for alias := range svc.ProviderAliases() {
			add(alias)
		}
		// Full model paths
		for _, m := range svc.Models("") {
			add(m)
		}

		sort.Strings(matches)
		return matches, cobra.ShellCompDirectiveNoFileComp
	}
}

// completeServices returns completion for service IDs from detected providers.
func completeServices(loadService ServiceLoader) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		svc, err := loadService(cmd.Context())
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		services := svc.RegisteredServices() // already sorted
		if toComplete == "" {
			return services, cobra.ShellCompDirectiveNoFileComp
		}

		toComplete = strings.ToLower(toComplete)
		var matches []string
		for _, s := range services {
			if strings.Contains(strings.ToLower(s), toComplete) {
				matches = append(matches, s)
			}
		}
		return matches, cobra.ShellCompDirectiveNoFileComp
	}
}

// completeCatalogServices returns completion for service IDs from the catalog.
func completeCatalogServices(loadCatalog CatalogLoader) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cat, err := loadCatalog(cmd.Context())
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		// Extract and sort service IDs
		ids := make([]string, 0, len(cat.Services))
		for id := range cat.Services {
			ids = append(ids, id)
		}
		sort.Strings(ids)

		if toComplete == "" {
			return ids, cobra.ShellCompDirectiveNoFileComp
		}

		toComplete = strings.ToLower(toComplete)
		var matches []string
		for _, id := range ids {
			if strings.Contains(strings.ToLower(id), toComplete) {
				matches = append(matches, id)
			}
		}
		return matches, cobra.ShellCompDirectiveNoFileComp
	}
}

// completeThinking returns completion for --thinking flag values.
func completeThinking(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return []string{"auto", "on", "off"}, cobra.ShellCompDirectiveNoFileComp
}

// completeEffort returns completion for --effort flag values.
func completeEffort(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return []string{"low", "medium", "high", "max"}, cobra.ShellCompDirectiveNoFileComp
}
