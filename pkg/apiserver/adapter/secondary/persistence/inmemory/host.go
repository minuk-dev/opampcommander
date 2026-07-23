//nolint:dupl // in-memory discovery repositories intentionally share this shape.
package inmemory

import (
	"context"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var _ agentport.HostPersistencePort = (*HostRepository)(nil)

// HostRepository is the in-memory implementation of
// [agentport.HostPersistencePort].
type HostRepository struct {
	store *store[string, *agentmodel.Host]
}

// NewHostRepository creates a new in-memory HostRepository.
func NewHostRepository() *HostRepository {
	return &HostRepository{
		store: newStore[string](cloneHost, nil),
	}
}

// GetHost implements agentport.HostPersistencePort.
func (r *HostRepository) GetHost(_ context.Context, id string) (*agentmodel.Host, error) {
	return r.store.get(id, nil)
}

// PutHost implements agentport.HostPersistencePort.
//
// Like the MongoDB adapter, this is an optimistic-concurrency write: an update
// (ResourceVersion > 0) succeeds only if the stored version still matches, else it
// returns [model.ErrConflict]. On success the version is incremented and written
// back onto the passed host.
func (r *HostRepository) PutHost(_ context.Context, host *agentmodel.Host) (*agentmodel.Host, error) {
	expected := host.Metadata.ResourceVersion
	next := expected + 1

	toStore := cloneHost(host)
	toStore.Metadata.ResourceVersion = next

	err := r.store.casPutOrCreate(host.Metadata.ID, toStore, expected, func(h *agentmodel.Host) int64 {
		return h.Metadata.ResourceVersion
	})
	if err != nil {
		return nil, err
	}

	host.Metadata.ResourceVersion = next

	return host, nil
}

// ListHosts implements agentport.HostPersistencePort.
func (r *HostRepository) ListHosts(
	_ context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Host], error) {
	return r.store.list(options, nil)
}
