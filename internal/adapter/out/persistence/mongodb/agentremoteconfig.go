package mongodb

import (
	"context"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb/entity"
	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

var _ agentport.AgentRemoteConfigPersistencePort = (*AgentRemoteConfigMongoAdapter)(nil)

const (
	agentRemoteConfigCollectionName = "agentremoteconfigs"
)

// AgentRemoteConfigMongoAdapter is a struct that implements the AgentRemoteConfigPersistencePort interface.
type AgentRemoteConfigMongoAdapter struct {
	common commonEntityAdapter[entity.AgentRemoteConfigResourceEntity, string]
}

// NewAgentRemoteConfigRepository creates a new instance of AgentRemoteConfigMongoAdapter.
func NewAgentRemoteConfigRepository(
	mongoDatabase *mongo.Database,
	logger *slog.Logger,
) *AgentRemoteConfigMongoAdapter {
	collection := mongoDatabase.Collection(agentRemoteConfigCollectionName)
	keyFunc := func(en *entity.AgentRemoteConfigResourceEntity) string {
		return en.Name
	}
	keyQueryFunc := func(key string) any {
		return key
	}

	return &AgentRemoteConfigMongoAdapter{
		common: newCommonAdapter(
			logger,
			collection,
			entity.AgentRemoteConfigKeyFieldName,
			keyFunc,
			keyQueryFunc,
		),
	}
}

// GetAgentRemoteConfig implements agentport.AgentRemoteConfigPersistencePort.
func (a *AgentRemoteConfigMongoAdapter) GetAgentRemoteConfig(
	ctx context.Context, name string,
) (*agentmodel.AgentRemoteConfig, error) {
	en, err := a.common.get(ctx, name, nil)
	if err != nil {
		return nil, fmt.Errorf("get agent remote config: %w", err)
	}

	return en.ToDomain(), nil
}

// ListAgentRemoteConfigs implements agentport.AgentRemoteConfigPersistencePort.
func (a *AgentRemoteConfigMongoAdapter) ListAgentRemoteConfigs(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentRemoteConfig], error) {
	resp, err := a.common.list(ctx, options)
	if err != nil {
		return nil, err
	}

	items := make([]*agentmodel.AgentRemoteConfig, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, item.ToDomain())
	}

	return &model.ListResponse[*agentmodel.AgentRemoteConfig]{
		Items:              items,
		Continue:           resp.Continue,
		RemainingItemCount: resp.RemainingItemCount,
	}, nil
}

// PutAgentRemoteConfig implements agentport.AgentRemoteConfigPersistencePort.
func (a *AgentRemoteConfigMongoAdapter) PutAgentRemoteConfig(
	ctx context.Context, config *agentmodel.AgentRemoteConfig,
) (*agentmodel.AgentRemoteConfig, error) {
	en := entity.AgentRemoteConfigResourceEntityFromDomain(config)

	err := a.common.put(ctx, en)
	if err != nil {
		return nil, fmt.Errorf("put agent remote config: %w", err)
	}

	return a.GetAgentRemoteConfig(ctx, config.Metadata.Name)
}
