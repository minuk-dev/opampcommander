package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"

	"github.com/minuk-dev/minuk-apiserver/internal/domain/model"
)

type OpAMPUsecase interface {
	// HandleAgentToServer handles the AgentToServer message.
	HandleAgentToServer(ctx context.Context, agentToServer *protobufs.AgentToServer) error

	// FetchServerToAgent fetches the ServerToAgent message by the given UUID.
	FetchServerToAgent(ctx context.Context, instanceUID uuid.UUID) (*protobufs.ServerToAgent, error)
}

type ConnectionUsecase interface {
	GetConnection(ctx context.Context, instanceUID uuid.UUID) (model.Connection, error)
	StoreConnection(ctx context.Context, connection model.Connection) error
}
