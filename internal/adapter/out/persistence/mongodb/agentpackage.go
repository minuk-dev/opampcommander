//nolint:dupl // MongoDB adapter pattern - similar structure is intentional
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

var _ agentport.AgentPackagePersistencePort = (*AgentPackageMongoAdapter)(nil)

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

// GetAgentPackage implements agentport.AgentPackagePersistencePort.
func (a *AgentPackageMongoAdapter) GetAgentPackage(
	ctx context.Context, name string,
) (*agentmodel.AgentPackage, error) {
	en, err := a.common.get(ctx, name, nil)
	if err != nil {
		return nil, fmt.Errorf("get agent package: %w", err)
	}

	return en.ToDomain(), nil
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
	en := entity.AgentPackageFromDomain(agentPackage)

	err := a.common.put(ctx, en)
	if err != nil {
		return nil, fmt.Errorf("put agent package: %w", err)
	}

	// Return the domain model directly instead of querying again
	// This avoids issues with soft-deleted documents not being found by GetAgentPackage
	return agentPackage, nil
}
