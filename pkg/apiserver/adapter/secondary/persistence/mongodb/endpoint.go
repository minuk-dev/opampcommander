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

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb/entity"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/port"
)

var _ agentport.EndpointPersistencePort = (*EndpointMongoAdapter)(nil)

const (
	endpointCollectionName     = "endpoints"
	endpointNamespaceFieldName = "metadata.namespace"
	endpointNameFieldName      = "metadata.name"
	endpointDeletedAtFieldName = "metadata.deletedAt"
)

// EndpointMongoAdapter is a struct that implements the EndpointPersistencePort interface.
type EndpointMongoAdapter struct {
	collection *mongo.Collection
	common     commonEntityAdapter[entity.EndpointResourceEntity, string]
	logger     *slog.Logger
}

// NewEndpointRepository creates a new instance of EndpointMongoAdapter.
func NewEndpointRepository(
	mongoDatabase *mongo.Database,
	logger *slog.Logger,
) *EndpointMongoAdapter {
	collection := mongoDatabase.Collection(endpointCollectionName)
	keyFunc := func(en *entity.EndpointResourceEntity) string {
		return en.Metadata.Name
	}
	keyQueryFunc := func(key string) any {
		return key
	}

	return &EndpointMongoAdapter{
		collection: collection,
		logger:     logger,
		common: newCommonAdapter(
			logger,
			collection,
			entity.EndpointNameFieldName,
			keyFunc,
			keyQueryFunc,
		),
	}
}

// GetEndpoint implements agentport.EndpointPersistencePort.
func (a *EndpointMongoAdapter) GetEndpoint(
	ctx context.Context, namespace string, name string, options *model.GetOptions,
) (*agentmodel.Endpoint, error) {
	var filter bson.M
	if options != nil && options.IncludeDeleted {
		filter = a.filterByNamespaceAndName(namespace, name)
	} else {
		filter = a.filterByNamespaceAndNameExcludingDeleted(namespace, name)
	}

	result := a.collection.FindOne(ctx, filter)

	err := result.Err()
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, port.ErrResourceNotExist
		}

		return nil, fmt.Errorf("get endpoint: %w", err)
	}

	var endpointEntity entity.EndpointResourceEntity

	err = result.Decode(&endpointEntity)
	if err != nil {
		return nil, fmt.Errorf("decode endpoint: %w", err)
	}

	return endpointEntity.ToDomain(), nil
}

// ListEndpoints implements agentport.EndpointPersistencePort.
func (a *EndpointMongoAdapter) ListEndpoints(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Endpoint], error) {
	resp, err := a.common.list(ctx, options)
	if err != nil {
		return nil, err
	}

	items := make([]*agentmodel.Endpoint, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, item.ToDomain())
	}

	return &model.ListResponse[*agentmodel.Endpoint]{
		Items:              items,
		Continue:           resp.Continue,
		RemainingItemCount: resp.RemainingItemCount,
	}, nil
}

// PutEndpoint implements agentport.EndpointPersistencePort.
func (a *EndpointMongoAdapter) PutEndpoint(
	ctx context.Context, endpoint *agentmodel.Endpoint,
) (*agentmodel.Endpoint, error) {
	endpointEntity := entity.EndpointResourceEntityFromDomain(endpoint)
	namespace := endpoint.Metadata.Namespace
	name := endpoint.Metadata.Name

	_, err := a.collection.ReplaceOne(ctx,
		a.filterByNamespaceAndName(namespace, name),
		endpointEntity,
		options.Replace().SetUpsert(true),
	)
	if err != nil {
		return nil, fmt.Errorf("put endpoint: %w", err)
	}

	// Return the domain model directly instead of querying again
	// This avoids issues with soft-deleted documents not being found
	return endpoint, nil
}

func (a *EndpointMongoAdapter) filterByNamespaceAndName(
	namespace, name string,
) bson.M {
	return bson.M{
		endpointNamespaceFieldName: sanitizeResourceName(namespace),
		endpointNameFieldName:      sanitizeResourceName(name),
	}
}

func (a *EndpointMongoAdapter) filterByNamespaceAndNameExcludingDeleted(
	namespace, name string,
) bson.M {
	filter := a.filterByNamespaceAndName(namespace, name)
	filter[endpointDeletedAtFieldName] = nil

	return filter
}
