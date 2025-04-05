package provider

import (
	"context"
	"github.com/shaharia-lab/telemetry-forwarder/internal/types"
)

type Provider interface {
	Send(ctx context.Context, event types.OTelEvent) error
	Name() string
	IsEnabled() bool
}
