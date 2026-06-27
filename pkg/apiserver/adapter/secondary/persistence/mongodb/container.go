//nolint:dupl // MongoDB adapter pattern - similar structure to host is intentional.
package mongodb

import (
	"context"
	"fmt"
	"log/slog"

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
	common commonEntityAdapter[entity.Container, string]
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
func (a *ContainerMongoAdapter) PutContainer(
	ctx context.Context, container *agentmodel.Container,
) (*agentmodel.Container, error) {
	containerEntity := entity.ContainerFromDomain(container)

	err := a.common.put(ctx, containerEntity)
	if err != nil {
		return nil, fmt.Errorf("put container: %w", err)
	}

	return container, nil
}
