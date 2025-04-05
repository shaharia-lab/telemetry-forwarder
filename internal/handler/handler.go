package handler

import (
	"encoding/json"
	"github.com/shaharia-lab/telemetry-forwarder/internal/provider"
	"github.com/shaharia-lab/telemetry-forwarder/internal/types"
	"log"
	"net/http"
	"sync"
)

// TelemetryCollectHandler handles incoming telemetry events and forwards them to the configured providers.
func TelemetryCollectHandler(providerRegistry *provider.ProviderRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var event types.OTelEvent
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		var wg sync.WaitGroup
		for _, prv := range providerRegistry.GetAll() {
			if prv.IsEnabled() {
				wg.Add(1)
				go func(p provider.Provider) {
					defer wg.Done()
					if err := p.Send(r.Context(), event); err != nil {
						log.Printf("Error forwarding to %s: %v", p.Name(), err)
					}
				}(prv)
			}
		}
		wg.Wait()

		w.WriteHeader(http.StatusOK)
	}
}
