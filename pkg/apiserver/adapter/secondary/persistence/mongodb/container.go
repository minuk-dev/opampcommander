//nolint:dupl // MongoDB adapter pattern - similar structure to host is intentional.
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

var _ agentport.ContainerPersistencePort = (*ContainerMongoAdapter)(nil)

const (
	containerCollectionName = "containers"
)

// ContainerMongoAdapter implements the ContainerPersistencePort interface.
type ContainerMongoAdapter struct {
	collection *mongo.Collection
	common     commonEntityAdapter[entity.Container, string]
}

// NewContainerRepository creates a new instance of ContainerMongoAdapter.
func NewContainerRepository(
	mongoDatabase *mongo.Database,
	logger *slog.Logger,
) *ContainerMongoAdapter {
	collection := mongoDatabase.Collection(containerCollectionName)
	keyFunc := func(containerEntity *entity.Container) string {
		return containerEntity.Metadata.ID
	}
	keyQueryFunc := func(key string) any {
		return key
	}

	return &ContainerMongoAdapter{
		collection: collection,
		common: newCommonAdapter(
			logger,
			collection,
			entity.ContainerKeyFieldName,
			keyFunc,
			keyQueryFunc,
		),
	}
}

// GetContainer implements agentport.ContainerPersistencePort.
func (a *ContainerMongoAdapter) GetContainer(ctx context.Context, id string) (*agentmodel.Container, error) {
	containerEntity, err := a.common.get(ctx, id, nil)
	if err != nil {
		return nil, fmt.Errorf("get container: %w", err)
	}

	return containerEntity.ToDomain(), nil
}

// ListContainers implements agentport.ContainerPersistencePort.
func (a *ContainerMongoAdapter) ListContainers(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Container], error) {
	resp, err := a.common.list(ctx, options)
	if err != nil {
		return nil, err
	}

	items := make([]*agentmodel.Container, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, item.ToDomain())
	}

	return &model.ListResponse[*agentmodel.Container]{
		Items:              items,
		Continue:           resp.Continue,
		RemainingItemCount: resp.RemainingItemCount,
	}, nil
}

// PutContainer implements agentport.ContainerPersistencePort.
//
// PutContainer is an optimistic-concurrency write: an update only succeeds when the
// stored document's resourceVersion still equals the version the in-memory
// container was loaded with, otherwise it returns [model.ErrConflict] rather than
// clobbering a concurrent writer (another HA node that discovered the same
// container). On success the version is incremented and written back onto the
// passed container.
func (a *ContainerMongoAdapter) PutContainer(
	ctx context.Context, container *agentmodel.Container,
) (*agentmodel.Container, error) {
	expected := container.Metadata.ResourceVersion
	next := expected + 1

	containerEntity := entity.ContainerFromDomain(container)
	containerEntity.Metadata.ResourceVersion = next

	filter := bson.M{entity.ContainerKeyFieldName: container.Metadata.ID}

	err := casReplace(ctx, a.collection, filter, containerEntity, expected)
	if err != nil {
		return nil, fmt.Errorf("put container: %w", err)
	}

	container.Metadata.ResourceVersion = next

	return container, nil
}
