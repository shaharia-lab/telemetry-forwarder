package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shaharia-lab/telemetry-forwarder/internal/config"
	"github.com/shaharia-lab/telemetry-forwarder/internal/event"
	"github.com/shaharia-lab/telemetry-forwarder/internal/provider"
	"github.com/shaharia-lab/telemetry-forwarder/internal/types"
)

// MockProvider implements the provider.Provider interface for testing
type MockProvider struct {
	SentEvents []types.OTelEvent
}

func (mp *MockProvider) Send(ctx context.Context, event types.OTelEvent) error {
	mp.SentEvents = append(mp.SentEvents, event)
	return nil
}

func (mp *MockProvider) Name() string {
	return "mock-provider"
}

func (mp *MockProvider) IsEnabled() bool {
	return true
}

func (mp *MockProvider) Close() error {
	return nil
}

func TestTelemetryCollectHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		requestBody    interface{}
		expectedStatus int
		shouldEnqueue  bool
	}{
		{
			name:           "valid POST request",
			method:         http.MethodPost,
			requestBody:    types.OTelEvent{Name: "test-event", Attributes: map[string]interface{}{"key": "value"}},
			expectedStatus: http.StatusAccepted,
			shouldEnqueue:  true,
		},
		{
			name:           "invalid method",
			method:         http.MethodGet,
			requestBody:    nil,
			expectedStatus: http.StatusMethodNotAllowed,
			shouldEnqueue:  false,
		},
		{
			name:           "invalid request body",
			method:         http.MethodPost,
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			shouldEnqueue:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			registry := provider.NewProviderRegistry(cfg)

			mockProvider := &MockProvider{
				SentEvents: []types.OTelEvent{},
			}

			registry.Register(mockProvider)

			processor := event.NewEventProcessor(registry, 10)

			processor.Start()
			defer processor.Shutdown()

			handler := TelemetryCollectHandler(processor)

			var body []byte
			var err error

			if tt.requestBody != nil {
				switch v := tt.requestBody.(type) {
				case string:
					body = []byte(v)
				default:
					body, err = json.Marshal(tt.requestBody)
					if err != nil {
						t.Fatalf("Failed to marshal request body: %v", err)
					}
				}
			}

			req := httptest.NewRequest(tt.method, "/collect", bytes.NewReader(body))
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, rec.Code)
			}

			time.Sleep(50 * time.Millisecond)

			if tt.shouldEnqueue && len(mockProvider.SentEvents) != 1 {
				t.Errorf("Expected event to be processed, but it wasn't")
			} else if !tt.shouldEnqueue && len(mockProvider.SentEvents) > 0 {
				t.Errorf("Expected no events to be processed, but got %d", len(mockProvider.SentEvents))
			}

			if tt.shouldEnqueue && len(mockProvider.SentEvents) == 1 {
				expectedEvt, ok := tt.requestBody.(types.OTelEvent)
				if !ok {
					t.Fatalf("Test case error: requestBody not of type OTelEvent")
				}

				actualEvt := mockProvider.SentEvents[0]
				if expectedEvt.Name != actualEvt.Name {
					t.Errorf("Expected event name %s, got %s", expectedEvt.Name, actualEvt.Name)
				}
			}
		})
	}
}
