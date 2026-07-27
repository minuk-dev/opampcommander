// Package direct implements the sender side of the direct (peer-to-peer) server-event
// transport. Instead of publishing to a broker, it resolves the destination server's
// address from the server registry and delivers the message to that peer directly, so
// a targeted message reaches exactly one server (O(1) delivery).
package direct

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
)

var _ agentport.ServerEventSenderPort = (*EventSenderAdapter)(nil)

// ErrNoPeerAddress is returned when the target server has no direct address registered,
// so it cannot be reached over the direct transport.
var ErrNoPeerAddress = errors.New("target server has no direct address")

// PeerResolver resolves a server ID to its registry record, whose Address the sender
// dials. It is satisfied by the server persistence adapter.
type PeerResolver interface {
	// GetServer retrieves a server by its ID.
	GetServer(ctx context.Context, id string) (*agentmodel.Server, error)
}

// Deliverer sends a single message to a peer at the given address using a concrete
// wire protocol (HTTP or gRPC).
type Deliverer interface {
	io.Closer
	// Deliver sends the message to the peer listening at address.
	Deliver(ctx context.Context, address string, message serverevent.Message) error
}

// EventSenderAdapter implements agentport.ServerEventSenderPort using direct peer delivery.
type EventSenderAdapter struct {
	resolver  PeerResolver
	deliverer Deliverer
	logger    *slog.Logger
}

// NewEventSenderAdapter creates a new EventSenderAdapter.
func NewEventSenderAdapter(
	resolver PeerResolver,
	deliverer Deliverer,
	logger *slog.Logger,
) *EventSenderAdapter {
	return &EventSenderAdapter{
		resolver:  resolver,
		deliverer: deliverer,
		logger:    logger,
	}
}

// SendMessageToServer implements agentport.ServerEventSenderPort.
//
// Delivery is at-most-once: a failure to reach the peer is returned to the caller and
// not retried here. Losing an in-flight routed message is acceptable because the agent
// is re-driven on the next request, and each server also reconciles agent state
// periodically.
func (a *EventSenderAdapter) SendMessageToServer(
	ctx context.Context,
	serverID string,
	message serverevent.Message,
) error {
	server, err := a.resolver.GetServer(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to resolve peer %s: %w", serverID, err)
	}

	if server.Address == "" {
		return fmt.Errorf("%w: server %s", ErrNoPeerAddress, serverID)
	}

	err = a.deliverer.Deliver(ctx, server.Address, message)
	if err != nil {
		return fmt.Errorf("failed to deliver message to server %s at %s: %w", serverID, server.Address, err)
	}

	return nil
}

// Close releases resources held by the underlying deliverer.
func (a *EventSenderAdapter) Close() error {
	err := a.deliverer.Close()
	if err != nil {
		return fmt.Errorf("failed to close deliverer: %w", err)
	}

	return nil
}
