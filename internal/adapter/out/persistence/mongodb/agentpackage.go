//nolint:dupl // MongoDB adapter pattern - similar structure is intentional
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

var _ agentport.AgentPackagePersistencePort = (*AgentPackageMongoAdapter)(nil)

const (
	agentPackageCollectionName     = "agentpackages"
	agentPackageNamespaceFieldName = "metadata.namespace"
	agentPackageNameFieldName      = "metadata.name"
	agentPackageDeletedAtFieldName = "metadata.deletedAt"
)

// AgentPackageMongoAdapter is a struct that implements the AgentPackagePersistencePort interface.
type AgentPackageMongoAdapter struct {
	collection *mongo.Collection
	common     commonEntityAdapter[entity.AgentPackage, string]
	logger     *slog.Logger
}

// NewAgentPackageRepository creates a new instance of AgentPackageMongoAdapter.
func NewAgentPackageRepository(
	mongoDatabase *mongo.Database,
	logger *slog.Logger,
) *AgentPackageMongoAdapter {
	collection := mongoDatabase.Collection(agentPackageCollectionName)
	keyFunc := func(en *entity.AgentPackage) string {
		return en.Metadata.Name
	}
	keyQueryFunc := func(key string) any {
		return key
	}

	return &AgentPackageMongoAdapter{
		collection: collection,
		logger:     logger,
		common: newCommonAdapter(
			logger,
			collection,
			entity.AgentPackageNameFieldName,
			keyFunc,
			keyQueryFunc,
		),
	}
}

// GetAgentPackage implements agentport.AgentPackagePersistencePort.
func (a *AgentPackageMongoAdapter) GetAgentPackage(
	ctx context.Context, namespace string, name string,
) (*agentmodel.AgentPackage, error) {
	filter := a.filterByNamespaceAndNameExcludingDeleted(namespace, name)

	result := a.collection.FindOne(ctx, filter)

	err := result.Err()
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, port.ErrResourceNotExist
		}

		return nil, fmt.Errorf("get agent package: %w", err)
	}

	var agentPackageEntity entity.AgentPackage

	err = result.Decode(&agentPackageEntity)
	if err != nil {
		return nil, fmt.Errorf("decode agent package: %w", err)
	}

	return agentPackageEntity.ToDomain(), nil
}

// ListAgentPackages implements agentport.AgentPackagePersistencePort.
func (a *AgentPackageMongoAdapter) ListAgentPackages(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentPackage], error) {
	resp, err := a.common.list(ctx, options)
	if err != nil {
		return nil, err
	}

	items := make([]*agentmodel.AgentPackage, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, item.ToDomain())
	}

	return &model.ListResponse[*agentmodel.AgentPackage]{
		Items:              items,
		Continue:           resp.Continue,
		RemainingItemCount: resp.RemainingItemCount,
	}, nil
}

// PutAgentPackage implements agentport.AgentPackagePersistencePort.
func (a *AgentPackageMongoAdapter) PutAgentPackage(
	ctx context.Context, agentPackage *agentmodel.AgentPackage,
) (*agentmodel.AgentPackage, error) {
	agentPackageEntity := entity.AgentPackageFromDomain(agentPackage)
	namespace := agentPackage.Metadata.Namespace
	name := agentPackage.Metadata.Name

	_, err := a.collection.ReplaceOne(ctx,
		a.filterByNamespaceAndName(namespace, name),
		agentPackageEntity,
		options.Replace().SetUpsert(true),
	)
	if err != nil {
		return nil, fmt.Errorf("put agent package: %w", err)
	}

	// Return the domain model directly instead of querying again
	// This avoids issues with soft-deleted documents not being found
	return agentPackage, nil
}

func (a *AgentPackageMongoAdapter) filterByNamespaceAndName(
	namespace, name string,
) bson.M {
	return bson.M{
		agentPackageNamespaceFieldName: sanitizeResourceName(namespace),
		agentPackageNameFieldName:      sanitizeResourceName(name),
	}
}

func (a *AgentPackageMongoAdapter) filterByNamespaceAndNameExcludingDeleted(
	namespace, name string,
) bson.M {
	filter := a.filterByNamespaceAndName(namespace, name)
	filter[agentPackageDeletedAtFieldName] = nil

	return filter
}
