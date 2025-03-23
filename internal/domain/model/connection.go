package model

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
)

const (
	// OpAMPPollingInterval is a polling interval for OpAMP.
	// ref. https://github.com/minuk-dev/opampcommander/issues/8
	OpAMPPollingInterval = 30 * time.Second
)

type Connection struct {
	ID uuid.UUID

	serverToAgentChan chan *protobufs.ServerToAgent

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
		ID:                id,
		serverToAgentChan: make(chan *protobufs.ServerToAgent, 1), // buffer size 1
		state:             newConnectionState(),
	}
}

// SendAgentToServer sends a message to the connection.
func (conn *Connection) SendServerToAgent(ctx context.Context, serverToAgent *protobufs.ServerToAgent) error {
	select {
	case conn.serverToAgentChan <- serverToAgent:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("cannot send a message to the channel: %w", ctx.Err())
	}
}

// FetchServerToAgent fetches a message from the connection.
func (conn *Connection) FetchServerToAgent(ctx context.Context) (*protobufs.ServerToAgent, error) {
	select {
	case serverToAgent := <-conn.serverToAgentChan:
		return serverToAgent, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("cannot fetch a message from the channel: %w", ctx.Err())
	}
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
	if conn.serverToAgentChan != nil {
		close(conn.serverToAgentChan)
	}

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
