package forwarder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/shaharia-lab/telemetry-forwarder/internal/config"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// TelemetryCollectHandler handles incoming telemetry events
func TelemetryCollectHandler(config *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read body
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Parse telemetry event
		var event OTelEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, "Invalid telemetry event format", http.StatusBadRequest)
			return
		}

		if event.TimeUnixNano == 0 {
			event.TimeUnixNano = time.Now().UnixNano()
		}

		go forwardToHoneycomb(config, event)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "accepted"}); err != nil {
			log.Printf("Failed to write JSON response: %v", err)
		}
	}
}

func forwardToHoneycomb(config *config.Config, event OTelEvent) {
	if config.HoneycombAPIKey == "" {
		log.Println("Skipping Honeycomb forwarding: API key not set")
		log.Printf("Event received: %s", event.Name)
		return
	}

	eventData := map[string]interface{}{
		"event_name": event.Name,
	}

	if event.Attributes != nil {
		for k, v := range event.Attributes {
			eventData[k] = v
		}
	}

	if event.Resource != nil {
		for k, v := range event.Resource {
			eventData["resource."+k] = v
		}
	}

	if event.Body != nil {
		eventData["body"] = event.Body
	}
	if event.SeverityText != "" {
		eventData["severity_text"] = event.SeverityText
	}
	if event.SeverityNumber != 0 {
		eventData["severity_number"] = event.SeverityNumber
	}
	if event.TraceID != "" {
		eventData["trace.trace_id"] = event.TraceID
	}
	if event.SpanID != "" {
		eventData["trace.span_id"] = event.SpanID
	}
	if event.DroppedAttributesCount > 0 {
		eventData["dropped_attributes_count"] = event.DroppedAttributesCount
	}

	timestamp := time.Unix(0, event.TimeUnixNano).UTC().Format(time.RFC3339Nano)

	honeycombEvent := HoneycombEvent{
		Data:      eventData,
		Timestamp: timestamp,
	}

	payload, err := json.Marshal(honeycombEvent)
	if err != nil {
		log.Printf("Failed to marshal Honeycomb event: %v", err)
		return
	}

	url := fmt.Sprintf("%s/1/events/%s", config.HoneycombAPIURL, config.HoneycombDataset)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("Failed to create Honeycomb request: %v", err)
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Honeycomb-Team", config.HoneycombAPIKey)

	// Send request
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send event to Honeycomb: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Printf("Honeycomb API returned error status: %d", resp.StatusCode)
		respBody, _ := ioutil.ReadAll(resp.Body)
		log.Printf("Response body: %s", string(respBody))
		return
	}

	log.Printf("Successfully forwarded event '%s' to Honeycomb", event.Name)
}
