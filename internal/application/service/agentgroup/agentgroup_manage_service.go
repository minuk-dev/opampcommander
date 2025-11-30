// Package agentgroup provides the AgentGroupManageService for managing agent groups.
package agentgroup

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/samber/lo"
	k8sclock "k8s.io/utils/clock"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	v1agent "github.com/minuk-dev/opampcommander/api/v1/agent"
	v1agentgroup "github.com/minuk-dev/opampcommander/api/v1/agentgroup"
	"github.com/minuk-dev/opampcommander/internal/application/mapper"
	"github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/internal/security"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ port.AgentGroupManageUsecase = (*ManageService)(nil)

// ManageService implements port.AgentGroupManageUsecase. You can inject repository or other dependencies as needed.
type ManageService struct {
	agentgroupUsecase domainport.AgentGroupUsecase
	agentUsecase      domainport.AgentUsecase
	agentMapper       *mapper.Mapper
	clock             clock.Clock
	logger            *slog.Logger
}

// NewManageService returns a new ManageService.
func NewManageService(
	agentgroupUsecase domainport.AgentGroupUsecase,
	agentUsecase domainport.AgentUsecase,
	logger *slog.Logger,
) *ManageService {
	return &ManageService{
		agentgroupUsecase: agentgroupUsecase,
		agentUsecase:      agentUsecase,
		agentMapper:       mapper.New(),
		clock:             k8sclock.RealClock{},
		logger:            logger,
	}
}

// GetAgentGroup returns an agent group by its UUID.
func (s *ManageService) GetAgentGroup(
	ctx context.Context,
	name string,
) (*v1agentgroup.AgentGroup, error) {
	agentGroup, err := s.agentgroupUsecase.GetAgentGroup(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get agent group: %w", err)
	}

	return s.toAPIModelAgentGroup(ctx, agentGroup), nil
}

// ListAgentGroups returns a paginated list of agent groups.
func (s *ManageService) ListAgentGroups(
	ctx context.Context,
	options *model.ListOptions,
) (*v1agentgroup.ListResponse, error) {
	domainResp, err := s.agentgroupUsecase.ListAgentGroups(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("list agent groups: %w", err)
	}

	return v1agentgroup.NewListResponse(
		lo.Map(domainResp.Items, func(agentGroup *model.AgentGroup, _ int) v1agentgroup.AgentGroup {
			return *s.toAPIModelAgentGroup(ctx, agentGroup)
		}),
		v1.ListMeta{
			Continue:           domainResp.Continue,
			RemainingItemCount: domainResp.RemainingItemCount,
		},
	), nil
}

// ListAgentsByAgentGroup implements port.AgentGroupManageUsecase.
func (s *ManageService) ListAgentsByAgentGroup(
	ctx context.Context,
	agentGroupName string,
	options *model.ListOptions,
) (*v1agent.ListResponse, error) {
	agentGroup, err := s.agentgroupUsecase.GetAgentGroup(ctx, agentGroupName)
	if err != nil {
		return nil, fmt.Errorf("get agent group: %w", err)
	}

	domainResp, err := s.agentUsecase.ListAgentsBySelector(ctx, agentGroup.Metadata.Selector, options)
	if err != nil {
		return nil, fmt.Errorf("list agents by agent group: %w", err)
	}

	return v1agent.NewListResponse(
		lo.Map(domainResp.Items, func(agent *model.Agent, _ int) v1agent.Agent {
			return *s.agentMapper.MapAgentToAPI(agent)
		}),
		v1.ListMeta{
			Continue:           domainResp.Continue,
			RemainingItemCount: domainResp.RemainingItemCount,
		},
	), nil
}

// CreateAgentGroup creates a new agent group.
func (s *ManageService) CreateAgentGroup(
	ctx context.Context,
	createCommand *port.CreateAgentGroupCommand,
) (*v1agentgroup.AgentGroup, error) {
	requestedBy, err := security.GetUser(ctx)
	if err != nil {
		s.logger.Warn("failed to get user from context", slog.String("error", err.Error()))

		requestedBy = security.NewAnonymousUser()
	}

	domainAgentGroup := s.toDomainModelAgentGroupForCreate(createCommand, requestedBy)

	agentGroup, err := s.agentgroupUsecase.SaveAgentGroup(ctx, createCommand.Name, domainAgentGroup)
	if err != nil {
		return nil, fmt.Errorf("create agent group: %w", err)
	}

	return s.toAPIModelAgentGroup(ctx, agentGroup), nil
}

// UpdateAgentGroup updates an existing agent group.
func (s *ManageService) UpdateAgentGroup(
	ctx context.Context,
	name string,
	apiAgentGroup *v1agentgroup.AgentGroup,
) (*v1agentgroup.AgentGroup, error) {
	domainAgentGroup := toDomainModelAgentGroupFromAPI(apiAgentGroup)

	agentGroup, err := s.agentgroupUsecase.SaveAgentGroup(ctx, name, domainAgentGroup)
	if err != nil {
		return nil, fmt.Errorf("update agent group: %w", err)
	}

	return s.toAPIModelAgentGroup(ctx, agentGroup), nil
}

// DeleteAgentGroup marks an agent group as deleted.
func (s *ManageService) DeleteAgentGroup(
	ctx context.Context,
	name string,
) error {
	deletedBy, err := security.GetUser(ctx)
	if err != nil {
		s.logger.Warn("failed to get user from context", slog.String("error", err.Error()))

		deletedBy = security.NewAnonymousUser()
	}

	deletedAt := s.clock.Now()

	err = s.agentgroupUsecase.DeleteAgentGroup(ctx, name, deletedAt, deletedBy.String())
	if err != nil {
		return fmt.Errorf("get agent group for delete: %w", err)
	}

	return nil
}

func (s *ManageService) toAPIModelAgentGroup(ctx context.Context, domain *model.AgentGroup) *v1agentgroup.AgentGroup {
	if domain == nil {
		return nil
	}

	var agentConfig *v1agentgroup.AgentConfig
	if domain.Spec.AgentConfig != nil {
		agentConfig = &v1agentgroup.AgentConfig{
			Value: domain.Spec.AgentConfig.Value,
		}
	}

	conditions := make([]v1agentgroup.Condition, len(domain.Status.Conditions))
	for i, condition := range domain.Status.Conditions {
		conditions[i] = v1agentgroup.Condition{
			Type:               v1agentgroup.ConditionType(condition.Type),
			LastTransitionTime: condition.LastTransitionTime,
			Status:             v1agentgroup.ConditionStatus(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
	}

	// Use statistics from domain model (calculated by persistence layer)
	return &v1agentgroup.AgentGroup{
		Metadata: v1agentgroup.Metadata{
			Name:       domain.Metadata.Name,
			Attributes: v1agentgroup.Attributes(domain.Metadata.Attributes),
			Priority:   domain.Metadata.Priority,
			Selector: v1agentgroup.AgentSelector{
				IdentifyingAttributes:    domain.Metadata.Selector.IdentifyingAttributes,
				NonIdentifyingAttributes: domain.Metadata.Selector.NonIdentifyingAttributes,
			},
		},
		Spec: v1agentgroup.Spec{
			AgentConfig: agentConfig,
		},
		Status: v1agentgroup.Status{
			NumAgents:             domain.Status.NumAgents,
			NumConnectedAgents:    domain.Status.NumConnectedAgents,
			NumHealthyAgents:      domain.Status.NumHealthyAgents,
			NumUnhealthyAgents:    domain.Status.NumUnhealthyAgents,
			NumNotConnectedAgents: domain.Status.NumNotConnectedAgents,
			Conditions:            conditions,
		},
	}
}

func (s *ManageService) toDomainModelAgentGroupForCreate(
	cmd *port.CreateAgentGroupCommand,
	requestedBy *security.User,
) *model.AgentGroup {
	var agentConfig *model.AgentConfig
	if cmd.AgentConfig != nil {
		agentConfig = &model.AgentConfig{
			Value: cmd.AgentConfig.Value,
		}
	}

	return &model.AgentGroup{
		Metadata: model.AgentGroupMetadata{
			Name:       cmd.Name,
			Attributes: model.Attributes(cmd.Attributes),
			Priority:   cmd.Priority,
			Selector: model.AgentSelector{
				IdentifyingAttributes:    cmd.Selector.IdentifyingAttributes,
				NonIdentifyingAttributes: cmd.Selector.NonIdentifyingAttributes,
			},
		},
		Spec: model.AgentGroupSpec{
			AgentConfig: agentConfig,
		},
		Status: model.AgentGroupStatus{
			Conditions: []model.Condition{
				{
					Type:               model.ConditionTypeCreated,
					LastTransitionTime: s.clock.Now(),
					Status:             model.ConditionStatusTrue,
					Reason:             requestedBy.String(),
					Message:            "Agent group created",
				},
			},
		},
	}
}

func toDomainModelAgentGroupFromAPI(api *v1agentgroup.AgentGroup) *model.AgentGroup {
	if api == nil {
		return nil
	}

	var agentConfig *model.AgentConfig
	if api.Spec.AgentConfig != nil {
		agentConfig = &model.AgentConfig{
			Value: api.Spec.AgentConfig.Value,
		}
	}

	conditions := make([]model.Condition, len(api.Status.Conditions))
	for i, condition := range api.Status.Conditions {
		conditions[i] = model.Condition{
			Type:               model.ConditionType(condition.Type),
			LastTransitionTime: condition.LastTransitionTime,
			Status:             model.ConditionStatus(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
	}

	return &model.AgentGroup{
		Metadata: model.AgentGroupMetadata{
			Name:       api.Metadata.Name,
			Priority:   api.Metadata.Priority,
			Attributes: model.Attributes(api.Metadata.Attributes),
			Selector: model.AgentSelector{
				IdentifyingAttributes:    api.Metadata.Selector.IdentifyingAttributes,
				NonIdentifyingAttributes: api.Metadata.Selector.NonIdentifyingAttributes,
			},
		},
		Spec: model.AgentGroupSpec{
			AgentConfig: agentConfig,
		},
		Status: model.AgentGroupStatus{
			NumAgents:             api.Status.NumAgents,
			NumConnectedAgents:    api.Status.NumConnectedAgents,
			NumHealthyAgents:      api.Status.NumHealthyAgents,
			NumUnhealthyAgents:    api.Status.NumUnhealthyAgents,
			NumNotConnectedAgents: api.Status.NumNotConnectedAgents,
			Conditions:            conditions,
		},
	}
}
