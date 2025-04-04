package forwarder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/shaharia-lab/telemetry-forwarder/internal/config"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"sync"
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

const (
	circuitClosed = iota
	circuitOpen
	circuitHalfOpen
)

type circuitBreaker struct {
	state       int
	failCount   int
	lastFailure time.Time
	mutex       sync.Mutex
	timeout     time.Duration
	maxFailures int
}

var honeycombCircuit = &circuitBreaker{
	state:       circuitClosed,
	maxFailures: 5,
	timeout:     1 * time.Minute,
}

func (cb *circuitBreaker) isAllowed() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case circuitClosed:
		return true
	case circuitOpen:
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.state = circuitHalfOpen
			log.Println("Honeycomb circuit half-open: testing API availability")
			return true
		}
		return false
	case circuitHalfOpen:
		return true
	default:
		return true
	}
}

func (cb *circuitBreaker) recordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if cb.state == circuitHalfOpen {
		cb.state = circuitClosed
		cb.failCount = 0
		log.Println("Honeycomb circuit closed: API is back to normal")
	}
}

func (cb *circuitBreaker) recordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.lastFailure = time.Now()

	switch cb.state {
	case circuitClosed:
		cb.failCount++
		if cb.failCount >= cb.maxFailures {
			cb.state = circuitOpen
			log.Println("Honeycomb circuit open: stopping requests temporarily")
		}
	case circuitHalfOpen:
		cb.state = circuitOpen
		log.Println("Honeycomb circuit reopened: API still having issues")
	}
}

func forwardToHoneycomb(config *config.Config, event OTelEvent) {
	if config.HoneycombAPIKey == "" {
		log.Println("Skipping Honeycomb forwarding: API key not set")
		log.Printf("Event received: %s", event.Name)
		return
	}

	if !honeycombCircuit.isAllowed() {
		log.Printf("Circuit open, skipping event '%s' to Honeycomb", event.Name)
		return
	}

	eventData := map[string]interface{}{
		"name": event.Name,                                                      // OTel standard field
		"time": time.Unix(0, event.TimeUnixNano).UTC().Format(time.RFC3339Nano), // Standard timestamp format
	}

	if event.Resource != nil {
		for k, v := range event.Resource {
			eventData[k] = v
		}
	}

	if event.Attributes != nil {
		for k, v := range event.Attributes {
			eventData[k] = v
		}
	}

	if event.Body != nil {
		eventData["body"] = event.Body
	}
	if event.SeverityText != "" {
		eventData["severity.text"] = event.SeverityText
	}
	if event.SeverityNumber != 0 {
		eventData["severity.number"] = event.SeverityNumber
	}
	if event.TraceID != "" {
		eventData["trace_id"] = event.TraceID
	}
	if event.SpanID != "" {
		eventData["span_id"] = event.SpanID
	}
	if event.DroppedAttributesCount > 0 {
		eventData["dropped_attributes_count"] = event.DroppedAttributesCount
	}

	payload, err := json.Marshal(eventData)
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

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Honeycomb-Team", config.HoneycombAPIKey)

	maxRetries := 3
	success := false

	client := &http.Client{Timeout: 15 * time.Second}

	for attempt := 0; attempt < maxRetries; attempt++ {
		var resp *http.Response
		resp, err = client.Do(req)

		if err != nil {
			log.Printf("Attempt %d: Failed to send event to Honeycomb: %v", attempt+1, err)
		} else {
			defer func(resp *http.Response) {
				if resp != nil && resp.Body != nil {
					resp.Body.Close()
				}
			}(resp)

			if resp.StatusCode < 300 {
				success = true
				log.Printf("Successfully forwarded event '%s' to Honeycomb", event.Name)
				break
			}

			respBody, readErr := ioutil.ReadAll(resp.Body)
			if readErr != nil {
				log.Printf("Attempt %d: Error reading response body: %v", attempt+1, readErr)
			} else {
				log.Printf("Attempt %d: Honeycomb API returned error status: %d, body: %s",
					attempt+1, resp.StatusCode, string(respBody))
			}

			if resp.StatusCode >= 400 && resp.StatusCode < 500 {
				break
			}
		}

		if attempt < maxRetries-1 {
			backoffTime := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			time.Sleep(backoffTime)
		}
	}

	if success {
		honeycombCircuit.recordSuccess()
	} else {
		honeycombCircuit.recordFailure()
	}
}
