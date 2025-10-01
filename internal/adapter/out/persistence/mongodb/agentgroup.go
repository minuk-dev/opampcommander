package mongodb

import (
	"context"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb/entity"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agentgroup"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"go.mongodb.org/mongo-driver/mongo"
)

var _ port.AgentGroupPersistencePort = (*AgentGroupMongoAdapter)(nil)

const (
	agentGroupCollectionName = "agentgroups"
)

// AgentGroupMongoAdapter is a struct that implements the AgentGroupPersistencePort interface.
type AgentGroupMongoAdapter struct {
	collection *mongo.Collection
	common     commonAdapter[agentgroup.AgentGroup]
}

// NewAgentGroupEtcdAdapter creates a new instance of AgentGroupEtcdAdapter.
func NewAgentGroupEtcdAdapter(
	mongoDatabase *mongo.Database,
) *AgentGroupMongoAdapter {
	collection := mongoDatabase.Collection(agentGroupCollectionName)
	ToEntityFunc := func(domain *agentgroup.AgentGroup) (Entity[agentgroup.AgentGroup], error) {
		return entity.AgentGroupFromDomain(domain), nil
	}

	CreateNewEmptyEntityFunc := func() Entity[agentgroup.AgentGroup] {
		//exhaustruct:ignore
		return &entity.AgentGroup{}
	}

	keyFunc := func(domain *agentgroup.AgentGroup) string {
		return domain.Name
	}

	return &AgentGroupMongoAdapter{
		collection: collection,
		common: newCommonAdapter(
			collection,
			ToEntityFunc,
			CreateNewEmptyEntityFunc,
			keyFunc,
		),
	}
}

// GetAgentGroup implements port.AgentGroupPersistencePort.
func (a *AgentGroupMongoAdapter) GetAgentGroup(
	ctx context.Context, name string,
) (*agentgroup.AgentGroup, error) {
	return a.common.get(ctx, name)
}

// ListAgentGroups implements port.AgentGroupPersistencePort.
func (a *AgentGroupMongoAdapter) ListAgentGroups(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentgroup.AgentGroup], error) {
	return a.common.list(ctx, options)
}

// PutAgentGroup implements port.AgentGroupPersistencePort.
//
//nolint:godox,revive
func (a *AgentGroupMongoAdapter) PutAgentGroup(
	ctx context.Context, name string, agentGroup *agentgroup.AgentGroup,
) error {
	// TODO: name should be used to save the agent group with the given name.
	// TODO: https://github.com/minuk-dev/opampcommander/issues/145
	// Because some update operations may change the name of the agent group.
	return a.common.put(ctx, agentGroup)
}
