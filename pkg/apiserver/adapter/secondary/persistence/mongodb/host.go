//nolint:dupl // MongoDB adapter pattern - similar structure to container is intentional.
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

var _ agentport.HostPersistencePort = (*HostMongoAdapter)(nil)

const (
	hostCollectionName = "hosts"
)

// HostMongoAdapter implements the HostPersistencePort interface.
type HostMongoAdapter struct {
	common commonEntityAdapter[entity.Host, string]
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
func (a *HostMongoAdapter) PutHost(
	ctx context.Context, host *agentmodel.Host,
) (*agentmodel.Host, error) {
	hostEntity := entity.HostFromDomain(host)

	err := a.common.put(ctx, hostEntity)
	if err != nil {
		return nil, fmt.Errorf("put host: %w", err)
	}

	return host, nil
}
