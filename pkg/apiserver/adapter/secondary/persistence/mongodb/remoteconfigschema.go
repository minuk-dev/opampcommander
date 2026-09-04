//nolint:dupl // Resource CRUD adapters intentionally mirror the other aggregates.
package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb/entity"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var _ agentport.RemoteConfigSchemaPersistencePort = (*RemoteConfigSchemaMongoAdapter)(nil)

const (
	remoteConfigSchemaCollectionName     = "remoteconfigschemas"
	remoteConfigSchemaNamespaceFieldName = "metadata.namespace"
	remoteConfigSchemaNameFieldName      = "metadata.name"
	remoteConfigSchemaDeletedAtFieldName = "metadata.deletedAt"
)

// RemoteConfigSchemaMongoAdapter implements the RemoteConfigSchemaPersistencePort interface.
type RemoteConfigSchemaMongoAdapter struct {
	collection *mongo.Collection
	common     commonEntityAdapter[entity.RemoteConfigSchemaResourceEntity, string]
	logger     *slog.Logger
}

// NewRemoteConfigSchemaRepository creates a new instance of RemoteConfigSchemaMongoAdapter.
func NewRemoteConfigSchemaRepository(
	mongoDatabase *mongo.Database,
	logger *slog.Logger,
) *RemoteConfigSchemaMongoAdapter {
	collection := mongoDatabase.Collection(remoteConfigSchemaCollectionName)
	keyFunc := func(en *entity.RemoteConfigSchemaResourceEntity) string {
		return en.Metadata.Name
	}
	keyQueryFunc := func(key string) any {
		return key
	}

	return &RemoteConfigSchemaMongoAdapter{
		collection: collection,
		logger:     logger,
		common: newCommonAdapter(
			logger,
			collection,
			entity.RemoteConfigSchemaNameFieldName,
			keyFunc,
			keyQueryFunc,
			remoteConfigSchemaSelectorSchema,
		),
	}
}

// GetRemoteConfigSchema implements agentport.RemoteConfigSchemaPersistencePort.
func (a *RemoteConfigSchemaMongoAdapter) GetRemoteConfigSchema(
	ctx context.Context, namespace string, name string, options *model.GetOptions,
) (*agentmodel.RemoteConfigSchema, error) {
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
			return nil, model.ErrResourceNotExist
		}

		return nil, fmt.Errorf("get remote config schema: %w", err)
	}

	var schemaEntity entity.RemoteConfigSchemaResourceEntity

	err = result.Decode(&schemaEntity)
	if err != nil {
		return nil, fmt.Errorf("decode remote config schema: %w", err)
	}

	return schemaEntity.ToDomain(), nil
}

// ListRemoteConfigSchemas implements agentport.RemoteConfigSchemaPersistencePort.
func (a *RemoteConfigSchemaMongoAdapter) ListRemoteConfigSchemas(
	ctx context.Context, namespace string, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.RemoteConfigSchema], error) {
	resp, err := a.common.listWithConditions(ctx, options, bson.M{
		remoteConfigSchemaNamespaceFieldName: sanitizeResourceName(namespace),
	})
	if err != nil {
		return nil, err
	}

	items := make([]*agentmodel.RemoteConfigSchema, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, item.ToDomain())
	}

	return &model.ListResponse[*agentmodel.RemoteConfigSchema]{
		Items:              items,
		Continue:           resp.Continue,
		RemainingItemCount: resp.RemainingItemCount,
	}, nil
}

// PutRemoteConfigSchema implements agentport.RemoteConfigSchemaPersistencePort.
//
// PutRemoteConfigSchema is an optimistic-concurrency write: an update only succeeds
// when the stored document's resourceVersion still equals the version the in-memory
// schema was loaded with, otherwise it returns [model.ErrConflict] rather than
// silently clobbering a concurrent writer. On success the version is incremented and
// written back onto the passed schema.
func (a *RemoteConfigSchemaMongoAdapter) PutRemoteConfigSchema(
	ctx context.Context, schema *agentmodel.RemoteConfigSchema,
) (*agentmodel.RemoteConfigSchema, error) {
	namespace := schema.Metadata.Namespace
	name := schema.Metadata.Name
	expected := schema.Metadata.ResourceVersion
	next := expected + 1

	schemaEntity := entity.RemoteConfigSchemaResourceEntityFromDomain(schema)
	schemaEntity.Metadata.ResourceVersion = next

	err := casReplace(ctx, a.collection, a.filterByNamespaceAndName(namespace, name), schemaEntity, expected)
	if err != nil {
		return nil, fmt.Errorf("put remote config schema: %w", err)
	}

	schema.Metadata.ResourceVersion = next

	return schema, nil
}

func (a *RemoteConfigSchemaMongoAdapter) filterByNamespaceAndName(
	namespace, name string,
) bson.M {
	return bson.M{
		remoteConfigSchemaNamespaceFieldName: sanitizeResourceName(namespace),
		remoteConfigSchemaNameFieldName:      sanitizeResourceName(name),
	}
}

func (a *RemoteConfigSchemaMongoAdapter) filterByNamespaceAndNameExcludingDeleted(
	namespace, name string,
) bson.M {
	filter := a.filterByNamespaceAndName(namespace, name)
	filter[remoteConfigSchemaDeletedAtFieldName] = nil

	return filter
}
