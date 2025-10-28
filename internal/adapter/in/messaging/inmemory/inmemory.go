// Package inmemory implements in-memory messaging adapters for standalone mode.
package inmemory

import (
	"context"

	"github.com/minuk-dev/opampcommander/internal/domain/model/serverevent"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var (
	_ port.ServerEventSenderPort = (*EventSenderAdapter)(nil)
)

// EventSenderAdapter implements port.ServerEventSenderPort using in-memory no-op behavior.
// This adapter is used when event communication is disabled for standalone server mode.
type EventSenderAdapter struct{}

// NewEventSenderAdapter creates a new EventSenderAdapter.
func NewEventSenderAdapter() *EventSenderAdapter {
	return &EventSenderAdapter{}
}

// SendMessageToServer implements port.ServerEventSenderPort.
// In standalone mode, this is a no-op as there are no other servers to communicate with.
func (e *EventSenderAdapter) SendMessageToServer(
	_ context.Context,
	_ string,
	_ serverevent.Message,
) error {
	// No-op: in standalone mode, we don't send messages to other servers
	return nil
}
