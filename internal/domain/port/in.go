package port

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

var (
	ErrConnectionAlreadyExists = errors.New("connection already exists")
	ErrConnectionNotFound      = errors.New("connection not found")
)

type ConnectionUsecase interface {
	GetConnectionUsecase
	SetConnectionUsecase
	DeleteConnectionUsecase
	ListConnectionIDsUsecase
}

type GetConnectionUsecase interface {
	GetConnection(instanceUID uuid.UUID) (*model.Connection, error)
}

type SetConnectionUsecase interface {
	SetConnection(connection *model.Connection) error
}

type DeleteConnectionUsecase interface {
	DeleteConnection(instanceUID uuid.UUID) error
	FetchAndDeleteConnection(instanceUID uuid.UUID) (*model.Connection, error)
}

type ListConnectionIDsUsecase interface {
	ListConnectionIDs() []uuid.UUID
}

type AgentUsecase interface {
	GetAgentUsecase
	SaveAgentUsecase
	ListAgentUsecase
}

type GetAgentUsecase interface {
	GetAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error)
	GetOrCreateAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error)
}

type SaveAgentUsecase interface {
	SaveAgent(ctx context.Context, agent *model.Agent) error
}

type ListAgentUsecase interface {
	ListAgents(ctx context.Context) ([]*model.Agent, error)
}
