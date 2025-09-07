package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	k8sclock "k8s.io/utils/clock"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agentgroup"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ port.AgentGroupUsecase = (*AgentGroupService)(nil)
var _ port.AgentGroupRelatedUsecase = (*AgentGroupService)(nil)

// AgentGroupPersistencePort is an alias for port.AgentGroupPersistencePort.
type AgentGroupPersistencePort = port.AgentGroupPersistencePort

// AgentByLabelsIndexer is an interface that defines the methods for listing agents by their labels.
type AgentByLabelsIndexer interface {
	// ListAgentsByIdentifyingAttributes lists agents by their identifying attributes such as labels.
	ListAgentsByAttributes(
		ctx context.Context,
		identifyingAttributes map[string]string,
		nonIdentifyingAttributes map[string]string,
		options *model.ListOptions,
	) (*model.ListResponse[*model.Agent], error)
}

// AgentGroupService is a struct that implements the AgentGroupUsecase interface.
type AgentGroupService struct {
	clock           clock.PassiveClock
	persistencePort AgentGroupPersistencePort
	agentIndexer    AgentByLabelsIndexer
}

// NewAgentGroupService creates a new instance of AgentGroupService.
func NewAgentGroupService() *AgentGroupService {
	return &AgentGroupService{
		persistencePort: nil,
		clock:           k8sclock.RealClock{},
		agentIndexer:    nil,
	}
}

// GetAgentGroup retrieves an agent group by its ID.
//
//nolint:wrapcheck
func (s *AgentGroupService) GetAgentGroup(
	ctx context.Context,
	id uuid.UUID,
) (*agentgroup.AgentGroup, error) {
	return s.persistencePort.GetAgentGroup(ctx, id)
}

// SaveAgentGroup saves the agent group.
//
//nolint:wrapcheck
func (s *AgentGroupService) SaveAgentGroup(
	ctx context.Context,
	agentGroup *agentgroup.AgentGroup,
) error {
	return s.persistencePort.PutAgentGroup(ctx, agentGroup)
}

// ListAgentGroups retrieves a list of agent groups with pagination options.
//
//nolint:wrapcheck
func (s *AgentGroupService) ListAgentGroups(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentgroup.AgentGroup], error) {
	return s.persistencePort.ListAgentGroups(ctx, options)
}

// DeleteAgentGroup marks an agent group as deleted.
func (s *AgentGroupService) DeleteAgentGroup(
	ctx context.Context,
	id uuid.UUID,
	deletedBy string,
) error {
	agentGroup, err := s.persistencePort.GetAgentGroup(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get agent group: %w", err)
	}

	agentGroup.MarkDeleted(s.clock.Now(), deletedBy)

	err = s.persistencePort.PutAgentGroup(ctx, agentGroup)
	if err != nil {
		return fmt.Errorf("failed to delete agent group: %w", err)
	}

	return nil
}

// ListAgentsByAgentGroup lists agents that belong to the specified agent group.
func (s *AgentGroupService) ListAgentsByAgentGroup(
	ctx context.Context,
	agentGroup *agentgroup.AgentGroup,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	agentSelector := agentGroup.Selector

	listResp, err := s.agentIndexer.ListAgentsByAttributes(
		ctx,
		agentSelector.IdentifyingAttributes,
		agentSelector.NonIdentifyingAttributes,
		options,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents by agent group: %w", err)
	}

	return listResp, nil
}
