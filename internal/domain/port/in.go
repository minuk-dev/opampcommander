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

// AgentUsecase is an interface that defines the methods for agent use cases.
type AgentUsecase interface {
	// GetAgent retrieves an agent by its instance UID.
	GetAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error)
	// GetOrCreateAgent retrieves an agent by its instance UID or creates a new one if it does not exist.
	GetOrCreateAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error)
	// SaveAgent saves the agent.
	SaveAgent(ctx context.Context, agent *model.Agent) error
	// ListAgents lists all agents.
	ListAgents(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.Agent], error)
	UpdateAgentConfigUsecase
}

// UpdateAgentConfigUsecase is an interface that defines the methods for updating agent configurations.
type UpdateAgentConfigUsecase interface {
	// UpdateAgentConfig updates the agent configuration.
	UpdateAgentConfig(ctx context.Context, instanceUID uuid.UUID, config any) error
}

// ConnectionUsecase is an interface that defines the methods for connection use cases.
type ConnectionUsecase interface {
	// GetConnectionByInstanceUID returns the connection for the given instance UID.
	GetConnectionByInstanceUID(ctx context.Context, instanceUID uuid.UUID) (*model.Connection, error)
	// GetConnectionByID returns the connection for the given ID.
	GetConnectionByID(ctx context.Context, id any) (*model.Connection, error)
	// ListConnections returns the list of connections.
	ListConnections(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.Connection], error)
	// SaveConnection saves the connection.
	SaveConnection(ctx context.Context, connection *model.Connection) error
	// DeleteConnection deletes the connection.
	DeleteConnection(ctx context.Context, connection *model.Connection) error
}

// CommandUsecase is an interface that defines the methods for command use cases.
type CommandUsecase interface {
	// GetCommand retrieves a command by its ID.
	GetCommand(ctx context.Context, commandID uuid.UUID) (*model.Command, error)
	// GetCommandByInstanceUID retrieves a command by its instance UID.
	GetCommandByInstanceUID(ctx context.Context, instanceUID uuid.UUID) ([]*model.Command, error)
	// SaveCommand saves the command.
	SaveCommand(ctx context.Context, command *model.Command) error
	// ListCommands lists all commands.
	ListCommands(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.Command], error)
}
