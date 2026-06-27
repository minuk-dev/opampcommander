package agentmodel

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	// OpAMPPollingInterval is a polling interval for OpAMP.
	// ref. https://github.com/minuk-dev/opampcommander/issues/8
	OpAMPPollingInterval = 30 * time.Second
)

// ConnectionType represents the type of the connection.
type ConnectionType int

const (
	// ConnectionTypeUnknown is the unknown type.
	ConnectionTypeUnknown ConnectionType = iota
	// ConnectionTypeHTTP is the HTTP type.
	ConnectionTypeHTTP
	// ConnectionTypeWebSocket is the WebSocket type.
	ConnectionTypeWebSocket
)

// String returns the string representation of the ConnectionType.
func (ct ConnectionType) String() string {
	switch ct {
	case ConnectionTypeHTTP:
		return "HTTP"
	case ConnectionTypeWebSocket:
		return "WebSocket"
	case ConnectionTypeUnknown:
		fallthrough
	default:
		return "Unknown"
	}
}

// ConnectionTypeFromString converts a string to a ConnectionType.
func ConnectionTypeFromString(s string) ConnectionType {
	switch s {
	case "HTTP":
		return ConnectionTypeHTTP
	case "WebSocket":
		return ConnectionTypeWebSocket
	default:
		return ConnectionTypeUnknown
	}
}

// Connection represents a connection to an agent.
// This is a pure domain model containing only metadata about the connection.
// The actual WebSocket connection object is managed separately by the WebSocketRegistry.
type Connection struct {
	// Key is the unique identifier for the connection.
	// It should be unique across all connections to use as a key in a map.
	// Normally, it is [types.Connection] by OpAMP.
	ID any

	// Type is the type of the connection.
	Type ConnectionType

	// UID is the unique identifier for the connection.
	// It is used to identify the connection in the database.
	UID uuid.UUID

	// InstanceUID is id of the agent.
	InstanceUID uuid.UUID

	// Namespace is the namespace the connection belongs to.
	// It is derived from the connected agent's namespace.
	Namespace string

	// LastCommunicatedAt is the last time the connection was communicated with.
	LastCommunicatedAt time.Time
}

// NewConnection creates a new Connection instance with the given ID and type.
func NewConnection(id any, typ ConnectionType) *Connection {
	return &Connection{
		ID:                 id,
		Type:               typ,
		UID:                uuid.New(),
		InstanceUID:        uuid.Nil,
		Namespace:          DefaultNamespaceName,
		LastCommunicatedAt: time.Time{},
	}
}

// IsAlive returns true if the connection is alive.
func (conn *Connection) IsAlive(now time.Time) bool {
	return conn.Type == ConnectionTypeWebSocket || now.Sub(conn.LastCommunicatedAt) < 2*OpAMPPollingInterval
}

// IDString returns a string value
// In some cases, a unique string id instead of any type.
func (conn *Connection) IDString() string {
	return ConvertConnIDToString(conn.ID)
}

// ConvertConnIDToString converts the connection ID to a string.
// Uses %p for pointer-like types and %v fallback for value types (e.g. opamp-go httpConnection struct).
func ConvertConnIDToString(id any) string {
	raw := fmt.Sprintf("%p", id)
	h := sha256.New()
	_, _ = h.Write([]byte(raw))

	return hex.EncodeToString(h.Sum(nil))
}

// IsAnonymous returns true if the connection is anonymous.
func (conn *Connection) IsAnonymous() bool {
	return conn.InstanceUID == uuid.Nil
}

// ServerConnection is a persisted, cluster-visible snapshot of a single live connection,
// stamped with the server instance that owns it. Each server periodically writes the
// snapshot of its local connections so other servers can query a cluster-wide view (the
// in-memory connection map itself is node-local). It is not the live connection — the
// transport object (Connection.ID) is intentionally absent.
type ServerConnection struct {
	// ServerID is the identifier of the server instance that owns this connection.
	ServerID string
	// UID is the unique identifier of the connection.
	UID uuid.UUID
	// InstanceUID is the id of the agent on the other end.
	InstanceUID uuid.UUID
	// Type is the type of the connection.
	Type ConnectionType
	// Namespace is the namespace the connection belongs to.
	Namespace string
	// LastCommunicatedAt is the last time the connection was communicated with.
	LastCommunicatedAt time.Time
	// SnapshotAt is when this record was last refreshed by its owning server. It bounds
	// staleness: records from a crashed server stop refreshing and are filtered out of
	// cluster reads once SnapshotAt falls outside the staleness window.
	SnapshotAt time.Time
}

// IsAlive reports whether the connection is alive, using the same rule as Connection.IsAlive:
// WebSocket connections are always considered alive; others are alive while their last
// communication is within the polling window.
func (sc *ServerConnection) IsAlive(now time.Time) bool {
	return sc.Type == ConnectionTypeWebSocket || now.Sub(sc.LastCommunicatedAt) < 2*OpAMPPollingInterval
}

// IsManaged returns true if the connection is managed.
func (conn *Connection) IsManaged() bool {
	return !conn.IsAnonymous()
}

// SetInstanceUID sets the instance UID of the connection.
func (conn *Connection) SetInstanceUID(instanceUID uuid.UUID) {
	conn.InstanceUID = instanceUID
}

// SetNamespace sets the namespace of the connection.
func (conn *Connection) SetNamespace(namespace string) {
	conn.Namespace = namespace
}
