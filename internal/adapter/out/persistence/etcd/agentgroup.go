//nolint:dupl
package etcd

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/etcd/entity"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agentgroup"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ port.AgentGroupPersistencePort = (*AgentGroupEtcdAdapter)(nil)

// AgentGroupEtcdAdapter is a struct that implements the AgentGroupPersistencePort interface.
type AgentGroupEtcdAdapter struct {
	common commonAdapter[agentgroup.AgentGroup]
}

// NewAgentGroupEtcdAdapter creates a new instance of AgentGroupEtcdAdapter.
func NewAgentGroupEtcdAdapter(
	client *clientv3.Client,
	logger *slog.Logger,
) *AgentGroupEtcdAdapter {
	ToEntityFunc := func(domain *agentgroup.AgentGroup) (Entity[agentgroup.AgentGroup], error) {
		return entity.AgentGroupFromDomain(domain), nil
	}

	keyFunc := func(domain *agentgroup.AgentGroup) string {
		return domain.UID.String()
	}

	return &AgentGroupEtcdAdapter{
		common: newCommonAdapter(
			client,
			logger,
			ToEntityFunc,
			"agentgroups/",
			keyFunc,
		),
	}
}

// GetAgentGroup implements port.AgentGroupPersistencePort.
func (a *AgentGroupEtcdAdapter) GetAgentGroup(
	ctx context.Context, id uuid.UUID,
) (*agentgroup.AgentGroup, error) {
	return a.common.get(ctx, id.String())
}

// ListAgentGroups implements port.AgentGroupPersistencePort.
func (a *AgentGroupEtcdAdapter) ListAgentGroups(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentgroup.AgentGroup], error) {
	return a.common.list(ctx, options)
}

// PutAgentGroup implements port.AgentGroupPersistencePort.
func (a *AgentGroupEtcdAdapter) PutAgentGroup(
	ctx context.Context, agentGroup *agentgroup.AgentGroup,
) error {
	return a.common.put(ctx, agentGroup)
}
