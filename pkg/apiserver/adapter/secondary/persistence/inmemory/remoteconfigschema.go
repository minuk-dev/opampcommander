//nolint:dupl // Resource CRUD adapters intentionally mirror the other aggregates.
package inmemory

import (
	"context"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var _ agentport.RemoteConfigSchemaPersistencePort = (*RemoteConfigSchemaRepository)(nil)

// RemoteConfigSchemaRepository is the in-memory implementation of
// [agentport.RemoteConfigSchemaPersistencePort].
type RemoteConfigSchemaRepository struct {
	store *store[namespacedName, *agentmodel.RemoteConfigSchema]
}

// NewRemoteConfigSchemaRepository creates a new in-memory RemoteConfigSchemaRepository.
func NewRemoteConfigSchemaRepository() *RemoteConfigSchemaRepository {
	return &RemoteConfigSchemaRepository{
		store: newStore[namespacedName](cloneRemoteConfigSchema, func(s *agentmodel.RemoteConfigSchema) *time.Time {
			return s.Metadata.DeletedAt
		}, (*agentmodel.RemoteConfigSchema).SelectorValues),
	}
}

// GetRemoteConfigSchema implements agentport.RemoteConfigSchemaPersistencePort.
func (r *RemoteConfigSchemaRepository) GetRemoteConfigSchema(
	_ context.Context, namespace string, name string, options *model.GetOptions,
) (*agentmodel.RemoteConfigSchema, error) {
	return r.store.get(namespacedName{Namespace: namespace, Name: name}, options)
}

// PutRemoteConfigSchema implements agentport.RemoteConfigSchemaPersistencePort.
//
// Like the MongoDB adapter, this is an optimistic-concurrency write: an update
// (ResourceVersion > 0) succeeds only if the stored version still matches, else it
// returns [model.ErrConflict]. On success the version is incremented and written
// back onto the passed schema.
func (r *RemoteConfigSchemaRepository) PutRemoteConfigSchema(
	_ context.Context, schema *agentmodel.RemoteConfigSchema,
) (*agentmodel.RemoteConfigSchema, error) {
	key := namespacedName{
		Namespace: schema.Metadata.Namespace,
		Name:      schema.Metadata.Name,
	}
	expected := schema.Metadata.ResourceVersion
	next := expected + 1

	toStore := cloneRemoteConfigSchema(schema)
	toStore.Metadata.ResourceVersion = next

	err := r.store.casPutOrCreate(key, toStore, expected, func(s *agentmodel.RemoteConfigSchema) int64 {
		return s.Metadata.ResourceVersion
	})
	if err != nil {
		return nil, err
	}

	schema.Metadata.ResourceVersion = next

	return schema, nil
}

// ListRemoteConfigSchemas implements agentport.RemoteConfigSchemaPersistencePort.
func (r *RemoteConfigSchemaRepository) ListRemoteConfigSchemas(
	_ context.Context, namespace string, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.RemoteConfigSchema], error) {
	return r.store.list(options, func(schema *agentmodel.RemoteConfigSchema) bool {
		return schema.Metadata.Namespace == namespace
	})
}
