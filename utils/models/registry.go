package models

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/kris-hansen/comanda/utils/config"
)

// Global registry instance
var registry = &ProviderRegistry{
	factories: make(map[string]Factory),
}

// ProviderRegistry manages registered provider factories
type ProviderRegistry struct {
	factories map[string]Factory
	mutex     sync.RWMutex
}

// Factory creates provider instances and provides metadata
type Factory interface {
	CreateProvider() Provider
	GetMetadata() ProviderMetadata
}

// ProviderFactory is a reusable factory for all provider types
type ProviderFactory struct {
	constructor func() Provider
	metadata    ProviderMetadata
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory(constructor func() Provider, metadata ProviderMetadata) *ProviderFactory {
	return &ProviderFactory{
		constructor: constructor,
		metadata:    metadata,
	}
}

// CreateProvider creates a new provider instance using the constructor function
func (f *ProviderFactory) CreateProvider() Provider {
	return f.constructor()
}

// GetMetadata returns the provider metadata
func (f *ProviderFactory) GetMetadata() ProviderMetadata {
	return f.metadata
}

// ProviderMetadata contains information about a provider
type ProviderMetadata struct {
	Name          string
	Description   string
	Version       string
	ModelPrefixes []string // e.g., ["claude-", "gpt-"]
	Priority      int      // Higher priority = checked first
}

// RegisterProvider adds a provider factory to the registry
func RegisterProvider(name string, factory Factory) error {
	registry.mutex.Lock()
	defer registry.mutex.Unlock()

	if _, exists := registry.factories[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}

	registry.factories[name] = factory
	config.DebugLog("[Registry] Registered provider: %s", name)
	return nil
}

// FindProvider detects appropriate provider for model
func (r *ProviderRegistry) FindProvider(modelName string) Provider {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	config.DebugLog("[Registry] Finding provider for model: %s", modelName)

	// Get all factories and sort by priority
	type providerCandidate struct {
		factory  Factory
		metadata ProviderMetadata
	}

	var candidates []providerCandidate

	for _, factory := range r.factories {
		metadata := factory.GetMetadata()
		// Check if this provider supports the model
		for _, prefix := range metadata.ModelPrefixes {
			if strings.HasPrefix(strings.ToLower(modelName), prefix) {
				candidates = append(candidates, providerCandidate{
					factory:  factory,
					metadata: metadata,
				})
				config.DebugLog("[Registry] Provider %s matches model %s (prefix: %s)",
					metadata.Name, modelName, prefix)
				break
			}
		}
	}

	// Sort by priority (higher priority first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].metadata.Priority > candidates[j].metadata.Priority
	})

	// Return first match
	if len(candidates) > 0 {
		selected := candidates[0]
		config.DebugLog("[Registry] Selected provider %s for model %s (priority: %d)",
			selected.metadata.Name, modelName, selected.metadata.Priority)
		return selected.factory.CreateProvider()
	}

	config.DebugLog("[Registry] No provider found for model %s", modelName)
	return nil
}

// GetAvailableProviders returns list of registered providers
func GetAvailableProviders() []ProviderMetadata {
	registry.mutex.RLock()
	defer registry.mutex.RUnlock()

	var providers []ProviderMetadata
	for _, factory := range registry.factories {
		providers = append(providers, factory.GetMetadata())
	}

	// Sort by priority for consistent ordering
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].Priority > providers[j].Priority
	})

	return providers
}

// GetProviderByName returns a specific provider by name
func GetProviderByName(name string) Provider {
	registry.mutex.RLock()
	defer registry.mutex.RUnlock()

	if factory, exists := registry.factories[name]; exists {
		return factory.CreateProvider()
	}
	return nil
}

// ListRegisteredProviders returns names of all registered providers
func ListRegisteredProviders() []string {
	registry.mutex.RLock()
	defer registry.mutex.RUnlock()

	var names []string
	for name := range registry.factories {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}
