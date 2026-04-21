package registry

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/codewandler/agentapis/conversation"
)

var (
	ErrProviderNotFound = errors.New("provider not found")
	ErrNoProviders      = errors.New("no providers")
)

// Provider is the interface that all LLM providers must implement.
type Provider interface {
	// Name returns the unique instance name of this provider.
	Name() string

	// CreateSession creates a new conversation session.
	CreateSession(opts ...conversation.Option) *conversation.Session
}

// Registration defines how a provider registers with the system.
type Registration struct {
	// InstanceName is the unique identifier for this provider instance.
	// Defaults to ServiceID if empty. Multiple instances of the same
	// ServiceID can exist with different InstanceNames.
	InstanceName string

	// ServiceID is the modeldb service identifier (e.g., "anthropic", "openai", "openrouter").
	// This determines which offerings from the catalog are available.
	ServiceID string

	// Order controls detection priority (lower = higher priority).
	// Providers with lower Order values are preferred when resolving ambiguous models.
	Order int

	// Aliases maps short names to wire model IDs for this provider.
	// Example: {"sonnet": "claude-sonnet-4-6", "opus": "claude-opus-4-6"}
	Aliases map[string]string

	// IntentAliases maps intent names to wire model IDs.
	// Example: {"fast": "claude-haiku-4-5-20251001", "powerful": "claude-opus-4-6"}
	IntentAliases map[string]string

	// Detect checks if this provider is available (e.g., API key present).
	Detect func(ctx context.Context) (bool, error)

	// Build creates the provider instance.
	Build func(ctx context.Context, cfg BuildConfig) (Provider, error)
}

// DetectedProvider represents a provider that was detected as available.
type DetectedProvider struct {
	InstanceName  string
	ServiceID     string
	Order         int
	Aliases       map[string]string
	IntentAliases map[string]string
}

// BuildConfig provides configuration options when building a provider.
type BuildConfig struct {
	HTTPClient interface{}
	Options    []conversation.Option
}

// Registry manages provider registrations and detection.
type Registry struct {
	mu            sync.RWMutex
	registrations []Registration // ordered list
}

// New creates a new empty Registry.
func New() *Registry {
	return &Registry{
		registrations: make([]Registration, 0),
	}
}

// Register adds a provider registration to the registry.
func (r *Registry) Register(reg Registration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Default InstanceName to ServiceID if not set
	if reg.InstanceName == "" {
		reg.InstanceName = reg.ServiceID
	}

	r.registrations = append(r.registrations, reg)

	// Keep sorted by Order
	sort.SliceStable(r.registrations, func(i, j int) bool {
		return r.registrations[i].Order < r.registrations[j].Order
	})
}

// Detect runs detection for all registered providers and returns those that are available.
func (r *Registry) Detect(ctx context.Context) ([]DetectedProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []DetectedProvider
	for _, reg := range r.registrations {
		if reg.Detect == nil {
			continue
		}

		available, err := reg.Detect(ctx)
		if err != nil {
			return nil, fmt.Errorf("detect %s: %w", reg.InstanceName, err)
		}

		if available {
			out = append(out, DetectedProvider{
				InstanceName:  reg.InstanceName,
				ServiceID:     reg.ServiceID,
				Order:         reg.Order,
				Aliases:       reg.Aliases,
				IntentAliases: reg.IntentAliases,
			})
		}
	}

	// Already sorted by Order from registration
	return out, nil
}

// Build creates a provider instance for the given detected provider.
func (r *Registry) Build(ctx context.Context, dp DetectedProvider, cfg BuildConfig) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, reg := range r.registrations {
		if reg.InstanceName == dp.InstanceName {
			if reg.Build == nil {
				return nil, fmt.Errorf("provider %s has no build function", dp.InstanceName)
			}
			return reg.Build(ctx, cfg)
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, dp.InstanceName)
}

// ServiceIDs returns all unique service IDs from registrations.
func (r *Registry) ServiceIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]bool)
	var ids []string
	for _, reg := range r.registrations {
		if !seen[reg.ServiceID] {
			seen[reg.ServiceID] = true
			ids = append(ids, reg.ServiceID)
		}
	}
	sort.Strings(ids)
	return ids
}

// InstanceNames returns all instance names from registrations.
func (r *Registry) InstanceNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.registrations))
	for _, reg := range r.registrations {
		names = append(names, reg.InstanceName)
	}
	return names
}
