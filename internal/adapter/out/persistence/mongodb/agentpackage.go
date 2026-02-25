//nolint:dupl // MongoDB adapter pattern - similar structure is intentional
package mongodb

import (
	"context"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb/entity"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ port.AgentPackagePersistencePort = (*AgentPackageMongoAdapter)(nil)

const (
	agentPackageCollectionName = "agentpackages"
)

// AgentPackageMongoAdapter is a struct that implements the AgentPackagePersistencePort interface.
type AgentPackageMongoAdapter struct {
	common commonEntityAdapter[entity.AgentPackage, string]
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
		common: newCommonAdapter(
			logger,
			collection,
			entity.AgentPackageKeyFieldName,
			keyFunc,
			keyQueryFunc,
		),
	}
}

// GetAgentPackage implements port.AgentPackagePersistencePort.
func (a *AgentPackageMongoAdapter) GetAgentPackage(
	ctx context.Context, name string,
) (*model.AgentPackage, error) {
	en, err := a.common.get(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get agent package: %w", err)
	}

	return en.ToDomain(), nil
}

// ListAgentPackages implements port.AgentPackagePersistencePort.
func (a *AgentPackageMongoAdapter) ListAgentPackages(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*model.AgentPackage], error) {
	resp, err := a.common.list(ctx, options)
	if err != nil {
		return nil, err
	}

	items := make([]*model.AgentPackage, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, item.ToDomain())
	}

	return &model.ListResponse[*model.AgentPackage]{
		Items:              items,
		Continue:           resp.Continue,
		RemainingItemCount: resp.RemainingItemCount,
	}, nil
}

// PutAgentPackage implements port.AgentPackagePersistencePort.
func (a *AgentPackageMongoAdapter) PutAgentPackage(
	ctx context.Context, agentPackage *model.AgentPackage,
) (*model.AgentPackage, error) {
	en := entity.AgentPackageFromDomain(agentPackage)

	err := a.common.put(ctx, en)
	if err != nil {
		return nil, fmt.Errorf("put agent package: %w", err)
	}

	// Return the domain model directly instead of querying again
	// This avoids issues with soft-deleted documents not being found by GetAgentPackage
	return agentPackage, nil
}
