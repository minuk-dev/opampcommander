// Package agentgroup provides the AgentGroupManageService for managing agent groups.
package agentgroup

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	v1agentgroup "github.com/minuk-dev/opampcommander/api/v1/agentgroup"
	"github.com/minuk-dev/opampcommander/internal/application/helper"
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
	mapper            *helper.Mapper
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
		mapper:            helper.NewMapper(),
		clock:             clock.NewRealClock(),
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

	return s.mapper.MapAPIToAgentGroup(agentGroup), nil
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
			return *s.mapper.MapAPIToAgentGroup(agentGroup)
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
) (*v1.ListResponse[v1.Agent], error) {
	agentGroup, err := s.agentgroupUsecase.GetAgentGroup(ctx, agentGroupName)
	if err != nil {
		return nil, fmt.Errorf("get agent group: %w", err)
	}

	domainResp, err := s.agentUsecase.ListAgentsBySelector(ctx, agentGroup.Metadata.Selector, options)
	if err != nil {
		return nil, fmt.Errorf("list agents by agent group: %w", err)
	}

	return &v1.ListResponse[v1.Agent]{
		Kind:       v1.AgentKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.ListMeta{
			Continue:           domainResp.Continue,
			RemainingItemCount: domainResp.RemainingItemCount,
		},
		Items: lo.Map(domainResp.Items, func(agent *model.Agent, _ int) v1.Agent {
			return *s.mapper.MapAgentToAPI(agent)
		}),
	}, nil
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

	return s.toAPIModelAgentGroup(agentGroup), nil
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

	return s.toAPIModelAgentGroup(agentGroup), nil
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
