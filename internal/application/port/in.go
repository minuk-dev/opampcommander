// Package port is a package that defines the ports for the application layer.
package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	opamptypes "github.com/open-telemetry/opamp-go/server/types"

	v1agent "github.com/minuk-dev/opampcommander/api/v1/agent"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

// OpAMPUsecase is a use case that handles OpAMP protocol operations.
// Please see [github.com/open-telemetry/opamp-go/server/types/ConnectionCallbacks].
type OpAMPUsecase interface {
	OnConnected(ctx context.Context, conn opamptypes.Connection)
	OnMessage(ctx context.Context, conn opamptypes.Connection, message *protobufs.AgentToServer) *protobufs.ServerToAgent
	OnConnectionClose(conn opamptypes.Connection)
	OnReadMessageError(conn opamptypes.Connection, mt int, msgByte []byte, err error)
}

// AdminUsecase is a use case that handles admin operations.
type AdminUsecase interface {
	ApplyRawConfig(ctx context.Context, targetInstanceUID uuid.UUID, config any) error
	ListConnections(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.Connection], error)
}

// AgentManageUsecase is a use case that handles agent management operations.
type AgentManageUsecase interface {
	GetAgent(ctx context.Context, instanceUID uuid.UUID) (*v1agent.Agent, error)
	ListAgents(ctx context.Context, options *model.ListOptions) (*v1agent.ListResponse, error)
	SendCommand(ctx context.Context, targetInstanceUID uuid.UUID, command *model.Command) error
}

// CommandLookUpUsecase is a use case that handles command operations.
type CommandLookUpUsecase interface {
	// GetCommand retrieves a command by its ID.
	GetCommand(ctx context.Context, commandID uuid.UUID) (*model.Command, error)
	// GetCommandByInstanceUID retrieves a command by its instance UID.
	GetCommandByInstanceUID(ctx context.Context, instanceUID uuid.UUID) ([]*model.Command, error)
	// ListCommands lists all commands.
	ListCommands(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.Command], error)
}
