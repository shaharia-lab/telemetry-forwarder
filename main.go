package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shaharia-lab/telemetry-forwarder/internal/config"
	"github.com/shaharia-lab/telemetry-forwarder/internal/event"
	"github.com/shaharia-lab/telemetry-forwarder/internal/handler"
	"github.com/shaharia-lab/telemetry-forwarder/internal/middleware"
	"github.com/shaharia-lab/telemetry-forwarder/internal/provider"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", handlePing)

	gcpLogging, err := provider.NewGCPLoggingProvider(context.Background(), cfg)
	if err != nil {
		log.Fatalf("Failed to create GCP Logging provider: %v", err)
	}

	providerRegistry := provider.NewProviderRegistry(cfg)
	providerRegistry.Register(gcpLogging)

	eventProcessor := event.NewEventProcessor(providerRegistry, 100) // Buffer size of 100 events
	eventProcessor.Start()

	mux.HandleFunc("/telemetry/event", middleware.CORS(handler.TelemetryCollectHandler(eventProcessor)))

	addr := fmt.Sprintf(":%s", cfg.HTTPAPIPort)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("Starting telemetry server on %s", addr)
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatalf("Error starting server: %v", err)

	case <-shutdown:
		log.Println("Shutdown signal received, gracefully shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		log.Println("Shutting down HTTP server...")
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Error during server shutdown: %v", err)
			server.Close()
		}

		log.Println("Shutting down event processor...")
		if err := eventProcessor.Shutdown(); err != nil {
			log.Printf("Error shutting down event processor: %v", err)
		}

		log.Println("Shutting down providers...")
		if err := providerRegistry.Shutdown(); err != nil {
			log.Printf("Error shutting down providers: %v", err)
		}

		log.Println("Server gracefully stopped")
	}
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}
