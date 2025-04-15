// File: internal/handler/telemetry_handler.go
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shaharia-lab/telemetry-forwarder/internal/event"
	"github.com/shaharia-lab/telemetry-forwarder/internal/types"
)

// TelemetryCollectHandler handles incoming telemetry events and forwards them to the event processor.
func TelemetryCollectHandler(eventProcessor *event.Processor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var evt types.OTelEvent
		if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		eventProcessor.EnqueueEvent(evt)

		w.WriteHeader(http.StatusAccepted)
	}
}
