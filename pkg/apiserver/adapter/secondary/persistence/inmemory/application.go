//nolint:dupl // in-memory discovery repositories intentionally share this shape.
package inmemory

import (
	"context"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var _ agentport.ApplicationPersistencePort = (*ApplicationRepository)(nil)

// ApplicationRepository is the in-memory implementation of
// [agentport.ApplicationPersistencePort].
type ApplicationRepository struct {
	store *store[string, *agentmodel.Application]
}

// NewApplicationRepository creates a new in-memory ApplicationRepository.
func NewApplicationRepository() *ApplicationRepository {
	return &ApplicationRepository{
		store: newStore[string](cloneApplication, nil, (*agentmodel.Application).SelectorValues, hasLabels),
	}
}

// GetApplication implements agentport.ApplicationPersistencePort.
func (r *ApplicationRepository) GetApplication(_ context.Context, id string) (*agentmodel.Application, error) {
	return r.store.get(id, nil)
}

// PutApplication implements agentport.ApplicationPersistencePort.
//
// Like the MongoDB adapter, this is an optimistic-concurrency write: an update
// (ResourceVersion > 0) succeeds only if the stored version still matches, else it
// returns [model.ErrConflict]. On success the version is incremented and written
// back onto the passed application.
func (r *ApplicationRepository) PutApplication(
	_ context.Context, application *agentmodel.Application,
) (*agentmodel.Application, error) {
	expected := application.Metadata.ResourceVersion
	next := expected + 1

	toStore := cloneApplication(application)
	toStore.Metadata.ResourceVersion = next

	err := r.store.casPutOrCreate(application.Metadata.ID, toStore, expected, func(c *agentmodel.Application) int64 {
		return c.Metadata.ResourceVersion
	})
	if err != nil {
		return nil, err
	}

	application.Metadata.ResourceVersion = next

	return application, nil
}

// ListApplications implements agentport.ApplicationPersistencePort.
func (r *ApplicationRepository) ListApplications(
	_ context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Application], error) {
	return r.store.list(options, nil)
}
