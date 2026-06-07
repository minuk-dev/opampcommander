// Package common provides shared adapter infrastructure that is used by both
// primary (inbound) and secondary (outbound) adapters.
//
// Messaging lives here because a single transport (the in-memory event hub, or
// a Kafka sender/receiver pair) backs both the inbound ServerEventReceiverPort
// and the outbound ServerEventSenderPort. It cannot be cleanly attributed to a
// single direction, so it is shared.
package common

import (
	"fmt"
	"log/slog"

	"go.uber.org/fx"

	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

// New creates the common adapter module.
func New() fx.Option {
	return fx.Options(
		fx.Provide(newEventSenderAndReceiver),
	)
}

// UnsupportedEventProtocolError is returned when an unsupported event protocol type is specified.
type UnsupportedEventProtocolError struct {
	ProtocolType string
}

// Error implements the error interface.
func (e *UnsupportedEventProtocolError) Error() string {
	return "unsupported event protocol type: " + e.ProtocolType
}

// newEventSenderAndReceiver provides the messaging sender/receiver pair, selecting
// the transport based on the configured event protocol.
func newEventSenderAndReceiver(
	settings *config.EventSettings,
	serverID config.ServerID,
	logger *slog.Logger,
	lifecycle fx.Lifecycle,
	serverIdentityProvider agentport.ServerIdentityProvider,
) (agentport.ServerEventSenderPort, agentport.ServerEventReceiverPort, error) {
	switch settings.ProtocolType {
	case config.EventProtocolTypeKafka:
		sender, receiver, err := newKafkaSenderAndReceiver(
			settings, serverID, logger, lifecycle, serverIdentityProvider,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create Kafka messaging: %w", err)
		}

		return sender, receiver, nil
	case config.EventProtocolTypeInMemory:
		return newInMemorySenderAndReceiver(logger)
	}

	return nil, nil, &UnsupportedEventProtocolError{
		ProtocolType: settings.ProtocolType.String(),
	}
}
