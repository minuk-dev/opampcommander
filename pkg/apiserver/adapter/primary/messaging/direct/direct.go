// Package direct implements the receiver side of the direct (peer-to-peer) server-event
// transport. It serves incoming targeted messages from peers over HTTP or gRPC and hands
// each one to the domain handler. Because only messages addressed to this server are ever
// delivered here, no broadcast filtering is needed.
package direct

import (
	"context"
	"fmt"
	"log/slog"

	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

var _ agentport.ServerEventReceiverPort = (*EventReceiverAdapter)(nil)

// Receiver serves a concrete wire protocol (HTTP or gRPC), blocking until the context
// is cancelled and dispatching each received message to the handler.
type Receiver interface {
	// Serve starts the transport server and blocks until ctx is cancelled.
	Serve(ctx context.Context, handler agentport.ReceiveServerEventHandler) error
}

// EventReceiverAdapter implements agentport.ServerEventReceiverPort using a direct
// peer-to-peer transport server.
type EventReceiverAdapter struct {
	receiver Receiver
	logger   *slog.Logger
}

// NewEventReceiverAdapter creates a new EventReceiverAdapter.
func NewEventReceiverAdapter(
	receiver Receiver,
	logger *slog.Logger,
) *EventReceiverAdapter {
	return &EventReceiverAdapter{
		receiver: receiver,
		logger:   logger,
	}
}

// StartReceiver implements agentport.ServerEventReceiverPort. It blocks until ctx is
// cancelled, matching the Kafka/in-memory receivers.
func (a *EventReceiverAdapter) StartReceiver(
	ctx context.Context,
	handler agentport.ReceiveServerEventHandler,
) error {
	err := a.receiver.Serve(ctx, handler)
	if err != nil {
		return fmt.Errorf("direct receiver stopped: %w", err)
	}

	return nil
}
