package port

import (
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
	HandleAgentToServer(agentToServer *protobufs.AgentToServer)
}

// FetchServerToAgentUsecase is a use case that fetches a message from the connection.
type FetchServerToAgentUsecase interface {
	FetchServerToAgent(instanceUID uuid.UUID) (*protobufs.ServerToAgent, error)
}
