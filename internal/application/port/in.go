// Package port is a package that defines the ports for the application layer.
package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	opamptypes "github.com/open-telemetry/opamp-go/server/types"

	v1agent "github.com/minuk-dev/opampcommander/api/v1/agent"
	v1agentgroup "github.com/minuk-dev/opampcommander/api/v1/agentgroup"
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

// AgentGroupManageUsecase is a use case that handles agent group management operations.
type AgentGroupManageUsecase interface {
	GetAgentGroup(ctx context.Context, name string) (*v1agentgroup.AgentGroup, error)
	ListAgentGroups(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*v1agentgroup.AgentGroup], error)
	CreateAgentGroup(ctx context.Context, createCommand *CreateAgentGroupCommand) (*v1agentgroup.AgentGroup, error)
	UpdateAgentGroup(
		ctx context.Context,
		name string,
		agentGroup *v1agentgroup.AgentGroup,
	) (*v1agentgroup.AgentGroup, error)
	DeleteAgentGroup(ctx context.Context, name string) error
}

// CreateAgentGroupCommand is a command to create an agent group.
type CreateAgentGroupCommand struct {
	Name       string
	Attributes v1agentgroup.Attributes
	Selector   v1agentgroup.AgentSelector
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
