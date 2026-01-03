package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

const (
	agentGroupServiceName = "AgentGroupService"
	// ChangedAgentGroupBufferSize is the buffer size for the changed agent group channel.
	ChangedAgentGroupBufferSize = 100
	// PropagationChunkSize is the number of agents to process in each batch when propagating changes.
	PropagationChunkSize = 50
)

var _ port.AgentGroupUsecase = (*AgentGroupService)(nil)
var _ port.AgentGroupRelatedUsecase = (*AgentGroupService)(nil)

// AgentGroupPersistencePort is an alias for port.AgentGroupPersistencePort.
type AgentGroupPersistencePort = port.AgentGroupPersistencePort

// AgentGroupService is a struct that implements the AgentGroupUsecase interface.
type AgentGroupService struct {
	persistencePort AgentGroupPersistencePort
	agentUsecase    port.AgentUsecase
	logger          *slog.Logger

	changedAgentGroupCh chan *model.AgentGroup
}

// NewAgentGroupService creates a new instance of AgentGroupService.
func NewAgentGroupService(
	persistencePort AgentGroupPersistencePort,
	agentUsecase port.AgentUsecase,
	logger *slog.Logger,
) *AgentGroupService {
	return &AgentGroupService{
		persistencePort:     persistencePort,
		agentUsecase:        agentUsecase,
		logger:              logger,
		changedAgentGroupCh: make(chan *model.AgentGroup, ChangedAgentGroupBufferSize),
	}
}

// Name implements lifecycle.Runner.
func (s *AgentGroupService) Name() string {
	return agentGroupServiceName
}

// Run implements lifecycle.Runner.
func (s *AgentGroupService) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case agentGroup := <-s.changedAgentGroupCh:
			err := s.updateAgentsByAgentGroup(ctx, agentGroup)
			if err != nil {
				s.logger.Error("failed to propagate agent group changes to agents",
					slog.String("agent_group", agentGroup.Metadata.Name),
					slog.String("error", err.Error()),
				)
			}
		}
	}
	// unreachable
}

// GetAgentGroup retrieves an agent group by its ID.
func (s *AgentGroupService) GetAgentGroup(
	ctx context.Context,
	name string,
) (*model.AgentGroup, error) {
	agentGroup, err := s.persistencePort.GetAgentGroup(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get agent group: %w", err)
	}

	return agentGroup, nil
}

// SaveAgentGroup saves the agent group.
func (s *AgentGroupService) SaveAgentGroup(
	ctx context.Context,
	name string,
	agentGroup *model.AgentGroup,
) (*model.AgentGroup, error) {
	agentGroup, err := s.persistencePort.PutAgentGroup(ctx, name, agentGroup)
	if err != nil {
		return nil, fmt.Errorf("save agent group: %w", err)
	}

	err = s.propagateAgentGroupChangesToAgents(ctx, agentGroup)
	if err != nil {
		return nil, fmt.Errorf("propagate agent group changes to agents: %w", err)
	}

	return agentGroup, nil
}

// ListAgentGroups retrieves a list of agent groups with pagination options.
func (s *AgentGroupService) ListAgentGroups(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.AgentGroup], error) {
	resp, err := s.persistencePort.ListAgentGroups(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("list agent groups: %w", err)
	}

	return resp, nil
}

// DeleteAgentGroup marks an agent group as deleted.
func (s *AgentGroupService) DeleteAgentGroup(
	ctx context.Context,
	name string,
	deletedAt time.Time,
	deletedBy string,
) error {
	agentGroup, err := s.persistencePort.GetAgentGroup(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get agent group: %w", err)
	}

	agentGroup.MarkDeleted(deletedAt, deletedBy)

	_, err = s.persistencePort.PutAgentGroup(ctx, name, agentGroup)
	if err != nil {
		return fmt.Errorf("failed to delete agent group: %w", err)
	}

	return nil
}

// ListAgentsByAgentGroup lists agents that belong to the specified agent group.
func (s *AgentGroupService) ListAgentsByAgentGroup(
	ctx context.Context,
	agentGroup *model.AgentGroup,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	agentSelector := agentGroup.Metadata.Selector

	listResp, err := s.agentUsecase.ListAgentsBySelector(
		ctx,
		agentSelector,
		options,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents by agent group: %w", err)
	}

	return listResp, nil
}

// GetAgentGroupsForAgent retrieves all agent groups that match the agent's attributes.
func (s *AgentGroupService) GetAgentGroupsForAgent(
	ctx context.Context,
	agent *model.Agent,
) ([]*model.AgentGroup, error) {
	// Get all agent groups
	allGroups, err := s.persistencePort.ListAgentGroups(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent groups: %w", err)
	}

	// Filter groups that match the agent
	var matchingGroups []*model.AgentGroup

	for _, group := range allGroups.Items {
		if group.IsDeleted() {
			continue
		}

		if matchesSelector(agent, group.Metadata.Selector) {
			matchingGroups = append(matchingGroups, group)
		}
	}

	return matchingGroups, nil
}

// matchesSelector checks if an agent matches the given selector.
func matchesSelector(agent *model.Agent, selector model.AgentSelector) bool {
	// Check identifying attributes
	for key, value := range selector.IdentifyingAttributes {
		agentValue, ok := agent.Metadata.Description.IdentifyingAttributes[key]
		if !ok || agentValue != value {
			return false
		}
	}

	// Check non-identifying attributes
	for key, value := range selector.NonIdentifyingAttributes {
		agentValue, ok := agent.Metadata.Description.NonIdentifyingAttributes[key]
		if !ok || agentValue != value {
			return false
		}
	}

	return true
}

func (s *AgentGroupService) propagateAgentGroupChangesToAgents(
	ctx context.Context,
	agentGroup *model.AgentGroup,
) error {
	select {
	case s.changedAgentGroupCh <- agentGroup:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	}
}

func (s *AgentGroupService) updateAgentsByAgentGroup(
	ctx context.Context,
	agentGroup *model.AgentGroup,
) error {
	var continueToken string

	for {
		listOptions := &model.ListOptions{
			Limit:    PropagationChunkSize,
			Continue: continueToken,
		}

		agentsResp, err := s.ListAgentsByAgentGroup(ctx, agentGroup, listOptions)
		if err != nil {
			return fmt.Errorf("list agents by agent group: %w", err)
		}

		if len(agentsResp.Items) == 0 {
			// No more agents to process
			break
		}

		for _, agent := range agentsResp.Items {
			// Here you can implement the logic to update the agent based on the agent group changes.
			// For example, you might want to update the agent's configuration or metadata.
			err := agent.ApplyRemoteConfig(
				agentGroup.Spec.AgentRemoteConfig.Value,
				agentGroup.Spec.AgentRemoteConfig.ContentType,
				agentGroup.Metadata.Priority,
			)
			if err != nil {
				return fmt.Errorf("apply remote config to agent %s: %w", agent.Metadata.InstanceUID, err)
			}

			// After updating the agent, save it back to the persistence layer.
			err = s.agentUsecase.SaveAgent(ctx, agent)
			if err != nil {
				return fmt.Errorf("save updated agent: %w", err)
			}
		}

		continueToken = agentsResp.Continue
	}

	return nil
}
