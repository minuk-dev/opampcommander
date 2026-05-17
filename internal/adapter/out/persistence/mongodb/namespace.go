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

var _ agentport.NamespacePersistencePort = (*NamespaceMongoAdapter)(nil)

const (
	namespaceCollectionName = "namespaces"
)

// NamespaceMongoAdapter implements the NamespacePersistencePort interface.
type NamespaceMongoAdapter struct {
	common commonEntityAdapter[entity.Namespace, string]
}

// NewNamespaceRepository creates a new instance of NamespaceMongoAdapter.
func NewNamespaceRepository(
	mongoDatabase *mongo.Database,
	logger *slog.Logger,
) *NamespaceMongoAdapter {
	collection := mongoDatabase.Collection(namespaceCollectionName)
	keyFunc := func(namespaceEntity *entity.Namespace) string {
		return namespaceEntity.Metadata.Name
	}
	keyQueryFunc := func(key string) any {
		return key
	}

	return &NamespaceMongoAdapter{
		common: newCommonAdapter(
			logger,
			collection,
			entity.NamespaceKeyFieldName,
			keyFunc,
			keyQueryFunc,
		),
	}
}

// GetNamespace implements agentport.NamespacePersistencePort.
func (a *NamespaceMongoAdapter) GetNamespace(
	ctx context.Context, name string, options *model.GetOptions,
) (*agentmodel.Namespace, error) {
	namespaceEntity, err := a.common.get(ctx, name, options)
	if err != nil {
		return nil, fmt.Errorf("get namespace: %w", err)
	}

	return namespaceEntity.ToDomain(), nil
}

// ListNamespaces implements agentport.NamespacePersistencePort.
func (a *NamespaceMongoAdapter) ListNamespaces(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Namespace], error) {
	resp, err := a.common.list(ctx, options)
	if err != nil {
		return nil, err
	}

	items := make([]*agentmodel.Namespace, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, item.ToDomain())
	}

	return &model.ListResponse[*agentmodel.Namespace]{
		Items:              items,
		Continue:           resp.Continue,
		RemainingItemCount: resp.RemainingItemCount,
	}, nil
}

// PutNamespace implements agentport.NamespacePersistencePort.
func (a *NamespaceMongoAdapter) PutNamespace(
	ctx context.Context, namespace *agentmodel.Namespace,
) (*agentmodel.Namespace, error) {
	namespaceEntity := entity.NamespaceFromDomain(namespace)

	err := a.common.put(ctx, namespaceEntity)
	if err != nil {
		return nil, fmt.Errorf("put namespace: %w", err)
	}

	return namespace, nil
}
