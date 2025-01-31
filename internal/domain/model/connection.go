package model

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
)

type Connection struct {
	ID uuid.UUID

	serverToAgentChan chan *protobufs.ServerToAgent
}

// NewConnection returns a new Connection instance.
func NewConnection(id uuid.UUID) *Connection {
	return &Connection{
		ID:                id,
		serverToAgentChan: make(chan *protobufs.ServerToAgent),
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

// Close closes the connection.
// Even if already closed, do nothing.
func (conn *Connection) Close() error {
	if conn.serverToAgentChan != nil {
		close(conn.serverToAgentChan)
	}

	return nil
}
