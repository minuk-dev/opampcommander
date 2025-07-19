package port

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

// AgentPersistencePort is an interface that defines the methods for agent persistence.
type AgentPersistencePort interface {
	// GetAgent retrieves an agent by its instance UID.
	GetAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error)
	// GetAgentByID retrieves an agent by its ID.
	PutAgent(ctx context.Context, agent *model.Agent) error
	// ListAgents retrieves a list of agents with pagination options.
	ListAgents(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.Agent], error)
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

var (
	// ErrAgentNotExist is an error that indicates that the agent does not exist.
	ErrAgentNotExist = errors.New("agent does not exist")
	// ErrMultipleAgentExist is an error that indicates that multiple agents exist.
	ErrMultipleAgentExist = errors.New("multiple agent exists")
)
