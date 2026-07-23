package inmemory

import (
	"context"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
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
//
// Like the MongoDB adapter, this is an optimistic-concurrency write: an update
// (ResourceVersion > 0) succeeds only if the stored version still matches, else it
// returns [model.ErrConflict]. On success the version is incremented and written
// back onto the passed agent package.
func (r *AgentPackageRepository) PutAgentPackage(
	_ context.Context, agentPackage *agentmodel.AgentPackage,
) (*agentmodel.AgentPackage, error) {
	key := namespacedName{
		Namespace: agentPackage.Metadata.Namespace,
		Name:      agentPackage.Metadata.Name,
	}
	expected := agentPackage.Metadata.ResourceVersion
	next := expected + 1

	toStore := cloneAgentPackage(agentPackage)
	toStore.Metadata.ResourceVersion = next

	err := r.store.casPutOrCreate(key, toStore, expected, func(ap *agentmodel.AgentPackage) int64 {
		return ap.Metadata.ResourceVersion
	})
	if err != nil {
		return nil, err
	}

	agentPackage.Metadata.ResourceVersion = next

	return agentPackage, nil
}

// ListAgentPackages implements agentport.AgentPackagePersistencePort.
func (r *AgentPackageRepository) ListAgentPackages(
	_ context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentPackage], error) {
	return r.store.list(options, nil)
}
