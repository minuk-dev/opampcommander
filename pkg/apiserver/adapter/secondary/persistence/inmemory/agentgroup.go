package inmemory

import (
	"context"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var _ agentport.AgentGroupPersistencePort = (*AgentGroupRepository)(nil)

// AgentGroupRepository is the in-memory implementation of
// [agentport.AgentGroupPersistencePort].
//
// Like the MongoDB adapter it recomputes per-group agent statistics
// (total/connected/healthy/...) on every read from the live agent store, so the
// counts reflect current agent state rather than whatever was stored.
type AgentGroupRepository struct {
	store     *store[namespacedName, *agentmodel.AgentGroup]
	agentRepo *AgentRepository
}

// NewAgentGroupRepository creates a new in-memory AgentGroupRepository. It reads
// the agent store (via agentRepo) to compute group statistics.
func NewAgentGroupRepository(agentRepo *AgentRepository) *AgentGroupRepository {
	return &AgentGroupRepository{
		store: newStore[namespacedName](func(ag *agentmodel.AgentGroup) *time.Time {
			return &ag.Metadata.DeletedAt
		}),
		agentRepo: agentRepo,
	}
}

// GetAgentGroup implements agentport.AgentGroupPersistencePort.
func (r *AgentGroupRepository) GetAgentGroup(
	_ context.Context, namespace string, name string, options *model.GetOptions,
) (*agentmodel.AgentGroup, error) {
	agentGroup, err := r.store.get(namespacedName{Namespace: namespace, Name: name}, options)
	if err != nil {
		return nil, err
	}

	r.applyStatistics(agentGroup)

	return agentGroup, nil
}

// ListAgentGroups implements agentport.AgentGroupPersistencePort.
func (r *AgentGroupRepository) ListAgentGroups(
	_ context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentGroup], error) {
	resp, err := r.store.list(options, nil)
	if err != nil {
		return nil, err
	}

	for _, agentGroup := range resp.Items {
		r.applyStatistics(agentGroup)
	}

	return resp, nil
}

// PutAgentGroup implements agentport.AgentGroupPersistencePort.
func (r *AgentGroupRepository) PutAgentGroup(
	_ context.Context, namespace string, name string, agentGroup *agentmodel.AgentGroup,
) (*agentmodel.AgentGroup, error) {
	r.store.put(namespacedName{Namespace: namespace, Name: name}, agentGroup)

	if !agentGroup.IsDeleted() {
		r.applyStatistics(agentGroup)
	}

	return agentGroup, nil
}

// applyStatistics recomputes the agent-count status fields for the group from
// the current agent store, mirroring the MongoDB aggregation pipeline. Healthy
// and unhealthy counts only consider connected agents.
func (r *AgentGroupRepository) applyStatistics(agentGroup *agentmodel.AgentGroup) {
	agents := r.agentRepo.agentsMatchingSelector(agentGroup.Spec.Selector)

	//exhaustruct:ignore
	stats := agentmodel.AgentGroupStatus{}
	stats.Conditions = agentGroup.Status.Conditions

	for _, agent := range agents {
		stats.NumAgents++

		if !r.agentRepo.isConnected(agent) {
			stats.NumNotConnectedAgents++

			continue
		}

		stats.NumConnectedAgents++

		if agent.Status.ComponentHealth.Healthy {
			stats.NumHealthyAgents++
		} else {
			stats.NumUnhealthyAgents++
		}
	}

	agentGroup.Status = stats
}
