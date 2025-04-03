package forwarder

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shaharia-lab/telemetry-forwarder/internal/config"
)

func TestTelemetryCollectHandler(t *testing.T) {
	cfg := &config.Config{
		HoneycombAPIKey:  "test-key",
		HoneycombDataset: "test-dataset",
		HoneycombAPIURL:  "http://localhost:8080",
	}

	tests := []struct {
		name           string
		method         string
		requestBody    OTelEvent
		expectedStatus int
	}{
		{
			name:   "Valid telemetry event",
			method: http.MethodPost,
			requestBody: OTelEvent{
				Name: "test-event",
				Attributes: map[string]interface{}{
					"key": "value",
				},
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "Wrong HTTP method",
			method:         http.MethodGet,
			requestBody:    OTelEvent{},
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request body
			bodyBytes, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			// Create request
			req, err := http.NewRequest(tt.method, "/collect", bytes.NewBuffer(bodyBytes))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute the handler
			handler := TelemetryCollectHandler(cfg)
			handler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, rr.Code)
			}

			// For successful requests, verify response body
			if tt.expectedStatus == http.StatusAccepted {
				var response map[string]string
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if status, exists := response["status"]; !exists || status != "accepted" {
					t.Errorf("Expected status 'accepted', got '%s'", status)
				}
			}
		})
	}
}

func TestForwardToHoneycomb(t *testing.T) {
	mockHoneycomb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type header to be application/json, got %s", r.Header.Get("Content-Type"))
		}

		if r.Header.Get("X-Honeycomb-Team") != "test-key" {
			t.Errorf("Expected X-Honeycomb-Team header to be test-key, got %s", r.Header.Get("X-Honeycomb-Team"))
		}

		if !strings.Contains(r.URL.Path, "test-dataset") {
			t.Errorf("Expected URL to contain test-dataset, got %s", r.URL.Path)
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}

		var event HoneycombEvent
		if err := json.Unmarshal(body, &event); err != nil {
			t.Fatalf("Failed to unmarshal request body: %v", err)
		}

		if event.Data["event_name"] != "test-event" {
			t.Errorf("Expected event_name to be test-event, got %v", event.Data["event_name"])
		}

		if event.Data["test_attr"] != "test-value" {
			t.Errorf("Expected test_attr to be test-value, got %v", event.Data["test_attr"])
		}

		if event.Data["resource.service.name"] != "test-service" {
			t.Errorf("Expected resource.service.name to be test-service, got %v", event.Data["resource.service.name"])
		}

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer mockHoneycomb.Close()

	cfg := &config.Config{
		HoneycombAPIKey:  "test-key",
		HoneycombDataset: "test-dataset",
		HoneycombAPIURL:  mockHoneycomb.URL,
	}
	tests := []struct {
		name           string
		event          OTelEvent
		mockStatusCode int
		expectError    bool
	}{
		{
			name: "Successful forwarding",
			event: OTelEvent{
				Name:         "test-event",
				TimeUnixNano: time.Now().UnixNano(),
				Attributes: map[string]interface{}{
					"test_attr": "test-value",
				},
				Resource: map[string]interface{}{
					"service.name": "test-service",
				},
				Body:                   "test body",
				SeverityText:           "INFO",
				SeverityNumber:         1,
				TraceID:                "trace123",
				SpanID:                 "span456",
				DroppedAttributesCount: 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, err := json.Marshal(tt.event)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			req, err := http.NewRequest(http.MethodPost, "/telemetry/event", bytes.NewBuffer(bodyBytes))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			handler := TelemetryCollectHandler(cfg)
			handler(rr, req)

			if rr.Code != http.StatusAccepted {
				t.Errorf("Expected status code %d, got %d", http.StatusAccepted, rr.Code)
			}

			time.Sleep(100 * time.Millisecond)
		})
	}
}

func TestForwardToHoneycombError(t *testing.T) {
	mockHoneycomb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"server error"}`))
	}))
	defer mockHoneycomb.Close()

	cfg := &config.Config{
		HoneycombAPIKey:  "test-key",
		HoneycombDataset: "test-dataset",
		HoneycombAPIURL:  mockHoneycomb.URL,
	}

	event := OTelEvent{
		Name:         "test-event",
		TimeUnixNano: time.Now().UnixNano(),
		Attributes: map[string]interface{}{
			"test_attr": "test-value",
		},
	}

	bodyBytes, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal request body: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "/telemetry/event", bytes.NewBuffer(bodyBytes))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	handler := TelemetryCollectHandler(cfg)
	handler(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("Expected status code %d, got %d", http.StatusAccepted, rr.Code)
	}

	time.Sleep(100 * time.Millisecond)
}

func TestNoHoneycombKey(t *testing.T) {
	cfg := &config.Config{
		HoneycombAPIKey:  "",
		HoneycombDataset: "test-dataset",
		HoneycombAPIURL:  "http://example.com",
	}

	event := OTelEvent{
		Name:         "test-event",
		TimeUnixNano: time.Now().UnixNano(),
	}

	bodyBytes, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal request body: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "/telemetry/event", bytes.NewBuffer(bodyBytes))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	handler := TelemetryCollectHandler(cfg)
	handler(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("Expected status code %d, got %d", http.StatusAccepted, rr.Code)
	}

	time.Sleep(100 * time.Millisecond)
}
