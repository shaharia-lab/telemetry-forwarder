// File: internal/handler/telemetry_handler.go
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shaharia-lab/telemetry-forwarder/internal/event"
	"github.com/shaharia-lab/telemetry-forwarder/internal/types"
)

// TelemetryCollectHandler handles incoming telemetry events and forwards them to the event processor.
func TelemetryCollectHandler(eventProcessor *event.EventProcessor) http.HandlerFunc {
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

		// Enqueue the event for processing
		eventProcessor.EnqueueEvent(event)

		// Return immediately, don't wait for processing to complete
		w.WriteHeader(http.StatusAccepted)
	}
}
