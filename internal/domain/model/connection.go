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

func NewConnection(id uuid.UUID) *Connection {
	return &Connection{
		ID:                id,
		serverToAgentChan: make(chan *protobufs.ServerToAgent),
	}
}

func (conn *Connection) Watch() *protobufs.ServerToAgent {
	return <-conn.serverToAgentChan
}

func (conn *Connection) SendServerToAgent(ctx context.Context, serverToAgent *protobufs.ServerToAgent) error {
	select {
	case conn.serverToAgentChan <- serverToAgent:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("cannot send a message to the channel: %w", ctx.Err())
	}
}

func (conn *Connection) Close() {
	close(conn.serverToAgentChan)
}
