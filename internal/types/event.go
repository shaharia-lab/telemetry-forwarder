package types

import "time"

// OTelEvent represents an OpenTelemetry event following the spec
type OTelEvent struct {
	Name                   string                 `json:"name"`
	TimeUnixNano           int64                  `json:"timeUnixNano,omitempty"`
	TraceID                string                 `json:"traceId,omitempty"`
	SpanID                 string                 `json:"spanId,omitempty"`
	SeverityText           string                 `json:"severityText,omitempty"`
	SeverityNumber         int                    `json:"severityNumber,omitempty"`
	Body                   interface{}            `json:"body,omitempty"`
	Attributes             map[string]interface{} `json:"attributes,omitempty"`
	DroppedAttributesCount int                    `json:"droppedAttributesCount,omitempty"`
	Resource               map[string]interface{} `json:"resource,omitempty"`
}

// Prepare converts the OTelEvent to a map for sending to telemetry services
func (event OTelEvent) Prepare() map[string]interface{} {
	eventData := map[string]interface{}{
		"name": event.Name,
		"time": time.Unix(0, event.TimeUnixNano).UTC().Format(time.RFC3339Nano),
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

	return eventData
}
