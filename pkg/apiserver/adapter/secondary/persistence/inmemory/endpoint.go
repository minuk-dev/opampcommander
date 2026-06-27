package inmemory

import (
	"context"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var _ agentport.EndpointPersistencePort = (*EndpointRepository)(nil)

// EndpointRepository is the in-memory implementation of
// [agentport.EndpointPersistencePort].
type EndpointRepository struct {
	store *store[namespacedName, *agentmodel.Endpoint]
}

// NewEndpointRepository creates a new in-memory EndpointRepository.
func NewEndpointRepository() *EndpointRepository {
	return &EndpointRepository{
		store: newStore[namespacedName](cloneEndpoint, func(e *agentmodel.Endpoint) *time.Time {
			return e.Metadata.DeletedAt
		}),
	}
}

// GetEndpoint implements agentport.EndpointPersistencePort.
func (r *EndpointRepository) GetEndpoint(
	_ context.Context, namespace string, name string, options *model.GetOptions,
) (*agentmodel.Endpoint, error) {
	return r.store.get(namespacedName{Namespace: namespace, Name: name}, options)
}

// PutEndpoint implements agentport.EndpointPersistencePort.
func (r *EndpointRepository) PutEndpoint(
	_ context.Context, endpoint *agentmodel.Endpoint,
) (*agentmodel.Endpoint, error) {
	r.store.put(namespacedName{
		Namespace: endpoint.Metadata.Namespace,
		Name:      endpoint.Metadata.Name,
	}, endpoint)

	return endpoint, nil
}

// ListEndpoints implements agentport.EndpointPersistencePort.
func (r *EndpointRepository) ListEndpoints(
	_ context.Context, namespace string, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Endpoint], error) {
	return r.store.list(options, func(endpoint *agentmodel.Endpoint) bool {
		return endpoint.Metadata.Namespace == namespace
	})
}
