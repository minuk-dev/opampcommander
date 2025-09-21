package etcd

import (
	"context"
	"log/slog"

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

	CreateNewEmptyEntityFunc := func() Entity[agentgroup.AgentGroup] {
		//exhaustruct:ignore
		return &entity.AgentGroup{}
	}

	keyFunc := func(domain *agentgroup.AgentGroup) string {
		return domain.Name
	}

	return &AgentGroupEtcdAdapter{
		common: newCommonAdapter(
			client,
			logger,
			ToEntityFunc,
			CreateNewEmptyEntityFunc,
			"agentgroups/",
			keyFunc,
		),
	}
}

// GetAgentGroup implements port.AgentGroupPersistencePort.
func (a *AgentGroupEtcdAdapter) GetAgentGroup(
	ctx context.Context, name string,
) (*agentgroup.AgentGroup, error) {
	return a.common.get(ctx, name)
}

// ListAgentGroups implements port.AgentGroupPersistencePort.
func (a *AgentGroupEtcdAdapter) ListAgentGroups(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentgroup.AgentGroup], error) {
	return a.common.list(ctx, options)
}

// PutAgentGroup implements port.AgentGroupPersistencePort.
//
//nolint:godox,revive
func (a *AgentGroupEtcdAdapter) PutAgentGroup(
	ctx context.Context, name string, agentGroup *agentgroup.AgentGroup,
) error {
	// TODO: name should be used to save the agent group with the given name.
	// TODO: https://github.com/minuk-dev/opampcommander/issues/145
	// Because some update operations may change the name of the agent group.
	return a.common.put(ctx, agentGroup)
}
