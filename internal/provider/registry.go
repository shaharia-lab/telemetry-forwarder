package provider

import (
	"github.com/shaharia-lab/telemetry-forwarder/internal/config"
	"log"
	"sync"
)

type Registry struct {
	providers map[string]Provider
	mu        sync.RWMutex
}

func NewProviderRegistry(cfg *config.Config) *Registry {
	registry := &Registry{
		providers: make(map[string]Provider),
	}

	return registry
}

func (r *Registry) Register(provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[provider.Name()] = provider
}

func (r *Registry) Get(name string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	return p, ok
}

func (r *Registry) GetAll() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Provider, 0, len(r.providers))
	for _, prv := range r.providers {
		result = append(result, prv)
	}
	return result
}

func (r *Registry) Shutdown() error {
	var lastErr error
	for _, p := range r.providers {
		if err := p.Close(); err != nil {
			lastErr = err
			log.Printf("Error closing provider %s: %v", p.Name(), err)
		}
	}
	return lastErr
}
