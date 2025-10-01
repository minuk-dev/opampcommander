package mongodb

import (
	"context"

	"github.com/google/uuid"
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
}

// GetAgent implements port.AgentPersistencePort.
func (a *AgentRepository) GetAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error) {
}

// ListAgents implements port.AgentPersistencePort.
func (a *AgentRepository) ListAgents(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.Agent], error) {
	panic("unimplemented")
}

// PutAgent implements port.AgentPersistencePort.
func (a *AgentRepository) PutAgent(ctx context.Context, agent *model.Agent) error {
	panic("unimplemented")
}

// NewAgentRepository creates a new instance of AgentRepository.
func NewAgentRepository(
	mongoDatabase *mongo.Database,
) *AgentRepository {
	return &AgentRepository{
		collection: mongoDatabase.Collection(agentCollectionName),
	}
}
