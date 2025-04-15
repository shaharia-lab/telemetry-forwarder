package main

import (
	"context"
	"fmt"
	"github.com/shaharia-lab/telemetry-forwarder/internal/config"
	"github.com/shaharia-lab/telemetry-forwarder/internal/handler"
	"github.com/shaharia-lab/telemetry-forwarder/internal/middleware"
	"github.com/shaharia-lab/telemetry-forwarder/internal/provider"
	"log"
	"net/http"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	http.HandleFunc("/ping", handlePing)

	//registry.Register(NewHoneycombProvider(cfg))
	gcpLogging, err := provider.NewGCPLoggingProvider(context.Background(), cfg)
	if err != nil {
		log.Fatalf("Failed to create GCP Logging provider: %v", err)
	}

	providerRegistry := provider.NewProviderRegistry(cfg)
	providerRegistry.Register(gcpLogging)

	http.HandleFunc("/telemetry/event", middleware.CORS(handler.TelemetryCollectHandler(providerRegistry)))

	addr := fmt.Sprintf(":%s", cfg.HTTPAPIPort)
	log.Printf("Starting telemetry server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}
