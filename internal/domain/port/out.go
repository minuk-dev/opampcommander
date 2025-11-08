package port

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agentgroup"
	"github.com/minuk-dev/opampcommander/internal/domain/model/serverevent"
)

// AgentPersistencePort is an interface that defines the methods for agent persistence.
type AgentPersistencePort interface {
	// GetAgent retrieves an agent by its instance UID.
	GetAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error)
	// GetAgentByID retrieves an agent by its ID.
	PutAgent(ctx context.Context, agent *model.Agent) error
	// ListAgents retrieves a list of agents with pagination options.
	ListAgents(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.Agent], error)
	// ListAgentsBySelector retrieves a list of agents matching the given selector with pagination options.
	ListAgentsBySelector(
		ctx context.Context,
		selector model.AgentSelector,
		options *model.ListOptions,
	) (*model.ListResponse[*model.Agent], error)
}

// ServerEventSenderPort is an interface that defines the methods for sending events to servers.
type ServerEventSenderPort interface {
	// SendMessageToServer sends a message to the specified server.
	SendMessageToServer(ctx context.Context, serverID string, message serverevent.Message) error
}

// ServerEventReceiverPort is an interface that defines the methods for receiving events from servers.
type ServerEventReceiverPort interface {
	// ReceiveMessageFromServer receives a message from a server.
	ReceiveMessageFromServer(ctx context.Context) (*serverevent.Message, error)
}

// AgentGroupPersistencePort is an interface that defines the methods for agent group persistence.
type AgentGroupPersistencePort interface {
	// GetAgentGroup retrieves an agent group by its ID.
	GetAgentGroup(ctx context.Context, name string) (*agentgroup.AgentGroup, error)
	// PutAgentGroup saves the agent group.
	PutAgentGroup(ctx context.Context, name string, agentGroup *agentgroup.AgentGroup) error
	// ListAgentGroups retrieves a list of agent groups with pagination options.
	ListAgentGroups(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*agentgroup.AgentGroup], error)
}

// CommandPersistencePort is an interface that defines the methods for command persistence.
type CommandPersistencePort interface {
	// GetCommand retrieves a command by its ID.
	GetCommand(ctx context.Context, commandID uuid.UUID) (*model.Command, error)
	// GetCommandByInstanceUID retrieves a command by its instance UID.
	GetCommandByInstanceUID(ctx context.Context, instanceUID uuid.UUID) (*model.Command, error)
	// SaveCommand saves the command.
	SaveCommand(ctx context.Context, command *model.Command) error
}

// ServerPersistencePort is an interface that defines the methods for server persistence.
type ServerPersistencePort interface {
	// GetServer retrieves a server by its ID.
	GetServer(ctx context.Context, id string) (*model.Server, error)
	// PutServer saves or updates a server.
	PutServer(ctx context.Context, server *model.Server) error
	// ListServers retrieves a list of all servers.
	ListServers(ctx context.Context) ([]*model.Server, error)
}

var (
	// ErrResourceNotExist is an error that indicates that the resource does not exist.
	ErrResourceNotExist = errors.New("resource does not exist")
	// ErrMultipleResourceExist is an error that indicates that multiple resources exist.
	ErrMultipleResourceExist = errors.New("multiple resources exist")
)
