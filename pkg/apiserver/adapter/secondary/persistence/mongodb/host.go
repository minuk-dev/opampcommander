//nolint:dupl // MongoDB adapter pattern - similar structure to container is intentional.
package mongodb

import (
	"context"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb/entity"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var _ agentport.HostPersistencePort = (*HostMongoAdapter)(nil)

const (
	hostCollectionName = "hosts"
)

// HostMongoAdapter implements the HostPersistencePort interface.
type HostMongoAdapter struct {
	collection *mongo.Collection
	common     commonEntityAdapter[entity.Host, string]
}

// NewHostRepository creates a new instance of HostMongoAdapter.
func NewHostRepository(
	mongoDatabase *mongo.Database,
	logger *slog.Logger,
) *HostMongoAdapter {
	collection := mongoDatabase.Collection(hostCollectionName)
	keyFunc := func(hostEntity *entity.Host) string {
		return hostEntity.Metadata.ID
	}
	keyQueryFunc := func(key string) any {
		return key
	}

	return &HostMongoAdapter{
		collection: collection,
		common: newCommonAdapter(
			logger,
			collection,
			entity.HostKeyFieldName,
			keyFunc,
			keyQueryFunc,
		),
	}
}

// GetHost implements agentport.HostPersistencePort.
func (a *HostMongoAdapter) GetHost(ctx context.Context, id string) (*agentmodel.Host, error) {
	hostEntity, err := a.common.get(ctx, id, nil)
	if err != nil {
		return nil, fmt.Errorf("get host: %w", err)
	}

	return hostEntity.ToDomain(), nil
}

// ListHosts implements agentport.HostPersistencePort.
func (a *HostMongoAdapter) ListHosts(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Host], error) {
	resp, err := a.common.list(ctx, options)
	if err != nil {
		return nil, err
	}

	items := make([]*agentmodel.Host, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, item.ToDomain())
	}

	return &model.ListResponse[*agentmodel.Host]{
		Items:              items,
		Continue:           resp.Continue,
		RemainingItemCount: resp.RemainingItemCount,
	}, nil
}

// PutHost implements agentport.HostPersistencePort.
//
// PutHost is an optimistic-concurrency write: an update only succeeds when the
// stored document's resourceVersion still equals the version the in-memory host
// was loaded with, otherwise it returns [model.ErrConflict] rather than clobbering
// a concurrent writer (another HA node that discovered the same host). On success
// the version is incremented and written back onto the passed host.
func (a *HostMongoAdapter) PutHost(
	ctx context.Context, host *agentmodel.Host,
) (*agentmodel.Host, error) {
	expected := host.Metadata.ResourceVersion
	next := expected + 1

	hostEntity := entity.HostFromDomain(host)
	hostEntity.Metadata.ResourceVersion = next

	filter := bson.M{entity.HostKeyFieldName: host.Metadata.ID}

	err := casReplace(ctx, a.collection, filter, hostEntity, expected)
	if err != nil {
		return nil, fmt.Errorf("put host: %w", err)
	}

	host.Metadata.ResourceVersion = next

	return host, nil
}
