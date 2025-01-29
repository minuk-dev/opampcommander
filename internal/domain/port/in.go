package port

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"

	"github.com/minuk-dev/minuk-apiserver/internal/domain/model"
)

var (
	ErrConnectionAlreadyExists = errors.New("connection already exists")
	ErrConnectionNotFound      = errors.New("connection not found")
)

type OpAMPUsecase interface {
	// HandleAgentToServer handles the AgentToServer message.
	HandleAgentToServer(ctx context.Context, agentToServer *protobufs.AgentToServer) error

	// FetchServerToAgent fetches the ServerToAgent message by the given UUID.
	FetchServerToAgent(ctx context.Context, instanceUID uuid.UUID) (*protobufs.ServerToAgent, error)
}

type ConnectionUsecase interface {
	GetConnection(instanceUID uuid.UUID) (*model.Connection, error)
	SetConnection(connection *model.Connection) error
	DeleteConnection(instanceUID uuid.UUID) error
	ListConnectionIDs() []uuid.UUID
}
