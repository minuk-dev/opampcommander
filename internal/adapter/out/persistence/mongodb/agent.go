package mongodb

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb/entity"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ domainport.AgentPersistencePort = (*AgentRepository)(nil)

const (
	agentCollectionName = "agents"
)

// AgentRepository is a struct that implements the AgentPersistencePort interface.
type AgentRepository struct {
	collection *mongo.Collection
	common     commonEntityAdapter[entity.Agent, uuid.UUID]
}

// NewAgentRepository creates a new instance of AgentRepository.
func NewAgentRepository(
	mongoDatabase *mongo.Database,
	logger *slog.Logger,
) *AgentRepository {
	collection := mongoDatabase.Collection(agentCollectionName)
	keyFunc := func(domain *entity.Agent) uuid.UUID {
		return domain.InstanceUID
	}

	return &AgentRepository{
		collection: collection,
		common: newCommonAdapter(
			logger,
			collection,
			entity.AgentKeyFieldName,
			keyFunc,
		),
	}
}

// GetAgent implements port.AgentPersistencePort.
func (a *AgentRepository) GetAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error) {
	entity, err := a.common.get(ctx, instanceUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent from persistence: %w", err)
	}

	return entity.ToDomain(), nil
}

// ListAgents implements port.AgentPersistencePort.
func (a *AgentRepository) ListAgents(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	resp, err := a.common.list(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents from persistence: %w", err)
	}

	return &model.ListResponse[*model.Agent]{
		Items: lo.Map(resp.Items, func(item *entity.Agent, _ int) *model.Agent {
			return item.ToDomain()
		}),
		Continue:           resp.Continue,
		RemainingItemCount: resp.RemainingItemCount,
	}, nil
}

// PutAgent implements port.AgentPersistencePort.
func (a *AgentRepository) PutAgent(ctx context.Context, agent *model.Agent) error {
	entity := entity.AgentFromDomain(agent)

	err := a.common.put(ctx, entity)
	if err != nil {
		return fmt.Errorf("failed to put agent to persistence: %w", err)
	}

	return nil
}
