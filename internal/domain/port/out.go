package port

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

type AgentPersistencePort interface {
	GetAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error)
	PutAgent(ctx context.Context, agent *model.Agent) error
	ListAgents(ctx context.Context) ([]*model.Agent, error)
}

type CommandPersistencePort interface {
	GetCommand(ctx context.Context, commandID uuid.UUID) (*model.Command, error)
	GetCommandByInstanceUID(ctx context.Context, instanceUID uuid.UUID) (*model.Command, error)
	SaveCommand(ctx context.Context, command *model.Command) error
}

var (
	ErrAgentNotExist      = errors.New("agent does not exist")
	ErrMultipleAgentExist = errors.New("multiple agent exists")
)
