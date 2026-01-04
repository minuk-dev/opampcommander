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
	OnConnectedWithType(ctx context.Context, conn opamptypes.Connection, isWebSocket bool)
	OnMessage(ctx context.Context, conn opamptypes.Connection, message *protobufs.AgentToServer) *protobufs.ServerToAgent
	OnConnectionClose(conn opamptypes.Connection)
	OnReadMessageError(conn opamptypes.Connection, mt int, msgByte []byte, err error)
	OnMessageResponseError(conn opamptypes.Connection, message *protobufs.ServerToAgent, err error)
}

// AdminUsecase is a use case that handles admin operations.
type AdminUsecase interface {
	ListConnections(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.Connection], error)
}

// AgentManageUsecase is a use case that handles agent management operations.
type AgentManageUsecase interface {
	GetAgent(ctx context.Context, instanceUID uuid.UUID) (*v1agent.Agent, error)
	ListAgents(ctx context.Context, options *model.ListOptions) (*v1agent.ListResponse, error)
	SearchAgents(ctx context.Context, query string, options *model.ListOptions) (*v1agent.ListResponse, error)
	SetNewInstanceUID(ctx context.Context, instanceUID uuid.UUID, newInstanceUID uuid.UUID) (*v1agent.Agent, error)
	RestartAgent(ctx context.Context, instanceUID uuid.UUID) error
}

// AgentGroupManageUsecase is a use case that handles agent group management operations.
type AgentGroupManageUsecase interface {
	GetAgentGroup(ctx context.Context, name string) (*v1agentgroup.AgentGroup, error)
	ListAgentGroups(ctx context.Context, options *model.ListOptions) (*v1agentgroup.ListResponse, error)
	ListAgentsByAgentGroup(
		ctx context.Context,
		agentGroupName string,
		options *model.ListOptions,
	) (*v1agent.ListResponse, error)
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
	Name        string
	Attributes  v1agentgroup.Attributes
	Priority    int
	Selector    v1agentgroup.AgentSelector
	AgentConfig *v1agentgroup.AgentConfig
}
