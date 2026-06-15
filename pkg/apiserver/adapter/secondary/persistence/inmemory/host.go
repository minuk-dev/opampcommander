//nolint:dupl // in-memory discovery repositories intentionally share this shape.
package inmemory

import (
	"context"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
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
func (r *HostRepository) PutHost(_ context.Context, host *agentmodel.Host) (*agentmodel.Host, error) {
	r.store.put(host.Metadata.ID, host)

	return host, nil
}

// ListHosts implements agentport.HostPersistencePort.
func (r *HostRepository) ListHosts(
	_ context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Host], error) {
	return r.store.list(options, nil)
}
