package forwarder

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
