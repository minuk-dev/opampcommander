package agentport

import (
	"context"
	"time"
)

// ConnectionShutdownUsecase closes the live connections owned by this server.
type ConnectionShutdownUsecase interface {
	CloseLocalConnections(ctx context.Context, window time.Duration) error
}

// ConnectionTransportPort controls admission and closes one live transport.
type ConnectionTransportPort interface {
	// StopAccepting rejects new upgrades and waits for accepted handshakes to be
	// registered in ConnectionStore. At the deadline it closes pending handshakes.
	StopAccepting(ctx context.Context) error
	// CloseConnection sends a graceful close and waits for the peer, force-closing
	// the transport when ctx expires. id is the opaque Connection.ID from the Store.
	CloseConnection(ctx context.Context, id any) error
}
