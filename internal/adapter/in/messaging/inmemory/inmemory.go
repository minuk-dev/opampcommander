// Package inmemory implements in-memory messaging adapters for standalone mode.
package inmemory

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/minuk-dev/opampcommander/internal/domain/agent/model/serverevent"
	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
)

var (
	_ agentport.ServerEventSenderPort   = (*EventSenderAdapter)(nil)
	_ agentport.ServerEventReceiverPort = (*EventSenderAdapter)(nil)
)

// EventSenderAdapter implements agentport.ServerEventSenderPort and agentport.ServerEventReceiverPort
// using in-memory no-op behavior.
// This adapter is used when event communication is disabled for standalone server mode.
type EventSenderAdapter struct {
	messageCh chan *serverevent.Message
	logger    *slog.Logger
}

// NewEventHubAdapter creates a new EventSenderAdapter.
func NewEventHubAdapter(
	logger *slog.Logger,
) *EventSenderAdapter {
	return &EventSenderAdapter{
		messageCh: make(chan *serverevent.Message, 1),
		logger:    logger,
	}
}

// SendMessageToServer implements agentport.ServerEventSenderPort.
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

// StartReceiver implements agentport.ServerEventReceiverPort.
func (e *EventSenderAdapter) StartReceiver(ctx context.Context, handler agentport.ReceiveServerEventHandler) error {
	for {
		select {
		case msg := <-e.messageCh:
			err := handler(ctx, msg)
			if err != nil {
				e.logger.Warn("failed to handle received message",
					slog.String("messageType", msg.Type.String()),
					slog.String("error", err.Error()),
				)
			}
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		}
	}
}
