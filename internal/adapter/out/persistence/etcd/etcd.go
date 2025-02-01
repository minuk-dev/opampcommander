package etcd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/etcd/entity"
	domainmodel "github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ domainport.AgentPersistencePort = (*AgentEtcdAdapter)(nil)

type AgentEtcdAdapter struct {
	client *clientv3.Client
}

func NewAgentEtcdAdapter(
	client *clientv3.Client,
) *AgentEtcdAdapter {
	return &AgentEtcdAdapter{
		client: client,
	}
}

func (a *AgentEtcdAdapter) GetAgent(ctx context.Context, instanceUID uuid.UUID) (*domainmodel.Agent, error) {
	getResponse, err := a.client.Get(ctx, getAgentKey(instanceUID))
	if err != nil {
		return nil, fmt.Errorf("failed to get agent from etcd: %w", err)
	}

	if getResponse.Count == 0 {
		return nil, domainport.ErrAgentNotExist
	}

	if getResponse.Count > 1 {
		return nil, domainport.ErrMultipleAgentExist
	}

	var agent entity.Agent

	err = json.Unmarshal(getResponse.Kvs[0].Value, &agent)
	if err != nil {
		return nil, fmt.Errorf("failed to decode agent from received data: %w", err)
	}

	return agent.ToDomain(), nil
}

func (a *AgentEtcdAdapter) ListAgents(ctx context.Context) ([]*domainmodel.Agent, error) {
	getResponse, err := a.client.Get(ctx, "agents/", clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get agents from etcd: %w", err)
	}

	agents := make([]*domainmodel.Agent, 0, getResponse.Count)

	for _, kv := range getResponse.Kvs {
		var agent entity.Agent

		err = json.Unmarshal(kv.Value, &agent)
		if err != nil {
			return nil, fmt.Errorf("failed to decode agent from received data: %w", err)
		}

		agents = append(agents, agent.ToDomain())
	}

	return agents, nil
}

func (a *AgentEtcdAdapter) PutAgent(ctx context.Context, agent *domainmodel.Agent) error {
	agentEntity := entity.AgentFromDomain(agent)

	encoded, err := json.Marshal(agentEntity)
	if err != nil {
		return fmt.Errorf("failed to encode agent: %w", err)
	}

	_, err = a.client.Put(ctx, getAgentKey(agent.InstanceUID), string(encoded))
	if err != nil {
		return fmt.Errorf("failed to put agent to etcd: %w", err)
	}

	return nil
}

func getAgentKey(instanceUID uuid.UUID) string {
	return "agents/" + instanceUID.String()
}
