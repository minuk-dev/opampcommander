// Package event provides event handling functionalities.
package event

import (
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/messaging/nats"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

// New creates the event module.
func New() fx.Option {
	return fx.Module(
		"EventModule",
		// nats
		fx.Provide(
			NewEventReceiver,
			NewEventSender,
			fx.Annotate(nats.NewEventSenderAdapter, fx.As(new(port.ServerEventSenderPort))),
		),
	)
}
