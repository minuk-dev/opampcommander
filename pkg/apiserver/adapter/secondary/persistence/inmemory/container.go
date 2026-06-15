//nolint:dupl // in-memory discovery repositories intentionally share this shape.
package inmemory

import (
	"context"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var _ agentport.ContainerPersistencePort = (*ContainerRepository)(nil)

// ContainerRepository is the in-memory implementation of
// [agentport.ContainerPersistencePort].
type ContainerRepository struct {
	store *store[string, *agentmodel.Container]
}

// NewContainerRepository creates a new in-memory ContainerRepository.
func NewContainerRepository() *ContainerRepository {
	return &ContainerRepository{
		store: newStore[string](cloneContainer, nil),
	}
}

// GetContainer implements agentport.ContainerPersistencePort.
func (r *ContainerRepository) GetContainer(_ context.Context, id string) (*agentmodel.Container, error) {
	return r.store.get(id, nil)
}

// PutContainer implements agentport.ContainerPersistencePort.
func (r *ContainerRepository) PutContainer(
	_ context.Context, container *agentmodel.Container,
) (*agentmodel.Container, error) {
	r.store.put(container.Metadata.ID, container)

	return container, nil
}

// ListContainers implements agentport.ContainerPersistencePort.
func (r *ContainerRepository) ListContainers(
	_ context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Container], error) {
	return r.store.list(options, nil)
}
