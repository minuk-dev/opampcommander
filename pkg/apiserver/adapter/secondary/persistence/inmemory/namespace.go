package inmemory

import (
	"context"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var _ agentport.NamespacePersistencePort = (*NamespaceRepository)(nil)

// NamespaceRepository is the in-memory implementation of
// [agentport.NamespacePersistencePort].
type NamespaceRepository struct {
	store *store[string, *agentmodel.Namespace]
}

// NewNamespaceRepository creates a new in-memory NamespaceRepository.
func NewNamespaceRepository() *NamespaceRepository {
	return &NamespaceRepository{
		store: newStore[string](cloneNamespace, func(ns *agentmodel.Namespace) *time.Time {
			return ns.Metadata.DeletedAt
		}),
	}
}

// GetNamespace implements agentport.NamespacePersistencePort.
func (r *NamespaceRepository) GetNamespace(
	_ context.Context, name string, options *model.GetOptions,
) (*agentmodel.Namespace, error) {
	return r.store.get(name, options)
}

// PutNamespace implements agentport.NamespacePersistencePort.
func (r *NamespaceRepository) PutNamespace(
	_ context.Context, namespace *agentmodel.Namespace,
) (*agentmodel.Namespace, error) {
	r.store.put(namespace.Metadata.Name, namespace)

	return namespace, nil
}

// ListNamespaces implements agentport.NamespacePersistencePort.
func (r *NamespaceRepository) ListNamespaces(
	_ context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Namespace], error) {
	return r.store.list(options, nil)
}
