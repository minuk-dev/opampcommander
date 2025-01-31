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
	agentToServerChan chan *protobufs.AgentToServer
}

// NewConnection returns a new Connection instance.
func NewConnection(id uuid.UUID) *Connection {
	return &Connection{
		ID:                id,
		serverToAgentChan: make(chan *protobufs.ServerToAgent),
		agentToServerChan: make(chan *protobufs.AgentToServer),
	}
}

// Watch returns a channel that can be used to receive messages from the connection.
func (conn *Connection) Watch() <-chan *protobufs.ServerToAgent {
	return conn.serverToAgentChan
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

// HandleAgentToServer handles a message from the connection.
// It is called by the connection manager when a message arrives from the connection.
func (conn *Connection) HandleAgentToServer(_ context.Context, _ *protobufs.AgentToServer) error {
	return nil
}

func (conn *Connection) Close() {
	close(conn.serverToAgentChan)
}
