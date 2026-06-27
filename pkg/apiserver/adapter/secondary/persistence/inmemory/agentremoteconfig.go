//nolint:dupl // namespaced CRUD repositories intentionally share this shape.
package inmemory

import (
	"context"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var _ agentport.AgentRemoteConfigPersistencePort = (*AgentRemoteConfigRepository)(nil)

// AgentRemoteConfigRepository is the in-memory implementation of
// [agentport.AgentRemoteConfigPersistencePort].
type AgentRemoteConfigRepository struct {
	store *store[namespacedName, *agentmodel.AgentRemoteConfig]
}

// NewAgentRemoteConfigRepository creates a new in-memory AgentRemoteConfigRepository.
func NewAgentRemoteConfigRepository() *AgentRemoteConfigRepository {
	return &AgentRemoteConfigRepository{
		store: newStore[namespacedName](cloneAgentRemoteConfig, func(arc *agentmodel.AgentRemoteConfig) *time.Time {
			return arc.Metadata.DeletedAt
		}),
	}
}

// GetAgentRemoteConfig implements agentport.AgentRemoteConfigPersistencePort.
func (r *AgentRemoteConfigRepository) GetAgentRemoteConfig(
	_ context.Context, namespace string, name string, options *model.GetOptions,
) (*agentmodel.AgentRemoteConfig, error) {
	return r.store.get(namespacedName{Namespace: namespace, Name: name}, options)
}

// PutAgentRemoteConfig implements agentport.AgentRemoteConfigPersistencePort.
func (r *AgentRemoteConfigRepository) PutAgentRemoteConfig(
	_ context.Context, config *agentmodel.AgentRemoteConfig,
) (*agentmodel.AgentRemoteConfig, error) {
	r.store.put(namespacedName{
		Namespace: config.Metadata.Namespace,
		Name:      config.Metadata.Name,
	}, config)

	return config, nil
}

// ListAgentRemoteConfigs implements agentport.AgentRemoteConfigPersistencePort.
func (r *AgentRemoteConfigRepository) ListAgentRemoteConfigs(
	_ context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentRemoteConfig], error) {
	return r.store.list(options, nil)
}
