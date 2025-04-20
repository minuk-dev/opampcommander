package port

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

var (
	// ErrConnectionAlreadyExists is an error that indicates that the connection already exists.
	ErrConnectionAlreadyExists = errors.New("connection already exists")
	// ErrConnectionNotFound is an error that indicates that the connection was not found.
	ErrConnectionNotFound = errors.New("connection not found")
)

// ConnectionUsecase is an interface that defines the methods for connection use cases.
type ConnectionUsecase interface {
	GetConnectionUsecase
	SetConnectionUsecase
	DeleteConnectionUsecase
	ListConnectionIDsUsecase
}

// GetConnectionUsecase is an interface that defines the methods for getting connections.
type GetConnectionUsecase interface {
	// GetConnection retrieves a connection by its instance UID.
	GetConnection(instanceUID uuid.UUID) (*model.Connection, error)
	// GetOrCreateConnection retrieves a connection by its instance UID.
	GetOrCreateConnection(instanceUID uuid.UUID) (*model.Connection, error)
}

// SetConnectionUsecase is an interface that defines the methods for setting connections.
type SetConnectionUsecase interface {
	// SaveConnection saves the connection to the persistence layer.
	SaveConnection(connection *model.Connection) error
}

// DeleteConnectionUsecase is an interface that defines the methods for deleting connections.
type DeleteConnectionUsecase interface {
	// DeleteConnection deletes a connection by its instance UID.
	DeleteConnection(instanceUID uuid.UUID) error
	// FetchAndDeleteConnection fetches a connection by its instance UID and deletes it.
	FetchAndDeleteConnection(instanceUID uuid.UUID) (*model.Connection, error)
}

// ListConnectionIDsUsecase is an interface that defines the methods for listing connection IDs.
type ListConnectionIDsUsecase interface {
	// ListConnectionIDs lists all connection IDs.
	ListConnections() []*model.Connection
}

// AgentUsecase is an interface that defines the methods for agent use cases.
type AgentUsecase interface {
	GetAgentUsecase
	SaveAgentUsecase
	ListAgentUsecase
}

// GetAgentUsecase is an interface that defines the methods for getting agents.
type GetAgentUsecase interface {
	// GetAgent retrieves an agent by its instance UID.
	GetAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error)
	// GetOrCreateAgent retrieves an agent by its instance UID or creates a new one if it does not exist.
	GetOrCreateAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error)
}

// SaveAgentUsecase is an interface that defines the methods for saving agents.
type SaveAgentUsecase interface {
	// SaveAgent saves the agent.
	SaveAgent(ctx context.Context, agent *model.Agent) error
}

// ListAgentUsecase is an interface that defines the methods for listing agents.
type ListAgentUsecase interface {
	// ListAgents lists all agents.
	ListAgents(ctx context.Context) ([]*model.Agent, error)
}

// CommandUsecase is an interface that defines the methods for command use cases.
type CommandUsecase interface {
	GetCommandUsecase
	SaveCommandUsecase
}

// GetCommandUsecase is an interface that defines the methods for getting commands.
type GetCommandUsecase interface {
	// GetCommand retrieves a command by its ID.
	GetCommand(ctx context.Context, commandID uuid.UUID) (*model.Command, error)
	// GetCommandByInstanceUID retrieves a command by its instance UID.
	GetCommandByInstanceUID(ctx context.Context, instanceUID uuid.UUID) ([]*model.Command, error)
}

// SaveCommandUsecase is an interface that defines the methods for saving commands.
type SaveCommandUsecase interface {
	// SaveCommand saves the command.
	SaveCommand(ctx context.Context, command *model.Command) error
}
