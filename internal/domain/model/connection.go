package model

import (
	"crypto/sha256"
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
func ConvertConnIDToString(id any) string {
	// Use pointer address to avoid race condition with internal fields
	raw := fmt.Sprintf("%p", id)
	hash := sha256.New()
	result := hash.Sum([]byte(raw))

	return string(result)
}

// IsAnonymous returns true if the connection is anonymous.
func (conn *Connection) IsAnonymous() bool {
	return conn.InstanceUID == uuid.Nil
}

// IsManaged returns true if the connection is managed.
func (conn *Connection) IsManaged() bool {
	return !conn.IsAnonymous()
}

// SetInstanceUID sets the instance UID of the connection.
func (conn *Connection) SetInstanceUID(instanceUID uuid.UUID) {
	conn.InstanceUID = instanceUID
}
