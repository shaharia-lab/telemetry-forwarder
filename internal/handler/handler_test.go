package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/shaharia-lab/telemetry-forwarder/internal/config"
	"github.com/shaharia-lab/telemetry-forwarder/internal/provider"
	"github.com/shaharia-lab/telemetry-forwarder/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"testing"
)

// MockProvider implements the provider.Provider interface for testing
type MockProvider struct {
	mock.Mock
	enabled bool
	name    string
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) IsEnabled() bool {
	return m.enabled
}

func (m *MockProvider) Send(ctx context.Context, event types.OTelEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func TestTelemetryCollectHandler_Success(t *testing.T) {
	testCases := []struct {
		name           string
		event          types.OTelEvent
		providers      []provider.Provider
		expectedStatus int
	}{
		{
			name: "successful event processing",
			event: types.OTelEvent{
				Name: "test_event",
				Attributes: map[string]interface{}{
					"key1": "value1",
					"key2": "value2",
				},
			},
			providers: func() []provider.Provider {
				mockProvider1 := &MockProvider{enabled: true, name: "provider1"}
				mockProvider1.On("Send", mock.Anything, mock.Anything).Return(nil)

				mockProvider2 := &MockProvider{enabled: true, name: "provider2"}
				mockProvider2.On("Send", mock.Anything, mock.Anything).Return(nil)

				mockDisabledProvider := &MockProvider{enabled: false, name: "disabled-provider"}
				return []provider.Provider{mockProvider1, mockProvider2, mockDisabledProvider}
			}(),
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := provider.NewProviderRegistry(&config.Config{})

			for _, p := range tc.providers {
				registry.Register(p)
			}

			handler := TelemetryCollectHandler(registry)

			eventJSON, err := json.Marshal(tc.event)
			assert.NoError(t, err)
			req, err := http.NewRequest(http.MethodPost, "/collect", bytes.NewReader(eventJSON))
			assert.NoError(t, err)

			w := httptest.NewRecorder()

			handler(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			for _, p := range tc.providers {
				mockP := p.(*MockProvider)
				if mockP.IsEnabled() {
					mockP.AssertCalled(t, "Send", mock.Anything, tc.event)
				} else {
					mockP.AssertNotCalled(t, "Send", mock.Anything, mock.Anything)
				}
			}
		})
	}
}
