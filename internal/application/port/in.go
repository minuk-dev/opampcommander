package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
)

// OpAMPUsecase covers OpAMP protocol
// This usecase should be called by the adapter.
type OpAMPUsecase interface {
	HandleAgentToServerUsecase
	FetchServerToAgentUsecase
}

// HandleAgentToServerUsecase is a use case that handles a message from the connection.
type HandleAgentToServerUsecase interface {
	HandleAgentToServer(ctx context.Context, agentToServer *protobufs.AgentToServer) error
}

// FetchServerToAgentUsecase is a use case that fetches a message from the connection.
type FetchServerToAgentUsecase interface {
	FetchServerToAgent(ctx context.Context, instanceUID uuid.UUID) (*protobufs.ServerToAgent, error)
}

// AdminUsecase is a use case that handles admin operations.
type AdminUsecase interface {
	ApplyRawConfig(ctx context.Context, targetInstanceUID uuid.UUID, config any) error
}
