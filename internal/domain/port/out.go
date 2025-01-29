package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/minuk-dev/minuk-apiserver/internal/domain/model"
)

type AgentPersistencePort interface {
	GetAgentUsecase
	StoreAgentUsecase
}

type GetAgentUsecase interface {
	GetAgent(ctx context.Context, instanceUID uuid.UUID) (model.Agent, error)
}

type StoreAgentUsecase interface {
	StoreAgent(ctx context.Context, agent model.Agent) error
}
