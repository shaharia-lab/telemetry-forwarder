package provider

import (
	"github.com/shaharia-lab/telemetry-forwarder/internal/config"
	"sync"
)

type ProviderRegistry struct {
	providers map[string]Provider
	mu        sync.RWMutex
}

func NewProviderRegistry(cfg *config.Config) *ProviderRegistry {
	registry := &ProviderRegistry{
		providers: make(map[string]Provider),
	}

	registry.Register(NewHoneycombProvider(cfg))

	return registry
}

func (r *ProviderRegistry) Register(provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[provider.Name()] = provider
}

func (r *ProviderRegistry) Get(name string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	return p, ok
}

func (r *ProviderRegistry) GetAll() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Provider, 0, len(r.providers))
	for _, prv := range r.providers {
		result = append(result, prv)
	}
	return result
}
