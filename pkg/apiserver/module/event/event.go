// Package event provides event handling functionalities.
package event

import (
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/messaging/nats"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

// NewModule creates the event module.
func NewModule() fx.Option {
	return fx.Module(
		"EventModule",
		// nats
		fx.Provide(
			NewEventReceiver,
			NewEventSender,
			fx.Annotate(nats.NewEventSenderAdapter, fx.As(new(port.ServerMessageForServerToAgent))),
		),
	)
}

// EventSettings represents the event settings.
// TODO: Move config to pkg/apiserver/config
// and add appconfig provider for it with cmd options.
type EventSettings struct {
	// ProtocolType is the event protocol type.
	ProtocolType EventProtocolType

	// NATS settings. Used when ProtocolType is EventProtocolTypeNATS.
	NATS NATSSettings
}

// NATSSettings represents the NATS event settings.
type NATSSettings struct {
	Endpoint      string
	SubjectPrefix string // e.g. "prod.opampcommander."
}

// EventProtocolType represents the type of event protocol.
type EventProtocolType string

const (
	// EventProtocolTypeNATS represents the NATS event protocol.
	EventProtocolTypeNATS EventProtocolType = "nats"
	// TODO: Add kafka protocol support.
	//nolint:godox // too far future
)
