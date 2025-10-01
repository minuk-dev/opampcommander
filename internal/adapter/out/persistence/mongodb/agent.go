package mongodb

import (
	"context"

	"github.com/google/uuid"
	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb/entity"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	"go.mongodb.org/mongo-driver/mongo"
)

var _ domainport.AgentPersistencePort = (*AgentRepository)(nil)

const (
	agentCollectionName = "agents"
)

// AgentRepository is a struct that implements the AgentPersistencePort interface.
type AgentRepository struct {
	collection *mongo.Collection
	common     commonAdapter[model.Agent]
}

// NewAgentRepository creates a new instance of AgentRepository.
func NewAgentRepository(
	mongoDatabase *mongo.Database,
) *AgentRepository {
	collection := mongoDatabase.Collection(agentCollectionName)
	return &AgentRepository{
		collection: collection,
		common: newCommonAdapter(
			collection,
			func(domain *model.Agent) (Entity[model.Agent], error) {
				return entity.AgentFromDomain(domain), nil
			},
			func() Entity[model.Agent] {
				//exhaustruct:ignore
				return &entity.Agent{}
			},
			func(domain *model.Agent) string {
				return domain.InstanceUID.String()
			},
		),
	}
}

// GetAgent implements port.AgentPersistencePort.
func (a *AgentRepository) GetAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error) {
	return a.common.get(ctx, instanceUID.String())
}

// ListAgents implements port.AgentPersistencePort.
func (a *AgentRepository) ListAgents(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.Agent], error) {
	return a.common.list(ctx, options)
}

// PutAgent implements port.AgentPersistencePort.
func (a *AgentRepository) PutAgent(ctx context.Context, agent *model.Agent) error {
	return a.common.put(ctx, agent)
}
