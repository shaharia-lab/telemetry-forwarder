package provider

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/logging"
	"github.com/shaharia-lab/telemetry-forwarder/internal/config"
	"github.com/shaharia-lab/telemetry-forwarder/internal/types"
)

// GCPLoggingProvider sends telemetry events to Google Cloud Logging.
type GCPLoggingProvider struct {
	config    *config.Config
	client    *logging.Client
	logger    *logging.Logger // Logger instance for the specified log name
	projectID string
	logName   string
}

// NewGCPLoggingProvider creates a new provider for GCP Cloud Logging.
// It uses Application Default Credentials (ADC) by default.
func NewGCPLoggingProvider(ctx context.Context, cfg *config.Config) (*GCPLoggingProvider, error) {
	if cfg.GCPLogName == "" || cfg.GCPProjectID == "" {
		return nil, fmt.Errorf("GCP Logging provider is not enabled or project ID is not set")
	}

	// ADC is used by default if no explicit credentials option is provided.
	// You could add option.WithCredentialsFile(path) if needed, but avoid for Cloud Run.
	client, err := logging.NewClient(ctx, fmt.Sprintf("projects/%s", cfg.GCPProjectID))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP Logging client: %w", err)
	}

	client.OnError = func(err error) {
		// This will log errors happening during background sends (permissions, network, etc.)
		log.Printf("!!! GCP Logging Background Error: %v", err)
	}

	log.Printf("GCP Logging provider initialized for project %s, log name %s", cfg.GCPProjectID, cfg.GCPLogName)

	return &GCPLoggingProvider{
		config:    cfg,
		client:    client,
		logger:    client.Logger(cfg.GCPLogName),
		projectID: cfg.GCPProjectID,
		logName:   cfg.GCPLogName,
	}, nil
}

// Name returns the name of the provider.
func (g *GCPLoggingProvider) Name() string {
	return "GCP Cloud Logging"
}

// IsEnabled checks if the provider is configured and the client was initialized successfully.
func (g *GCPLoggingProvider) IsEnabled() bool {
	return g.config.GCPProjectID != "" && g.config.GCPLogName != "" && g.client != nil
}

// Send converts the OTelEvent to a Cloud Logging Entry and sends it.
func (g *GCPLoggingProvider) Send(ctx context.Context, event types.OTelEvent) error {
	if !g.IsEnabled() {
		return fmt.Errorf("GCP Logging provider is not enabled or failed to initialize")
	}

	payload := event.Prepare()

	entry := logging.Entry{
		Timestamp: time.Now().UTC(),
		Severity:  mapSeverity(event.SeverityText, event.SeverityNumber),
		Payload:   payload,
	}

	if event.TraceID != "" {
		entry.Trace = fmt.Sprintf("projects/%s/traces/%s", g.projectID, event.TraceID)
	}
	if event.SpanID != "" {
		entry.SpanID = event.SpanID
	}

	g.logger.Log(entry)

	return nil
}

// Close flushes any buffered logs and closes the client. Should be called on application shutdown.
func (g *GCPLoggingProvider) Close() error {
	if g.client == nil {
		return nil
	}
	log.Printf("Flushing and closing GCP Logging client for log %s...", g.logName)
	err := g.client.Close()
	if err != nil {
		log.Printf("Error closing GCP Logging client: %v", err)
		return fmt.Errorf("failed to close GCP Logging client: %w", err)
	}
	log.Printf("GCP Logging client closed for log %s.", g.logName)
	return nil
}

// mapSeverity converts OTel severity information to Google Cloud Logging Severity.
func mapSeverity(severityText string, severityNumber int) logging.Severity {
	lowerSeverityText := strings.ToLower(severityText)

	switch {
	case lowerSeverityText == "trace":
		return logging.Debug
	case lowerSeverityText == "debug":
		return logging.Debug
	case lowerSeverityText == "info", lowerSeverityText == "information":
		return logging.Info
	case lowerSeverityText == "warn", lowerSeverityText == "warning":
		return logging.Warning
	case lowerSeverityText == "error":
		return logging.Error
	case lowerSeverityText == "fatal", lowerSeverityText == "critical":
		return logging.Critical

	case severityNumber >= 1 && severityNumber <= 4:
		return logging.Debug
	case severityNumber >= 5 && severityNumber <= 8:
		return logging.Debug
	case severityNumber >= 9 && severityNumber <= 12:
		return logging.Info
	case severityNumber >= 13 && severityNumber <= 16:
		return logging.Warning
	case severityNumber >= 17 && severityNumber <= 20:
		return logging.Error
	case severityNumber >= 21 && severityNumber <= 24:
		return logging.Critical

	default:
		return logging.Default
	}
}
