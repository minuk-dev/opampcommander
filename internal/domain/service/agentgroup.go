package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agentgroup"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ port.AgentGroupUsecase = (*AgentGroupService)(nil)
var _ port.AgentGroupRelatedUsecase = (*AgentGroupService)(nil)

type AgentGroupPersistencePort = port.AgentGroupPersistencePort

type AgentByLabelsIndexer interface {
	// ListAgentsByIdentifyingAttributes lists agents by their identifying attributes such as labels.
	ListAgentsByAttributes(
		ctx context.Context,
		identifyingAttributes map[string]string,
		nonIdentifyingAttributes map[string]string,
		options *model.ListOptions,
	) (*model.ListResponse[*model.Agent], error)
}

type AgentGroupService struct {
	clock           clock.PassiveClock
	persistencePort AgentGroupPersistencePort
	agentIndexer    AgentByLabelsIndexer
}

func NewAgentGroupService() *AgentGroupService {
	return &AgentGroupService{
		persistencePort: nil,
	}
}

func (s *AgentGroupService) GetAgentGroup(ctx context.Context, id uuid.UUID) (*agentgroup.AgentGroup, error) {
	agentGroup, err := s.persistencePort.GetAgentGroup(ctx, id)
	if err != nil {
		return nil, err
	}

	return agentGroup, nil
}

func (s *AgentGroupService) SaveAgentGroup(ctx context.Context, agentGroup *agentgroup.AgentGroup) error {
	return s.persistencePort.PutAgentGroup(ctx, agentGroup)
}

func (s *AgentGroupService) ListAgentGroups(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*agentgroup.AgentGroup], error) {
	return s.persistencePort.ListAgentGroups(ctx, options)
}

func (s *AgentGroupService) DeleteAgentGroup(ctx context.Context, id uuid.UUID, deletedBy string) error {
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

func (s *AgentGroupService) ListAgentsByAgentGroup(
	ctx context.Context,
	agentGroup *agentgroup.AgentGroup,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	agentSelector := agentGroup.Selector

	listResp, err := s.agentIndexer.ListAgentsByAttributes(ctx, agentSelector.IdentifyingAttributes, agentSelector.NonIdentifyingAttributes, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents by agent group: %w", err)
	}

	return listResp, nil
}
