package port

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
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
	// SearchAgents searches agents by query with pagination options.
	SearchAgents(ctx context.Context, query string, options *model.ListOptions) (*model.ListResponse[*model.Agent], error)
}

// ServerEventSenderPort is an interface that defines the methods for sending events to servers.
type ServerEventSenderPort interface {
	// SendMessageToServer sends a message to the specified server.
	SendMessageToServer(ctx context.Context, serverID string, message serverevent.Message) error
}

// ReceiveServerEventHandler is a function type for handling received server events.
type ReceiveServerEventHandler func(ctx context.Context, message *serverevent.Message) error

// ServerEventReceiverPort is an interface that defines the methods for receiving events from servers.
type ServerEventReceiverPort interface {
	// StartReceiver starts receiving messages from servers using the provided handler.
	// It's a blocking call.
	StartReceiver(ctx context.Context, handler ReceiveServerEventHandler) error
}

// AgentGroupPersistencePort is an interface that defines the methods for agent group persistence.
type AgentGroupPersistencePort interface {
	// GetAgentGroup retrieves an agent group by its ID.
	GetAgentGroup(ctx context.Context, name string) (*model.AgentGroup, error)
	// PutAgentGroup saves the agent group.
	PutAgentGroup(ctx context.Context, name string, agentGroup *model.AgentGroup) (*model.AgentGroup, error)
	// ListAgentGroups retrieves a list of agent groups with pagination options.
	ListAgentGroups(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.AgentGroup], error)
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

// AgentPackagePersistencePort is an interface that defines the methods for agent package persistence.
type AgentPackagePersistencePort interface {
	// GetAgentPackage retrieves an agent package by its name.
	GetAgentPackage(ctx context.Context, name string) (*model.AgentPackage, error)
	// PutAgentPackage saves or updates an agent package.
	PutAgentPackage(ctx context.Context, agentPackage *model.AgentPackage) (*model.AgentPackage, error)
	// ListAgentPackages retrieves a list of agent packages with pagination options.
	ListAgentPackages(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.AgentPackage], error)
}

// AgentRemoteConfigPersistencePort is an interface that defines the methods for agent remote config persistence.
type AgentRemoteConfigPersistencePort interface {
	// GetAgentRemoteConfig retrieves an agent remote config by its name.
	GetAgentRemoteConfig(ctx context.Context, name string) (*model.AgentRemoteConfigResource, error)
	// PutAgentRemoteConfig saves or updates an agent remote config.
	PutAgentRemoteConfig(ctx context.Context, config *model.AgentRemoteConfigResource) (*model.AgentRemoteConfigResource, error)
	// ListAgentRemoteConfigs retrieves a list of agent remote configs with pagination options.
	ListAgentRemoteConfigs(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.AgentRemoteConfigResource], error)
}

var (
	// ErrResourceNotExist is an error that indicates that the resource does not exist.
	ErrResourceNotExist = errors.New("resource does not exist")
	// ErrMultipleResourceExist is an error that indicates that multiple resources exist.
	ErrMultipleResourceExist = errors.New("multiple resources exist")
)
