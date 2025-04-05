package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/shaharia-lab/telemetry-forwarder/internal/config"
	http2 "github.com/shaharia-lab/telemetry-forwarder/internal/http"
	"github.com/shaharia-lab/telemetry-forwarder/internal/types"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"time"
)

// SharedHTTP is a shared HTTP client for sending telemetry data.
var SharedHTTP = &http2.Client{}

// HoneycombProvider is a provider for sending telemetry data to Honeycomb.
type HoneycombProvider struct {
	config         *config.Config
	circuitBreaker *http2.CircuitBreaker
}

// NewHoneycombProvider creates a new HoneycombProvider with the given configuration.
func NewHoneycombProvider(config *config.Config) *HoneycombProvider {
	return &HoneycombProvider{
		config:         config,
		circuitBreaker: http2.NewCircuitBreaker("Honeycomb", 5, 1*time.Minute),
	}
}

// Name returns the name of the provider.
func (h *HoneycombProvider) Name() string {
	return "Honeycomb"
}

// IsEnabled checks if the provider is enabled based on the configuration.
func (h *HoneycombProvider) IsEnabled() bool {
	return h.config.HoneycombAPIKey != "" && h.config.HoneycombAPIURL != ""
}

// Send sends the telemetry event to Honeycomb.
func (h *HoneycombProvider) Send(ctx context.Context, event types.OTelEvent) error {
	if !h.IsEnabled() {
		return fmt.Errorf("honeycomb provider not configured")
	}

	if !h.circuitBreaker.IsAllowed() {
		return fmt.Errorf("circuit open for %s provider", h.Name())
	}

	payload, err := json.Marshal(event.Prepare())
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	url := fmt.Sprintf("%s/1/events/%s", h.config.HoneycombAPIURL, h.config.HoneycombDataset)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Honeycomb-Team", h.config.HoneycombAPIKey)

	client := SharedHTTP.Client()
	success := false

	for attempt := 0; attempt < 3; attempt++ {
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Attempt %d: Failed to send to Honeycomb: %v", attempt+1, err)

			if ctx.Err() != nil {
				return ctx.Err()
			}

			if attempt < 2 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Duration(math.Pow(2, float64(attempt))) * time.Second):
				}
			}
			continue
		}

		func() {
			defer resp.Body.Close()

			if resp.StatusCode < 300 {
				success = true
				return
			}

			respBody, _ := ioutil.ReadAll(resp.Body)
			log.Printf("Honeycomb API error (attempt %d): status=%d, body=%s",
				attempt+1, resp.StatusCode, string(respBody))

			if resp.StatusCode >= 400 && resp.StatusCode < 500 {
				err = fmt.Errorf("client error from Honeycomb API: %d", resp.StatusCode)
			}
		}()

		if success || (resp.StatusCode >= 400 && resp.StatusCode < 500) {
			break
		}

		if attempt < 2 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(math.Pow(2, float64(attempt))) * time.Second):
			}
		}
	}

	if success {
		h.circuitBreaker.RecordSuccess()
		return nil
	} else {
		h.circuitBreaker.RecordFailure()
		return fmt.Errorf("failed to send to Honeycomb after multiple attempts")
	}
}
