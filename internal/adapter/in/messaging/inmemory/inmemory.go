// Package inmemory implements in-memory messaging adapters for standalone mode.
package inmemory

import (
	"context"
	"fmt"

	"github.com/minuk-dev/opampcommander/internal/domain/model/serverevent"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var (
	_ port.ServerEventSenderPort   = (*EventSenderAdapter)(nil)
	_ port.ServerEventReceiverPort = (*EventSenderAdapter)(nil)
)

// EventSenderAdapter implements port.ServerEventSenderPort and port.ServerEventReceiverPort
// using in-memory no-op behavior.
// This adapter is used when event communication is disabled for standalone server mode.
type EventSenderAdapter struct {
	messageCh chan *serverevent.Message
}

// NewEventHubAdapter creates a new EventSenderAdapter.
func NewEventHubAdapter() *EventSenderAdapter {
	return &EventSenderAdapter{
		messageCh: make(chan *serverevent.Message, 1),
	}
}

// SendMessageToServer implements port.ServerEventSenderPort.
// In standalone mode, this is a no-op as there are no other servers to communicate with.
func (e *EventSenderAdapter) SendMessageToServer(
	ctx context.Context,
	_ string, // meaningless serverID in standalone mode
	msg serverevent.Message,
) error {
	select {
	case e.messageCh <- &msg:
		// message sent to channel
		return nil
	case <-ctx.Done():
		// context cancelled
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	}
}

// ReceiveMessageFromServer implements port.ServerEventReceiverPort.
// In standalone mode, this blocks forever as there are no messages to receive.
func (e *EventSenderAdapter) ReceiveMessageFromServer(ctx context.Context) (*serverevent.Message, error) {
	select {
	case msg := <-e.messageCh:
		return msg, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
	}
}
