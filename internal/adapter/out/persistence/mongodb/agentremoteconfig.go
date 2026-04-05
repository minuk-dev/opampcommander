//nolint:dupl // MongoDB adapter pattern - similar structure is intentional
package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb/entity"
	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ agentport.AgentRemoteConfigPersistencePort = (*AgentRemoteConfigMongoAdapter)(nil)

const (
	agentRemoteConfigCollectionName     = "agentremoteconfigs"
	agentRemoteConfigNamespaceFieldName = "metadata.namespace"
	agentRemoteConfigNameFieldName      = "metadata.name"
	agentRemoteConfigDeletedAtFieldName = "metadata.deletedAt"
)

// AgentRemoteConfigMongoAdapter is a struct that implements the AgentRemoteConfigPersistencePort interface.
type AgentRemoteConfigMongoAdapter struct {
	collection *mongo.Collection
	common     commonEntityAdapter[entity.AgentRemoteConfigResourceEntity, string]
	logger     *slog.Logger
}

// NewAgentRemoteConfigRepository creates a new instance of AgentRemoteConfigMongoAdapter.
func NewAgentRemoteConfigRepository(
	mongoDatabase *mongo.Database,
	logger *slog.Logger,
) *AgentRemoteConfigMongoAdapter {
	collection := mongoDatabase.Collection(agentRemoteConfigCollectionName)
	keyFunc := func(en *entity.AgentRemoteConfigResourceEntity) string {
		return en.Metadata.Name
	}
	keyQueryFunc := func(key string) any {
		return key
	}

	return &AgentRemoteConfigMongoAdapter{
		collection: collection,
		logger:     logger,
		common: newCommonAdapter(
			logger,
			collection,
			entity.AgentRemoteConfigNameFieldName,
			keyFunc,
			keyQueryFunc,
		),
	}
}

// GetAgentRemoteConfig implements agentport.AgentRemoteConfigPersistencePort.
func (a *AgentRemoteConfigMongoAdapter) GetAgentRemoteConfig(
	ctx context.Context, namespace string, name string,
) (*agentmodel.AgentRemoteConfig, error) {
	filter := a.filterByNamespaceAndNameExcludingDeleted(namespace, name)

	result := a.collection.FindOne(ctx, filter)

	err := result.Err()
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, port.ErrResourceNotExist
		}

		return nil, fmt.Errorf("get agent remote config: %w", err)
	}

	var agentRemoteConfigEntity entity.AgentRemoteConfigResourceEntity

	err = result.Decode(&agentRemoteConfigEntity)
	if err != nil {
		return nil, fmt.Errorf("decode agent remote config: %w", err)
	}

	return agentRemoteConfigEntity.ToDomain(), nil
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
	agentRemoteConfigEntity := entity.AgentRemoteConfigResourceEntityFromDomain(config)
	namespace := config.Metadata.Namespace
	name := config.Metadata.Name

	_, err := a.collection.ReplaceOne(ctx,
		a.filterByNamespaceAndName(namespace, name),
		agentRemoteConfigEntity,
		options.Replace().SetUpsert(true),
	)
	if err != nil {
		return nil, fmt.Errorf("put agent remote config: %w", err)
	}

	// Return the domain model directly instead of querying again
	// This avoids issues with soft-deleted documents not being found
	return config, nil
}

func (a *AgentRemoteConfigMongoAdapter) filterByNamespaceAndName(
	namespace, name string,
) bson.M {
	return bson.M{
		agentRemoteConfigNamespaceFieldName: sanitizeResourceName(namespace),
		agentRemoteConfigNameFieldName:      sanitizeResourceName(name),
	}
}

func (a *AgentRemoteConfigMongoAdapter) filterByNamespaceAndNameExcludingDeleted(
	namespace, name string,
) bson.M {
	filter := a.filterByNamespaceAndName(namespace, name)
	filter[agentRemoteConfigDeletedAtFieldName] = nil

	return filter
}
