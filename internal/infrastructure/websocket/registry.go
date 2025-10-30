// Package websocket provides infrastructure implementations for WebSocket connection management.
package websocket

import (
	"context"
	"sync"

	"github.com/open-telemetry/opamp-go/protobufs"
	opamptypes "github.com/open-telemetry/opamp-go/server/types"

	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ port.WebSocketRegistry = (*Registry)(nil)

// Registry is an in-memory implementation of WebSocketRegistry.
type Registry struct {
	mu sync.RWMutex

	// connections maps connection ID to WebSocket connection
	connections map[string]*OpAMPWebSocketConnection

	// instanceIndex maps agent instance UID to connection ID
	instanceIndex map[string]string
}

// NewRegistry creates a new WebSocket registry.
func NewRegistry() *Registry {
	return &Registry{
		connections:   make(map[string]*OpAMPWebSocketConnection),
		instanceIndex: make(map[string]string),
		mu:            sync.RWMutex{},
	}
}

// Register implements port.WebSocketRegistry.
func (r *Registry) Register(connID string, wsConn port.WebSocketConnection) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if opampConn, ok := wsConn.(*OpAMPWebSocketConnection); ok {
		r.connections[connID] = opampConn
	}
}

// Get implements port.WebSocketRegistry.
func (r *Registry) Get(connID string) (port.WebSocketConnection, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	conn, ok := r.connections[connID]

	return conn, ok
}

// Remove implements port.WebSocketRegistry.
func (r *Registry) Remove(connID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Remove from instance index
	for instanceUID, cID := range r.instanceIndex {
		if cID == connID {
			delete(r.instanceIndex, instanceUID)

			break
		}
	}

	delete(r.connections, connID)
}

// GetByInstanceUID implements port.WebSocketRegistry.
func (r *Registry) GetByInstanceUID(instanceUID string) (port.WebSocketConnection, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	connID, ok := r.instanceIndex[instanceUID]
	if !ok {
		return nil, false
	}

	conn, ok := r.connections[connID]

	return conn, ok
}

// UpdateInstanceUID implements port.WebSocketRegistry.
func (r *Registry) UpdateInstanceUID(connID string, instanceUID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if connection exists
	if _, ok := r.connections[connID]; !ok {
		return
	}

	// Update the instance index
	r.instanceIndex[instanceUID] = connID
}

// OpAMPWebSocketConnection is an adapter for OpAMP's Connection type.
type OpAMPWebSocketConnection struct {
	conn opamptypes.Connection
}

// NewOpAMPWebSocketConnection creates a new OpAMPWebSocketConnection.
func NewOpAMPWebSocketConnection(conn opamptypes.Connection) *OpAMPWebSocketConnection {
	return &OpAMPWebSocketConnection{
		conn: conn,
	}
}

// Send implements port.WebSocketConnection.
func (w *OpAMPWebSocketConnection) Send(ctx context.Context, message *protobufs.ServerToAgent) error {
	return w.conn.Send(ctx, message) //nolint:wrapcheck // adapter pattern
}

// Close implements port.WebSocketConnection.
func (w *OpAMPWebSocketConnection) Close() error {
	return w.conn.Disconnect() //nolint:wrapcheck // adapter pattern
}
