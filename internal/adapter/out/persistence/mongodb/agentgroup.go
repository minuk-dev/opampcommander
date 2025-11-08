package mongodb

import (
	"context"
	"log/slog"

	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb/entity"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agentgroup"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ port.AgentGroupPersistencePort = (*AgentGroupMongoAdapter)(nil)

const (
	agentGroupCollectionName = "agentgroups"
)

// AgentGroupMongoAdapter is a struct that implements the AgentGroupPersistencePort interface.
type AgentGroupMongoAdapter struct {
	collection *mongo.Collection
	common     commonEntityAdapter[entity.AgentGroup, string]
}

// NewAgentGroupRepository creates a new instance of AgentGroupMongoAdapter.
func NewAgentGroupRepository(
	mongoDatabase *mongo.Database,
	logger *slog.Logger,
) *AgentGroupMongoAdapter {
	collection := mongoDatabase.Collection(agentGroupCollectionName)
	keyFunc := func(en *entity.AgentGroup) string {
		return en.Name
	}
	keyQueryFunc := func(key string) any {
		return key
	}

	return &AgentGroupMongoAdapter{
		collection: collection,
		common: newCommonAdapter(
			logger,
			collection,
			entity.AgentGroupKeyFieldName,
			keyFunc,
			keyQueryFunc,
		),
	}
}

// GetAgentGroup implements port.AgentGroupPersistencePort.
func (a *AgentGroupMongoAdapter) GetAgentGroup(
	ctx context.Context, name string,
) (*agentgroup.AgentGroup, error) {
	entity, err := a.common.get(ctx, name)
	if err != nil {
		return nil, err
	}

	return entity.ToDomain(), nil
}

// ListAgentGroups implements port.AgentGroupPersistencePort.
func (a *AgentGroupMongoAdapter) ListAgentGroups(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentgroup.AgentGroup], error) {
	resp, err := a.common.list(ctx, options)
	if err != nil {
		return nil, err
	}

	return &model.ListResponse[*agentgroup.AgentGroup]{
		Items: lo.Map(resp.Items, func(item *entity.AgentGroup, _ int) *agentgroup.AgentGroup {
			return item.ToDomain()
		}),
		Continue:           resp.Continue,
		RemainingItemCount: resp.RemainingItemCount,
	}, nil
}

// PutAgentGroup implements port.AgentGroupPersistencePort.
//
//nolint:godox,revive
func (a *AgentGroupMongoAdapter) PutAgentGroup(
	ctx context.Context, name string, agentGroup *agentgroup.AgentGroup,
) error {
	// TODO: name should be used to save the agent group with the given name.
	// ref. https://github.com/minuk-dev/opampcommander/issues/145
	// Because some update operations may change the name of the agent group.
	entity := entity.AgentGroupFromDomain(agentGroup)

	err := a.common.put(ctx, entity)
	if err != nil {
		return err
	}

	return nil
}
