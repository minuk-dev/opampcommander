package port

import (
	"context"

	"github.com/open-telemetry/opamp-go/protobufs"
)

// WebSocketRegistry manages active WebSocket connections.
// This is an infrastructure concern, separated from the domain Connection model.
type WebSocketRegistry interface {
	// Register registers a WebSocket connection with the given connection ID.
	Register(connID string, wsConn WebSocketConnection)

	// Get retrieves a WebSocket connection by connection ID.
	Get(connID string) (WebSocketConnection, bool)

	// Remove removes a WebSocket connection by connection ID.
	Remove(connID string)

	// GetByInstanceUID retrieves a WebSocket connection by agent instance UID.
	GetByInstanceUID(instanceUID string) (WebSocketConnection, bool)

	// UpdateInstanceUID updates the mapping between connection ID and instance UID.
	UpdateInstanceUID(connID string, instanceUID string)
}

// WebSocketConnection represents a WebSocket connection abstraction.
// This interface abstracts the OpAMP library's Connection type.
type WebSocketConnection interface {
	// Send sends a ServerToAgent message through the WebSocket connection.
	Send(ctx context.Context, message *protobufs.ServerToAgent) error

	// Close closes the WebSocket connection.
	Close() error
}
