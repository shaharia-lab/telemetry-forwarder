package config

import (
	"github.com/kelseyhightower/envconfig"
)

// Config holds the configuration for the telemetry forwarder
type Config struct {
	HTTPAPIPort      string `envconfig:"HTTP_API_PORT" default:"8080"`
	HoneycombAPIKey  string `envconfig:"HONEYCOMB_API_KEY"`
	HoneycombDataset string `envconfig:"HONEYCOMB_DATASET" default:"cli-telemetry"`
	HoneycombAPIURL  string `envconfig:"HONEYCOMB_API_URL"`

	GCPProjectID string `envconfig:"GCP_PROJECT_ID"`
	GCPLogName   string `envconfig:"GCP_LOG_NAME" default:"telemetry-forwarder-events"`
}

// Load loads the configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}
	err := envconfig.Process("", cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
