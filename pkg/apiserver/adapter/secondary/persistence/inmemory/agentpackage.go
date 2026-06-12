//nolint:dupl // namespaced CRUD repositories intentionally share this shape.
package inmemory

import (
	"context"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var _ agentport.AgentPackagePersistencePort = (*AgentPackageRepository)(nil)

// AgentPackageRepository is the in-memory implementation of
// [agentport.AgentPackagePersistencePort].
type AgentPackageRepository struct {
	store *store[namespacedName, *agentmodel.AgentPackage]
}

// NewAgentPackageRepository creates a new in-memory AgentPackageRepository.
func NewAgentPackageRepository() *AgentPackageRepository {
	return &AgentPackageRepository{
		store: newStore[namespacedName](cloneAgentPackage, func(ap *agentmodel.AgentPackage) *time.Time {
			return ap.Metadata.DeletedAt
		}),
	}
}

// GetAgentPackage implements agentport.AgentPackagePersistencePort.
func (r *AgentPackageRepository) GetAgentPackage(
	_ context.Context, namespace string, name string, options *model.GetOptions,
) (*agentmodel.AgentPackage, error) {
	return r.store.get(namespacedName{Namespace: namespace, Name: name}, options)
}

// PutAgentPackage implements agentport.AgentPackagePersistencePort.
func (r *AgentPackageRepository) PutAgentPackage(
	_ context.Context, agentPackage *agentmodel.AgentPackage,
) (*agentmodel.AgentPackage, error) {
	r.store.put(namespacedName{
		Namespace: agentPackage.Metadata.Namespace,
		Name:      agentPackage.Metadata.Name,
	}, agentPackage)

	return agentPackage, nil
}

// ListAgentPackages implements agentport.AgentPackagePersistencePort.
func (r *AgentPackageRepository) ListAgentPackages(
	_ context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentPackage], error) {
	return r.store.list(options, nil)
}
