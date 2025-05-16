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

// Type represents the type of the connection.
type Type int

const (
	// TypeUnknown is the unknown type.
	TypeUnknown Type = iota
	// TypeHTTP is the HTTP type.
	TypeHTTP
	// TypeWebSocket is the WebSocket type.
	TypeWebSocket
)

// Connection represents a connection to an agent.
type Connection struct {
	// Key is the unique identifier for the connection.
	// It should be unique across all connections to use as a key in a map.
	// Normally, it is [types.Connection] by OpAMP.
	ID any

	// Type is the type of the connection.
	Type Type

	// UID is the unique identifier for the connection.
	// It is used to identify the connection in the database.
	UID uuid.UUID

	// InstanceUID is id of the agent.
	InstanceUID uuid.UUID

	// LastCommunicatedAt is the last time the connection was communicated with.
	LastCommunicatedAt time.Time
}

// NewConnection creates a new Connection instance with the given ID and type.
func NewConnection(id any, typ Type) *Connection {
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
	return conn.Type == TypeWebSocket || now.Sub(conn.LastCommunicatedAt) < 2*OpAMPPollingInterval
}

// IDString returns a string value
// In some cases, a unique string id instead of any type.
func (conn *Connection) IDString() string {
	return ConvertConnIDToString(conn.ID)
}

// ConvertConnIDToString converts the connection ID to a string.
func ConvertConnIDToString(id any) string {
	raw := fmt.Sprintf("%+v", id)
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
