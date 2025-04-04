package handler

import (
	"encoding/json"
	"github.com/shaharia-lab/telemetry-forwarder/internal/config"
	"github.com/shaharia-lab/telemetry-forwarder/internal/provider"
	"github.com/shaharia-lab/telemetry-forwarder/internal/types"
	"log"
	"net/http"
	"sync"
)

func TelemetryCollectHandler(config *config.Config) http.HandlerFunc {
	registry := provider.NewProviderRegistry(config)

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
		for _, prv := range registry.GetAll() {
			if prv.IsEnabled() {
				wg.Add(1)
				go func(p provider.Provider) {
					defer wg.Done()
					if err := p.Send(r.Context(), event); err != nil {
						log.Printf("Error forwarding to %s: %v", p.Name(), err)
					}
					log.Printf("Successfully forwarded to %s", p.Name())
				}(prv)
			}
		}
		wg.Wait()

		w.WriteHeader(http.StatusOK)
	}
}
