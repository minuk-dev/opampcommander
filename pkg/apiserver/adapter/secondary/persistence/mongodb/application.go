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

var _ agentport.ApplicationPersistencePort = (*ApplicationMongoAdapter)(nil)

const (
	applicationCollectionName = "applications"
)

// ApplicationMongoAdapter implements the ApplicationPersistencePort interface.
type ApplicationMongoAdapter struct {
	collection *mongo.Collection
	common     commonEntityAdapter[entity.Application, string]
}

// NewApplicationRepository creates a new instance of ApplicationMongoAdapter.
func NewApplicationRepository(
	mongoDatabase *mongo.Database,
	logger *slog.Logger,
) *ApplicationMongoAdapter {
	collection := mongoDatabase.Collection(applicationCollectionName)
	keyFunc := func(applicationEntity *entity.Application) string {
		return applicationEntity.Metadata.ID
	}
	keyQueryFunc := func(key string) any {
		return key
	}

	return &ApplicationMongoAdapter{
		collection: collection,
		common: newCommonAdapter(
			logger,
			collection,
			entity.ApplicationKeyFieldName,
			keyFunc,
			keyQueryFunc,
			applicationSelectorSchema,
		),
	}
}

// GetApplication implements agentport.ApplicationPersistencePort.
func (a *ApplicationMongoAdapter) GetApplication(ctx context.Context, id string) (*agentmodel.Application, error) {
	applicationEntity, err := a.common.get(ctx, id, nil)
	if err != nil {
		return nil, fmt.Errorf("get application: %w", err)
	}

	return applicationEntity.ToDomain(), nil
}

// ListApplications implements agentport.ApplicationPersistencePort.
func (a *ApplicationMongoAdapter) ListApplications(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Application], error) {
	resp, err := a.common.list(ctx, options)
	if err != nil {
		return nil, err
	}

	items := make([]*agentmodel.Application, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, item.ToDomain())
	}

	return &model.ListResponse[*agentmodel.Application]{
		Items:              items,
		Continue:           resp.Continue,
		RemainingItemCount: resp.RemainingItemCount,
	}, nil
}

// PutApplication implements agentport.ApplicationPersistencePort.
//
// PutApplication is an optimistic-concurrency write: an update only succeeds when the
// stored document's resourceVersion still equals the version the in-memory
// application was loaded with, otherwise it returns [model.ErrConflict] rather than
// clobbering a concurrent writer (another HA node that discovered the same
// application). On success the version is incremented and written back onto the
// passed application.
func (a *ApplicationMongoAdapter) PutApplication(
	ctx context.Context, application *agentmodel.Application,
) (*agentmodel.Application, error) {
	expected := application.Metadata.ResourceVersion
	next := expected + 1

	applicationEntity := entity.ApplicationFromDomain(application)
	applicationEntity.Metadata.ResourceVersion = next

	filter := bson.M{entity.ApplicationKeyFieldName: application.Metadata.ID}

	err := casReplace(ctx, a.collection, filter, applicationEntity, expected)
	if err != nil {
		return nil, fmt.Errorf("put application: %w", err)
	}

	application.Metadata.ResourceVersion = next

	return application, nil
}
