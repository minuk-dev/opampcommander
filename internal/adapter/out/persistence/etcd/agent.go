//nolint:dupl
package etcd

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/etcd/entity"
	domainmodel "github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ domainport.AgentPersistencePort = (*AgentEtcdAdapter)(nil)

// AgentEtcdAdapter is a struct that implements the AgentPersistencePort interface.
type AgentEtcdAdapter struct {
	common commonAdapter[domainmodel.Agent]
}

// NewAgentEtcdAdapter creates a new instance of AgentEtcdAdapter.
func NewAgentEtcdAdapter(
	client *clientv3.Client,
	logger *slog.Logger,
) *AgentEtcdAdapter {
	ToEntityFunc := func(domain *domainmodel.Agent) (Entity[domainmodel.Agent], error) {
		return entity.AgentFromDomain(domain), nil
	}

	CreateEmptyEntityFunc := func() Entity[domainmodel.Agent] {
		//exhaustruct:ignore
		return &entity.Agent{}
	}

	keyFunc := func(domain *domainmodel.Agent) string {
		return domain.InstanceUID.String()
	}

	return &AgentEtcdAdapter{
		common: newCommonAdapter(
			client,
			logger,
			ToEntityFunc,
			CreateEmptyEntityFunc,
			"agents/",
			keyFunc,
		),
	}
}

// GetAgent retrieves an agent by its instance UID.
func (a *AgentEtcdAdapter) GetAgent(ctx context.Context, instanceUID uuid.UUID) (*domainmodel.Agent, error) {
	return a.common.get(ctx, instanceUID.String())
}

// ListAgents retrieves all agents from the persistence layer.
func (a *AgentEtcdAdapter) ListAgents(
	ctx context.Context,
	options *domainmodel.ListOptions,
) (*domainmodel.ListResponse[*domainmodel.Agent], error) {
	return a.common.list(ctx, options)
}

// PutAgent saves the agent to the persistence layer.
func (a *AgentEtcdAdapter) PutAgent(ctx context.Context, agent *domainmodel.Agent) error {
	return a.common.put(ctx, agent)
}
