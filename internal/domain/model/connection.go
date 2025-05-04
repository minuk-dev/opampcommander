package model

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// OpAMPPollingInterval is a polling interval for OpAMP.
	// ref. https://github.com/minuk-dev/opampcommander/issues/8
	OpAMPPollingInterval = 30 * time.Second
)

// Connection represents a connection to an agent.
type Connection struct {
	ID uuid.UUID

	// Data is a map of string to string.
	// It is a kind of raw data to manage connection.
	// It's useful especially when you get/update Unknown Connection.
	// Please see [NewAnonymousConnection]
	Data map[string]string

	state connectionState
}

// connectionState is a state of the connection.
type connectionState struct {
	mu sync.RWMutex

	// lastCommunicatedAt is the last communicated time.
	lastCommunicatedAt time.Time
}

// NewConnection returns a new Connection instance.
func NewConnection(id uuid.UUID) *Connection {
	return &Connection{
		ID:    id,
		state: newConnectionState(),
	}
}

// NewAnonymousConnection returns a new Connection instance with an unknown ID.
func NewAnonymousConnection(data map[string]string) *Connection {
	return &Connection{
		ID:   uuid.Nil,
		Data: data,
	}
}

// IsAnonymous returns true if the connection is anonymous.
func (conn *Connection) IsAnonymous() bool {
	return conn.ID == uuid.Nil
}

// IsIdentified returns true if the connection is not anonymous.
func (conn *Connection) IsIdentified() bool {
	return !conn.IsAnonymous()
}

// RefreshLastCommunicatedAt refreshes the last communicated time.
func (conn *Connection) RefreshLastCommunicatedAt(at time.Time) {
	conn.state.SetLastCommunicatedAt(at)
}

// LastCommunicatedAt returns the last communicated time.
func (conn *Connection) LastCommunicatedAt() time.Time {
	return conn.state.LastCommunicatedAt()
}

// IsAlive returns true if the connection is alive.
func (conn *Connection) IsAlive(now time.Time) bool {
	return now.Sub(conn.LastCommunicatedAt()) < 2*OpAMPPollingInterval
}

// Close closes the connection.
// Even if already closed, do nothing.
func (conn *Connection) Close() error {
	return nil
}

func newConnectionState() connectionState {
	return connectionState{
		mu:                 sync.RWMutex{},
		lastCommunicatedAt: time.Time{},
	}
}

func (s *connectionState) SetLastCommunicatedAt(at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastCommunicatedAt = at
}

func (s *connectionState) LastCommunicatedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.lastCommunicatedAt
}
