package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/etcd/entity"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agentgroup"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var _ port.AgentGroupPersistencePort = (*AgentGroupEtcdAdapter)(nil)

type AgentGroupEtcdAdapter struct {
	client *clientv3.Client
	logger *slog.Logger
}

func NewAgentGroupEtcdAdapter(
	client *clientv3.Client,
	logger *slog.Logger,
) *AgentGroupEtcdAdapter {
	return &AgentGroupEtcdAdapter{
		client: client,
		logger: logger,
	}
}

// GetAgentGroup implements port.AgentGroupPersistencePort.
func (a *AgentGroupEtcdAdapter) GetAgentGroup(ctx context.Context, id uuid.UUID) (*agentgroup.AgentGroup, error) {
	getResponse, err := a.client.Get(ctx, getAgentGroupKey(id))
	if err != nil {
		return nil, fmt.Errorf("failed to get agent group from etcd: %w", err)
	}

	if getResponse.Count == 0 {
		return nil, port.ErrResourceNotExist
	}

	if getResponse.Count > 1 {
		// it should not happen, but if it does, we return an error
		// it's untestable because we always put a single agent group with a unique key
		return nil, port.ErrMultipleResourceExist
	}

	var agentGroup entity.AgentGroup

	err = json.Unmarshal(getResponse.Kvs[0].Value, &agentGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to decode agent group from received data: %w", err)
	}

	return agentGroup.ToDomain(), nil
}

// ListAgentGroups implements port.AgentGroupPersistencePort.
func (a *AgentGroupEtcdAdapter) ListAgentGroups(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*agentgroup.AgentGroup], error) {
	panic("unimplemented")
}

// PutAgentGroup implements port.AgentGroupPersistencePort.
func (a *AgentGroupEtcdAdapter) PutAgentGroup(ctx context.Context, agentGroup *agentgroup.AgentGroup) error {
	panic("unimplemented")
}

func getAgentGroupKey(id uuid.UUID) string {
	return "agentgroups/" + id.String()
}
