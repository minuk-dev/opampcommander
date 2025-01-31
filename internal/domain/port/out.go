package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

type AgentPersistencePort interface {
	GetAgentPort
	StoreAgentPort
}

type GetAgentPort interface {
	GetAgent(ctx context.Context, instanceUID uuid.UUID) (model.Agent, error)
}

type StoreAgentPort interface {
	StoreAgent(ctx context.Context, agent model.Agent) error
}
