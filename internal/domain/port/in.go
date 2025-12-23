package port

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/serverevent"
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
	GetAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error)
	// GetOrCreateAgent retrieves an agent by its instance UID or creates a new one if it does not exist.
	GetOrCreateAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error)
	// ListAgentsBySelector lists agents by the given selector.
	ListAgentsBySelector(
		ctx context.Context,
		selector model.AgentSelector,
		options *model.ListOptions,
	) (*model.ListResponse[*model.Agent], error)
	// SaveAgent saves the agent.
	SaveAgent(ctx context.Context, agent *model.Agent) error
	// ListAgents lists all agents.
	ListAgents(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.Agent], error)
}

// AgentNotificationUsecase is an interface for notifying servers about agent changes.
type AgentNotificationUsecase interface {
	// NotifyAgentUpdated notifies the connected server that the agent has pending messages.
	NotifyAgentUpdated(ctx context.Context, agent *model.Agent) error
	// RestartAgent requests the agent to restart.
	RestartAgent(ctx context.Context, instanceUID uuid.UUID) error
}

// AgentGroupUsecase is an interface that defines the methods for agent group use cases.
type AgentGroupUsecase interface {
	// GetAgentGroup retrieves an agent group by its name
	GetAgentGroup(ctx context.Context, name string) (*model.AgentGroup, error)
	// SaveAgentGroup saves the agent group.
	ListAgentGroups(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.AgentGroup], error)
	// SaveAgentGroup saves the agent group.
	SaveAgentGroup(ctx context.Context, name string, agentGroup *model.AgentGroup) (*model.AgentGroup, error)
	// DeleteAgentGroup deletes the agent group by its ID.
	DeleteAgentGroup(ctx context.Context, name string, deletedAt time.Time, deletedBy string) error
	// GetAgentGroupsForAgent retrieves all agent groups that match the agent's attributes.
	GetAgentGroupsForAgent(ctx context.Context, agent *model.Agent) ([]*model.AgentGroup, error)
}

// AgentGroupRelatedUsecase is an interface that defines methods related to agent groups.
type AgentGroupRelatedUsecase interface {
	// ListAgentsByAgentGroup lists agents belonging to a specific agent group.
	ListAgentsByAgentGroup(
		ctx context.Context,
		agentGroup *model.AgentGroup,
		options *model.ListOptions,
	) (*model.ListResponse[*model.Agent], error)
}

// ServerUsecase is an interface that defines the methods for server use cases.
type ServerUsecase interface {
	// ServerUsecase should also implement ServerMessageUsecase
	ServerMessageUsecase
	// ServerIdentityProvider
	ServerIdentityProvider

	// GetServer retrieves a server by its ID.
	GetServer(ctx context.Context, id string) (*model.Server, error)
	// ListServers lists all servers.
	// The number of servers is expected to be small, so no pagination is needed.
	ListServers(ctx context.Context) ([]*model.Server, error)
}

// ServerIdentityProvider is an interface that defines the methods for providing server identity.
type ServerIdentityProvider interface {
	// CurrentServer returns the current server.
	CurrentServer(ctx context.Context) (*model.Server, error)
}

// ServerMessageUsecase is an interface that defines the methods for server message use cases.
// Some usecases may require sending messages to other servers.
// So, this interface defines as a separate interface.
type ServerMessageUsecase interface {
	// SendMessageToServerByServerID sends a message to the specified server.
	SendMessageToServerByServerID(ctx context.Context, serverID string, message serverevent.Message) error
	// SendMessageToServer sends a message to the specified server.
	SendMessageToServer(ctx context.Context, server *model.Server, message serverevent.Message) error
}

// ServerReceiverUsecase is an interface that defines the methods for server receiver use cases.
type ServerReceiverUsecase interface {
	// ReceiveMessageFromServer processes a message received from a server.
	ReceiveMessageFromServer() error
}

// ConnectionUsecase is an interface that defines the methods for connection use cases.
type ConnectionUsecase interface {
	// GetConnectionByInstanceUID returns the connection for the given instance UID.
	GetConnectionByInstanceUID(ctx context.Context, instanceUID uuid.UUID) (*model.Connection, error)
	// GetOrCreateConnectionByID returns the connection for the given ID or creates a new one if it does not exist.
	GetOrCreateConnectionByID(ctx context.Context, id any) (*model.Connection, error)
	// GetConnectionByID returns the connection for the given ID.
	GetConnectionByID(ctx context.Context, id any) (*model.Connection, error)
	// ListConnections returns the list of connections.
	ListConnections(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.Connection], error)
	// SaveConnection saves the connection.
	SaveConnection(ctx context.Context, connection *model.Connection) error
	// DeleteConnection deletes the connection.
	DeleteConnection(ctx context.Context, connection *model.Connection) error
	// SendServerToAgent sends a ServerToAgent message to the agent via WebSocket connection.
	SendServerToAgent(ctx context.Context, instanceUID uuid.UUID, message *protobufs.ServerToAgent) error
}
