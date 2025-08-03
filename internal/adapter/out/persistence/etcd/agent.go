package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/samber/lo"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/etcd/entity"
	domainmodel "github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ domainport.AgentPersistencePort = (*AgentEtcdAdapter)(nil)

// AgentEtcdAdapter is a struct that implements the AgentPersistencePort interface.
type AgentEtcdAdapter struct {
	client *clientv3.Client
	logger *slog.Logger
}

// NewAgentEtcdAdapter creates a new instance of AgentEtcdAdapter.
func NewAgentEtcdAdapter(
	client *clientv3.Client,
	logger *slog.Logger,
) *AgentEtcdAdapter {
	return &AgentEtcdAdapter{
		client: client,
		logger: logger,
	}
}

// GetAgent retrieves an agent by its instance UID.
func (a *AgentEtcdAdapter) GetAgent(ctx context.Context, instanceUID uuid.UUID) (*domainmodel.Agent, error) {
	getResponse, err := a.client.Get(ctx, getAgentKey(instanceUID))
	if err != nil {
		return nil, fmt.Errorf("failed to get agent from etcd: %w", err)
	}

	if getResponse.Count == 0 {
		return nil, domainport.ErrAgentNotExist
	}

	if getResponse.Count > 1 {
		// it should not happen, but if it does, we return an error
		// it's untestable because we always put a single agent with a unique key
		return nil, domainport.ErrMultipleAgentExist
	}

	var agent entity.Agent

	err = json.Unmarshal(getResponse.Kvs[0].Value, &agent)
	if err != nil {
		return nil, fmt.Errorf("failed to decode agent from received data: %w", err)
	}

	return agent.ToDomain(), nil
}

// ListAgents retrieves all agents from the persistence layer.
func (a *AgentEtcdAdapter) ListAgents(
	ctx context.Context,
	options *domainmodel.ListOptions,
) (*domainmodel.ListResponse[*domainmodel.Agent], error) {
	if options == nil {
		options = &domainmodel.ListOptions{
			Limit:    0,  // 0 means no limit
			Continue: "", // empty continue token means start from the beginning
		}
	}

	startKey := "agents/" + options.Continue

	getResponse, err := a.client.Get(
		ctx,
		startKey,
		clientv3.WithLimit(options.Limit),
		clientv3.WithRange("agents/\xFF"), // Use a range to get all keys under "agents/"
	)
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
	// Use a null byte to ensure the next key is lexicographically greater
	var continueKey string
	if len(agents) > 0 {
		continueKey = lo.LastOrEmpty(agents).InstanceUID.String() + "\x00"
	}

	return &domainmodel.ListResponse[*domainmodel.Agent]{
		RemainingItemCount: getResponse.Count - int64(len(agents)),
		Continue:           continueKey,
		Items:              agents,
	}, nil
}

// PutAgent saves the agent to the persistence layer.
func (a *AgentEtcdAdapter) PutAgent(ctx context.Context, agent *domainmodel.Agent) error {
	agentEntity := entity.AgentFromDomain(agent)

	encoded, err := json.Marshal(agentEntity)
	if err != nil {
		return fmt.Errorf("failed to encode agent: %w", err)
	}

	a.logger.Debug("PutAgent",
		slog.String("key", getAgentKey(agent.InstanceUID)),
		slog.String("value", string(encoded)),
	)

	_, err = a.client.Put(ctx, getAgentKey(agent.InstanceUID), string(encoded))
	if err != nil {
		return fmt.Errorf("failed to put agent to etcd: %w", err)
	}

	return nil
}

func getAgentKey(instanceUID uuid.UUID) string {
	return "agents/" + instanceUID.String()
}
