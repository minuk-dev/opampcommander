package agentport

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"

	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	"github.com/minuk-dev/opampcommander/internal/domain/agent/model/serverevent"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

var (
	// ErrConnectionAlreadyExists is an error that indicates that the connection already exists.
	ErrConnectionAlreadyExists = errors.New("connection already exists")
	// ErrConnectionNotFound is an error that indicates that the connection was not found.
	ErrConnectionNotFound = errors.New("connection not found")
)

// AgentUsecase is an interface that defines the methods for agent use cases.
type AgentUsecase interface {
	// GetAgent retrieves an agent by its instance UID.
	GetAgent(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.Agent, error)
	// GetOrCreateAgent retrieves an agent by its instance UID or creates a new one if it does not exist.
	GetOrCreateAgent(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.Agent, error)
	// ListAgentsBySelector lists agents by the given selector.
	ListAgentsBySelector(
		ctx context.Context,
		selector agentmodel.AgentSelector,
		options *model.ListOptions,
	) (*model.ListResponse[*agentmodel.Agent], error)
	// SaveAgent saves the agent.
	SaveAgent(ctx context.Context, agent *agentmodel.Agent) error
	// ListAgents lists all agents.
	ListAgents(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*agentmodel.Agent], error)
	// SearchAgents searches agents by instance UID prefix.
	SearchAgents(ctx context.Context, query string,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.Agent], error)
}

// AgentNotificationUsecase is an interface for notifying servers about agent changes.
type AgentNotificationUsecase interface {
	// NotifyAgentUpdated notifies the connected server that the agent has pending messages.
	NotifyAgentUpdated(ctx context.Context, agent *agentmodel.Agent) error
	// RestartAgent requests the agent to restart.
	RestartAgent(ctx context.Context, instanceUID uuid.UUID) error
}

// AgentPackageUsecase is an interface that defines the methods for agent package use cases.
type AgentPackageUsecase interface {
	// GetAgentPackage retrieves an agent package by its name.
	GetAgentPackage(ctx context.Context, name string) (*agentmodel.AgentPackage, error)
	// ListAgentPackages lists all agent packages.
	ListAgentPackages(ctx context.Context,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.AgentPackage], error)
	// SaveAgentPackage saves the agent package.
	SaveAgentPackage(ctx context.Context,
		agentPackage *agentmodel.AgentPackage) (*agentmodel.AgentPackage, error)
	// DeleteAgentPackage deletes the agent package by its name.
	DeleteAgentPackage(ctx context.Context, name string, deletedAt time.Time, deletedBy string) error
}

// AgentRemoteConfigUsecase is an interface that defines the methods for agent remote config use cases.
type AgentRemoteConfigUsecase interface {
	// GetAgentRemoteConfig retrieves an agent remote config by its name.
	GetAgentRemoteConfig(ctx context.Context, name string) (*agentmodel.AgentRemoteConfig, error)
	// ListAgentRemoteConfigs lists all agent remote configs.
	ListAgentRemoteConfigs(
		ctx context.Context, options *model.ListOptions,
	) (*model.ListResponse[*agentmodel.AgentRemoteConfig], error)
	// SaveAgentRemoteConfig saves the agent remote config.
	SaveAgentRemoteConfig(
		ctx context.Context, agentRemoteConfig *agentmodel.AgentRemoteConfig,
	) (*agentmodel.AgentRemoteConfig, error)
	// DeleteAgentRemoteConfig deletes the agent remote config by its name.
	DeleteAgentRemoteConfig(ctx context.Context, name string, deletedAt time.Time, deletedBy string) error
}

// AgentGroupUsecase is an interface that defines the methods for agent group use cases.
type AgentGroupUsecase interface {
	// GetAgentGroup retrieves an agent group by its name.
	GetAgentGroup(ctx context.Context, name string, options *model.GetOptions) (*agentmodel.AgentGroup, error)
	// ListAgentGroups lists all agent groups.
	ListAgentGroups(
		ctx context.Context, options *model.ListOptions,
	) (*model.ListResponse[*agentmodel.AgentGroup], error)
	// SaveAgentGroup saves the agent group.
	SaveAgentGroup(ctx context.Context, name string,
		agentGroup *agentmodel.AgentGroup) (*agentmodel.AgentGroup, error)
	// DeleteAgentGroup deletes the agent group by its ID.
	DeleteAgentGroup(ctx context.Context, name string, deletedAt time.Time, deletedBy string) error
	// GetAgentGroupsForAgent retrieves all agent groups that match the agent's attributes.
	GetAgentGroupsForAgent(ctx context.Context, agent *agentmodel.Agent) ([]*agentmodel.AgentGroup, error)
}

// AgentGroupRelatedUsecase is an interface that defines methods related to agent groups.
type AgentGroupRelatedUsecase interface {
	// ListAgentsByAgentGroup lists agents belonging to a specific agent group.
	ListAgentsByAgentGroup(
		ctx context.Context,
		agentGroup *agentmodel.AgentGroup,
		options *model.ListOptions,
	) (*model.ListResponse[*agentmodel.Agent], error)
}

// CertificateUsecase defines the interface for certificate use cases.
type CertificateUsecase interface {
	GetCertificate(ctx context.Context, name string) (*agentmodel.Certificate, error)
	SaveCertificate(ctx context.Context,
		certificate *agentmodel.Certificate) (*agentmodel.Certificate, error)
	ListCertificate(ctx context.Context,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.Certificate], error)
	DeleteCertificate(ctx context.Context, name string, deletedAt time.Time,
		deletedBy string) (*agentmodel.Certificate, error)
}

// ServerUsecase is an interface that defines the methods for server use cases.
type ServerUsecase interface {
	// ServerUsecase should also implement ServerMessageUsecase.
	ServerMessageUsecase
	// GetServer retrieves a server by its ID.
	GetServer(ctx context.Context, id string) (*agentmodel.Server, error)
	// ListServers lists all servers.
	ListServers(ctx context.Context) ([]*agentmodel.Server, error)
}

// ServerIdentityProvider is an interface that defines the methods for providing server identity.
type ServerIdentityProvider interface {
	// CurrentServer returns the current server.
	CurrentServer(ctx context.Context) (*agentmodel.Server, error)
}

// ServerMessageUsecase is an interface that defines the methods for server message use cases.
type ServerMessageUsecase interface {
	// SendMessageToServerByServerID sends a message to the specified server.
	SendMessageToServerByServerID(ctx context.Context, serverID string, message serverevent.Message) error
	// SendMessageToServer sends a message to the specified server.
	SendMessageToServer(ctx context.Context, server *agentmodel.Server, message serverevent.Message) error
}

// ServerReceiverUsecase is an interface that defines the methods for server receiver use cases.
type ServerReceiverUsecase interface {
	// ReceiveMessageFromServer processes a message received from a server.
	ReceiveMessageFromServer() error
}

// ConnectionUsecase is an interface that defines the methods for connection use cases.
type ConnectionUsecase interface {
	// GetConnectionByInstanceUID returns the connection for the given instance UID.
	GetConnectionByInstanceUID(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.Connection, error)
	// GetOrCreateConnectionByID returns the connection for the given ID or creates a new one.
	GetOrCreateConnectionByID(ctx context.Context, id any) (*agentmodel.Connection, error)
	// GetConnectionByID returns the connection for the given ID.
	GetConnectionByID(ctx context.Context, id any) (*agentmodel.Connection, error)
	// ListConnections returns the list of connections.
	ListConnections(ctx context.Context,
		options *model.ListOptions) (*model.ListResponse[*agentmodel.Connection], error)
	// SaveConnection saves the connection.
	SaveConnection(ctx context.Context, connection *agentmodel.Connection) error
	// DeleteConnection deletes the connection.
	DeleteConnection(ctx context.Context, connection *agentmodel.Connection) error
	// SendServerToAgent sends a ServerToAgent message to the agent via WebSocket connection.
	SendServerToAgent(ctx context.Context, instanceUID uuid.UUID, message *protobufs.ServerToAgent) error
}
