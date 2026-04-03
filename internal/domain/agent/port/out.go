package agentport

import (
	"context"

	"github.com/google/uuid"

	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	"github.com/minuk-dev/opampcommander/internal/domain/agent/model/serverevent"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

// ReceiveServerEventHandler is a function type for handling received server events.
type ReceiveServerEventHandler func(ctx context.Context, message *serverevent.Message) error

// AgentPersistencePort is an interface that defines the methods for agent persistence.
type AgentPersistencePort interface {
	// GetAgent retrieves an agent by its instance UID.
	GetAgent(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.Agent, error)
	// PutAgent saves or updates an agent.
	PutAgent(ctx context.Context, agent *agentmodel.Agent) error
	// ListAgents retrieves a list of agents with pagination options.
	ListAgents(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*agentmodel.Agent], error)
	// ListAgentsBySelector retrieves a list of agents matching the given selector.
	ListAgentsBySelector(
		ctx context.Context,
		selector agentmodel.AgentSelector,
		options *model.ListOptions,
	) (*model.ListResponse[*agentmodel.Agent], error)
	// SearchAgents searches agents by query with pagination options.
	SearchAgents(ctx context.Context, query string,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.Agent], error)
}

// ServerEventSenderPort is an interface that defines the methods for sending events to servers.
type ServerEventSenderPort interface {
	// SendMessageToServer sends a message to the specified server.
	SendMessageToServer(ctx context.Context, serverID string, message serverevent.Message) error
}

// ServerEventReceiverPort is an interface that defines the methods for receiving events from servers.
type ServerEventReceiverPort interface {
	// StartReceiver starts receiving messages from servers using the provided handler.
	StartReceiver(ctx context.Context, handler ReceiveServerEventHandler) error
}

// AgentGroupPersistencePort is an interface that defines the methods for agent group persistence.
type AgentGroupPersistencePort interface {
	// GetAgentGroup retrieves an agent group by its ID.
	GetAgentGroup(ctx context.Context, name string,
		options *model.GetOptions) (*agentmodel.AgentGroup, error)
	// PutAgentGroup saves the agent group.
	PutAgentGroup(ctx context.Context, name string,
		agentGroup *agentmodel.AgentGroup) (*agentmodel.AgentGroup, error)
	// ListAgentGroups retrieves a list of agent groups with pagination options.
	ListAgentGroups(ctx context.Context,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.AgentGroup], error)
}

// ServerPersistencePort is an interface that defines the methods for server persistence.
type ServerPersistencePort interface {
	// GetServer retrieves a server by its ID.
	GetServer(ctx context.Context, id string) (*agentmodel.Server, error)
	// PutServer saves or updates a server.
	PutServer(ctx context.Context, server *agentmodel.Server) error
	// ListServers retrieves a list of all servers.
	ListServers(ctx context.Context) ([]*agentmodel.Server, error)
}

// AgentPackagePersistencePort is an interface that defines the methods for agent package persistence.
type AgentPackagePersistencePort interface {
	// GetAgentPackage retrieves an agent package by its name.
	GetAgentPackage(ctx context.Context, name string) (*agentmodel.AgentPackage, error)
	// PutAgentPackage saves or updates an agent package.
	PutAgentPackage(ctx context.Context,
		agentPackage *agentmodel.AgentPackage) (*agentmodel.AgentPackage, error)
	// ListAgentPackages retrieves a list of agent packages with pagination options.
	ListAgentPackages(ctx context.Context,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.AgentPackage], error)
}

// AgentRemoteConfigPersistencePort is an interface that defines the methods for agent remote config persistence.
type AgentRemoteConfigPersistencePort interface {
	// GetAgentRemoteConfig retrieves an agent remote config by its name.
	GetAgentRemoteConfig(ctx context.Context, name string) (*agentmodel.AgentRemoteConfig, error)
	// PutAgentRemoteConfig saves or updates an agent remote config.
	PutAgentRemoteConfig(
		ctx context.Context,
		config *agentmodel.AgentRemoteConfig,
	) (*agentmodel.AgentRemoteConfig, error)
	// ListAgentRemoteConfigs retrieves a list of agent remote configs with pagination options.
	ListAgentRemoteConfigs(
		ctx context.Context,
		options *model.ListOptions,
	) (*model.ListResponse[*agentmodel.AgentRemoteConfig], error)
}

// CertificatePersistencePort is an interface that defines the methods for certificate config persistence.
type CertificatePersistencePort interface {
	GetCertificate(ctx context.Context, name string) (*agentmodel.Certificate, error)
	PutCertificate(ctx context.Context,
		certificate *agentmodel.Certificate) (*agentmodel.Certificate, error)
	ListCertificate(ctx context.Context,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.Certificate], error)
}
