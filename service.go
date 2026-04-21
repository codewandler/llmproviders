package llmproviders

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/codewandler/llmproviders/registry"
	"github.com/codewandler/modeldb"
)

var (
	ErrProviderNotFound = errors.New("provider not found")
	ErrModelNotFound    = errors.New("model not found")
	ErrAmbiguousModel   = errors.New("ambiguous model")
	ErrNoProviders      = errors.New("no providers available")
)

// ResolvedRef represents a fully resolved model reference.
type ResolvedRef struct {
	InstanceName string
	ServiceID    string
	WireModelID  string
}

// AliasTarget represents the target of a provider alias.
type AliasTarget struct {
	ServiceID   string
	WireModelID string
}

// providerInstance holds a registered provider and its metadata.
type providerInstance struct {
	InstanceName  string
	ServiceID     string
	Order         int
	Provider      registry.Provider
	Aliases       map[string]string
	IntentAliases map[string]string
}

// Service manages provider registration and model resolution.
type Service struct {
	mu sync.RWMutex

	// catalog is the modeldb catalog for model lookup
	catalog modeldb.Catalog

	// instances maps instanceName → provider instance
	instances map[string]*providerInstance

	// serviceProviders maps serviceID → []instanceName (for lookup by service)
	serviceProviders map[string][]string

	// intentAliases maps "fast", "default", "powerful" → resolved ref
	intentAliases map[string]ResolvedRef

	// providerAliases maps short alias → target (serviceID, wireModelID)
	providerAliases map[string]AliasTarget

	// registry is used for detection and building
	registry *registry.Registry
}

// ServiceConfig holds configuration for creating a Service.
type ServiceConfig struct {
	Registry          *registry.Registry
	Catalog           modeldb.Catalog
	AliasOverlay      *modeldb.AliasOverlay
	PreferenceOverlay *modeldb.PreferenceOverlay
}

// ServiceOption configures the Service.
type ServiceOption func(*ServiceConfig)

// WithRegistry sets the provider registry.
func WithRegistry(r *registry.Registry) ServiceOption {
	return func(c *ServiceConfig) { c.Registry = r }
}

// WithCatalog sets a custom modeldb catalog.
func WithCatalog(cat modeldb.Catalog) ServiceOption {
	return func(c *ServiceConfig) { c.Catalog = cat }
}

// NewService creates a new Service with the given options.
func NewService(opts ...ServiceOption) (*Service, error) {
	cfg := ServiceConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	// Load catalog
	var catalog modeldb.Catalog
	var err error
	if cfg.Catalog.Offerings != nil {
		catalog = cfg.Catalog
	} else {
		catalog, err = modeldb.LoadBuiltIn()
		if err != nil {
			return nil, fmt.Errorf("failed to load modeldb catalog: %w", err)
		}
	}

	s := &Service{
		catalog:          catalog,
		registry:         cfg.Registry,
		instances:        make(map[string]*providerInstance),
		serviceProviders: make(map[string][]string),
		intentAliases:    make(map[string]ResolvedRef),
		providerAliases:  make(map[string]AliasTarget),
	}

	if cfg.Registry != nil {
		if err := s.loadFromRegistry(); err != nil {
			return nil, err
		}
	}

	return s, nil
}

// loadFromRegistry detects available providers and builds them.
func (s *Service) loadFromRegistry() error {
	ctx := context.Background()

	detected, err := s.registry.Detect(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect providers: %w", err)
	}

	for _, d := range detected {
		p, err := s.registry.Build(ctx, d, registry.BuildConfig{})
		if err != nil {
			continue // Skip providers that fail to build
		}

		inst := &providerInstance{
			InstanceName:  d.InstanceName,
			ServiceID:     d.ServiceID,
			Order:         d.Order,
			Provider:      p,
			Aliases:       d.Aliases,
			IntentAliases: d.IntentAliases,
		}

		s.instances[d.InstanceName] = inst
		s.serviceProviders[d.ServiceID] = append(s.serviceProviders[d.ServiceID], d.InstanceName)
	}

	if len(s.instances) == 0 {
		return ErrNoProviders
	}

	// Sort service providers by order
	for svcID := range s.serviceProviders {
		sort.Slice(s.serviceProviders[svcID], func(i, j int) bool {
			instI := s.instances[s.serviceProviders[svcID][i]]
			instJ := s.instances[s.serviceProviders[svcID][j]]
			return instI.Order < instJ.Order
		})
	}

	// Merge aliases from all providers (first provider wins by Order)
	s.mergeAliases()

	return nil
}

// mergeAliases combines aliases from all providers, respecting priority.
func (s *Service) mergeAliases() {
	// Get instances sorted by Order
	sorted := s.instancesByOrder()

	// Merge intent aliases (first provider wins)
	for _, inst := range sorted {
		for intent, wireModel := range inst.IntentAliases {
			if _, exists := s.intentAliases[intent]; !exists {
				s.intentAliases[intent] = ResolvedRef{
					InstanceName: inst.InstanceName,
					ServiceID:    inst.ServiceID,
					WireModelID:  wireModel,
				}
			}
		}
	}

	// Merge provider aliases (first provider wins)
	for _, inst := range sorted {
		for alias, wireModel := range inst.Aliases {
			if _, exists := s.providerAliases[alias]; !exists {
				s.providerAliases[alias] = AliasTarget{
					ServiceID:   inst.ServiceID,
					WireModelID: wireModel,
				}
			}
		}
	}
}

// instancesByOrder returns all provider instances sorted by Order.
func (s *Service) instancesByOrder() []*providerInstance {
	instances := make([]*providerInstance, 0, len(s.instances))
	for _, inst := range s.instances {
		instances = append(instances, inst)
	}
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].Order < instances[j].Order
	})
	return instances
}

// ProviderFor resolves a model reference to a provider and wire model ID.
//
// Resolution order (first match wins):
//
//  1. Intent aliases ("fast", "default", "powerful")
//     Maps to the highest-priority detected provider's intent configuration.
//     Example: "fast" → anthropic's claude-haiku-4-5-20251001 (if anthropic detected)
//
//  2. Provider aliases ("sonnet", "opus", "haiku", "mini", etc.)
//     Short names registered by each provider, merged by priority.
//     Example: "sonnet" → claude-sonnet-4-6 via anthropic (order=20)
//
//  3. Catalog wire model lookup (full string as-is)
//     Checks if the input exists as a wire model ID in any registered service's
//     catalog. This handles OpenRouter-style models like "anthropic/claude-3-5-haiku".
//     Example: "anthropic/claude-3-5-haiku" → openrouter (it's in their catalog)
//
//  4. Parse as [instance/]service/model
//     If the input contains "/", try to parse it as a service or instance reference.
//     Example: "anthropic/claude-sonnet-4-6" → native anthropic provider
//
//  5. Bare model search
//     Search all registered services for the model. Returns error if ambiguous.
//     Example: "gpt-5.4" → openai (if only openai has it)
//
// Important: Step 3 runs BEFORE step 4, so OpenRouter wire models like
// "anthropic/claude-3-5-haiku" route to OpenRouter, not native Anthropic.
// To force native routing, the model must not exist in any aggregator's catalog.
func (s *Service) ProviderFor(model string) (registry.Provider, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Step 1: Check intent aliases (fast, default, powerful)
	if ref, ok := s.intentAliases[model]; ok {
		inst, ok := s.instances[ref.InstanceName]
		if ok {
			return inst.Provider, ref.WireModelID, nil
		}
	}

	// Step 2: Check provider aliases (sonnet, opus, haiku, etc.)
	if target, ok := s.providerAliases[model]; ok {
		return s.findProviderForService(target.ServiceID, target.WireModelID)
	}

	// Step 3: Check if full string is a wire model ID in catalog
	if services := s.findServicesForWireModel(model); len(services) > 0 {
		return s.resolveByWireModel(model, services)
	}

	// Step 4: Try parsing as [instance/]service/model
	parsed := s.parseModelRef(model)

	if parsed.InstanceName != "" {
		return s.resolveByInstance(parsed)
	}

	if parsed.ServiceID != "" {
		return s.resolveByServiceID(parsed)
	}

	// Step 5: Bare model - search all services
	return s.resolveBareModel(parsed.WireModel)
}

// parsedModelRef holds the components of a parsed model reference.
type parsedModelRef struct {
	InstanceName string
	ServiceID    string
	WireModel    string
}

// parseModelRef parses a model reference string.
// Formats:
//   - "model" → ("", "", "model")
//   - "service/model" → ("", "service", "model") if service is known
//   - "instance/model" → ("instance", "", "model") if instance is known
//   - "instance/service/model" → ("instance", "service", "model")
func (s *Service) parseModelRef(model string) parsedModelRef {
	parts := strings.Split(model, "/")

	switch len(parts) {
	case 1:
		return parsedModelRef{WireModel: parts[0]}

	case 2:
		// Check if first part is a known service ID
		if _, ok := s.serviceProviders[parts[0]]; ok {
			return parsedModelRef{ServiceID: parts[0], WireModel: parts[1]}
		}
		// Check if first part is a known instance name
		if _, ok := s.instances[parts[0]]; ok {
			return parsedModelRef{InstanceName: parts[0], WireModel: parts[1]}
		}
		// Unknown prefix - treat as service/model
		return parsedModelRef{ServiceID: parts[0], WireModel: parts[1]}

	default:
		// 3+ parts: could be instance/service/model or just a complex wire model
		// Check if first part is known
		if _, ok := s.instances[parts[0]]; ok {
			if _, ok := s.serviceProviders[parts[1]]; ok {
				return parsedModelRef{
					InstanceName: parts[0],
					ServiceID:    parts[1],
					WireModel:    strings.Join(parts[2:], "/"),
				}
			}
			return parsedModelRef{
				InstanceName: parts[0],
				WireModel:    strings.Join(parts[1:], "/"),
			}
		}
		if _, ok := s.serviceProviders[parts[0]]; ok {
			return parsedModelRef{
				ServiceID: parts[0],
				WireModel: strings.Join(parts[1:], "/"),
			}
		}
		// Unknown - treat as bare model
		return parsedModelRef{WireModel: model}
	}
}

// findServicesForWireModel finds all services that offer a given wire model.
func (s *Service) findServicesForWireModel(wireModel string) []string {
	var services []string
	seen := make(map[string]bool)

	for serviceID := range s.serviceProviders {
		if s.modelExistsForService(serviceID, wireModel) {
			if !seen[serviceID] {
				services = append(services, serviceID)
				seen[serviceID] = true
			}
		}
	}

	return services
}

// modelExistsForService checks if a wire model exists in the catalog for a service.
func (s *Service) modelExistsForService(serviceID, wireModel string) bool {
	offerings := s.catalog.OfferingsByService(serviceID)
	for _, offering := range offerings {
		if offering.WireModelID == wireModel {
			return true
		}
		for _, alias := range offering.Aliases {
			if alias == wireModel {
				return true
			}
		}
	}
	return false
}

// resolveByWireModel resolves a wire model ID to a provider.
func (s *Service) resolveByWireModel(wireModel string, services []string) (registry.Provider, string, error) {
	if len(services) == 1 {
		return s.findProviderForService(services[0], wireModel)
	}

	// Multiple matches - use priority ordering
	best := s.pickBestService(services)
	if best == "" {
		return nil, "", fmt.Errorf("%w: %q", ErrModelNotFound, wireModel)
	}

	return s.findProviderForService(best, wireModel)
}

// pickBestService returns the service with the lowest Order value.
func (s *Service) pickBestService(services []string) string {
	var best string
	bestOrder := int(^uint(0) >> 1) // max int

	for _, svcID := range services {
		instances := s.serviceProviders[svcID]
		if len(instances) > 0 {
			inst := s.instances[instances[0]]
			if inst.Order < bestOrder {
				bestOrder = inst.Order
				best = svcID
			}
		}
	}

	return best
}

// findProviderForService returns the provider for a given service.
func (s *Service) findProviderForService(serviceID, wireModel string) (registry.Provider, string, error) {
	instances := s.serviceProviders[serviceID]
	if len(instances) == 0 {
		return nil, "", fmt.Errorf("%w: service %q not configured", ErrProviderNotFound, serviceID)
	}

	inst := s.instances[instances[0]]
	return inst.Provider, wireModel, nil
}

// resolveByInstance resolves a model reference by instance name.
func (s *Service) resolveByInstance(ref parsedModelRef) (registry.Provider, string, error) {
	inst, ok := s.instances[ref.InstanceName]
	if !ok {
		return nil, "", fmt.Errorf("%w: instance %q", ErrProviderNotFound, ref.InstanceName)
	}

	// Validate model exists for the service if catalog has it
	if ref.ServiceID != "" && ref.ServiceID != inst.ServiceID {
		return nil, "", fmt.Errorf("%w: instance %q is service %q, not %q",
			ErrProviderNotFound, ref.InstanceName, inst.ServiceID, ref.ServiceID)
	}

	return inst.Provider, ref.WireModel, nil
}

// resolveByServiceID resolves a model reference by service ID.
func (s *Service) resolveByServiceID(ref parsedModelRef) (registry.Provider, string, error) {
	instances := s.serviceProviders[ref.ServiceID]
	if len(instances) == 0 {
		return nil, "", fmt.Errorf("%w: service %q not configured", ErrProviderNotFound, ref.ServiceID)
	}

	// Validate model exists in catalog
	if !s.modelExistsForService(ref.ServiceID, ref.WireModel) {
		// Try to find a matching alias
		offerings := s.catalog.OfferingsByService(ref.ServiceID)
		for _, offering := range offerings {
			for _, alias := range offering.Aliases {
				if alias == ref.WireModel {
					return s.instances[instances[0]].Provider, offering.WireModelID, nil
				}
			}
		}
		return nil, "", fmt.Errorf("%w: model %q not available for service %q",
			ErrModelNotFound, ref.WireModel, ref.ServiceID)
	}

	inst := s.instances[instances[0]]
	return inst.Provider, ref.WireModel, nil
}

// resolveBareModel resolves a bare model name (no service/instance prefix).
func (s *Service) resolveBareModel(wireModel string) (registry.Provider, string, error) {
	services := s.findServicesForWireModel(wireModel)

	if len(services) == 0 {
		return nil, "", fmt.Errorf("%w: %q", ErrModelNotFound, wireModel)
	}

	if len(services) > 1 {
		return nil, "", fmt.Errorf("%w: model %q matches services %v - specify as service/model",
			ErrAmbiguousModel, wireModel, services)
	}

	return s.findProviderForService(services[0], wireModel)
}

// RegisteredServices returns all unique service IDs that have registered providers.
func (s *Service) RegisteredServices() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.serviceProviders))
	for id := range s.serviceProviders {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// RegisteredInstances returns all instance names.
func (s *Service) RegisteredInstances() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.instances))
	for name := range s.instances {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// IntentAliases returns the merged intent aliases.
func (s *Service) IntentAliases() map[string]ResolvedRef {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy
	result := make(map[string]ResolvedRef, len(s.intentAliases))
	for k, v := range s.intentAliases {
		result[k] = v
	}
	return result
}

// ProviderAliases returns the merged provider aliases.
func (s *Service) ProviderAliases() map[string]AliasTarget {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy
	result := make(map[string]AliasTarget, len(s.providerAliases))
	for k, v := range s.providerAliases {
		result[k] = v
	}
	return result
}

// Models returns available model IDs for the given service, or all services if empty.
func (s *Service) Models(serviceID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if serviceID != "" {
		return s.modelsForService(serviceID)
	}

	// Return all models from all registered services
	var all []string
	seen := make(map[string]bool)

	for id := range s.serviceProviders {
		for _, m := range s.modelsForService(id) {
			if !seen[m] {
				seen[m] = true
				all = append(all, m)
			}
		}
	}

	sort.Strings(all)
	return all
}

// modelsForService returns model IDs for a specific service.
func (s *Service) modelsForService(serviceID string) []string {
	offerings := s.catalog.OfferingsByService(serviceID)
	var models []string
	seen := make(map[string]bool)

	for _, offering := range offerings {
		// Add prefixed wire model ID
		prefixed := serviceID + "/" + offering.WireModelID
		if !seen[prefixed] {
			seen[prefixed] = true
			models = append(models, prefixed)
		}

		// Add prefixed aliases
		for _, alias := range offering.Aliases {
			prefixedAlias := serviceID + "/" + alias
			if !seen[prefixedAlias] {
				seen[prefixedAlias] = true
				models = append(models, prefixedAlias)
			}
		}
	}

	sort.Strings(models)
	return models
}
