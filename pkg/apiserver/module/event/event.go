// Package event provides event handling functionalities.
package event

import (
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/messaging/inmemory"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/messaging/nats"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

// New creates the event module.
func New() fx.Option {
	return fx.Module(
		"EventModule",
		// Provide NATS components (may return nil when disabled)
		fx.Provide(
			NewEventReceiverOptional,
			NewEventSenderOptional,
		),
		// Provide the appropriate event sender adapter
		fx.Provide(
			fx.Annotate(
				NewEventSenderAdapter,
				fx.As(new(port.ServerEventSenderPort)),
			),
		),
	)
}

// NewEventSenderAdapter creates the appropriate event sender adapter based on configuration.
//
//nolint:ireturn // Factory function that returns different implementations based on config.
func NewEventSenderAdapter(
	settings *config.EventSettings,
	lifecycle fx.Lifecycle,
) port.ServerEventSenderPort {
	if !settings.Enabled {
		// Use in-memory adapter for standalone mode
		return inmemory.NewEventSenderAdapter()
	}

	// Create NATS sender
	sender, err := NewEventSender(settings, lifecycle)
	if err != nil {
		// If NATS connection fails, fall back to in-memory adapter
		// This allows graceful degradation
		return inmemory.NewEventSenderAdapter()
	}

	// Use NATS adapter when events are enabled
	return nats.NewEventSenderAdapter(sender)
}
