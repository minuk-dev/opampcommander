// Package agentgroup provides the AgentGroupManageService for managing agent groups.
package agentgroup

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/samber/lo"
	k8sclock "k8s.io/utils/clock"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	v1agent "github.com/minuk-dev/opampcommander/api/v1/agent"
	v1agentgroup "github.com/minuk-dev/opampcommander/api/v1/agentgroup"
	"github.com/minuk-dev/opampcommander/internal/application/mapper"
	"github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainagentgroup "github.com/minuk-dev/opampcommander/internal/domain/model/agentgroup"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/internal/security"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ port.AgentGroupManageUsecase = (*ManageService)(nil)

// ManageService implements port.AgentGroupManageUsecase. You can inject repository or other dependencies as needed.
type ManageService struct {
	agentgroupUsecase domainport.AgentGroupUsecase
	agentUsecase      domainport.AgentUsecase
	agentMapper       mapper.Mapper
	clock             clock.Clock
	logger            *slog.Logger
}

// NewManageService returns a new ManageService.
func NewManageService(
	agentgroupUsecase domainport.AgentGroupUsecase,
	logger *slog.Logger,
) *ManageService {
	return &ManageService{
		agentgroupUsecase: agentgroupUsecase,
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

	return toAPIModelAgentGroup(agentGroup), nil
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
		lo.Map(domainResp.Items, func(agentGroup *domainagentgroup.AgentGroup, _ int) v1agentgroup.AgentGroup {
			return *toAPIModelAgentGroup(agentGroup)
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

	domainResp, err := s.agentUsecase.ListAgentsBySelector(ctx, agentGroup.Selector, options)
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
		return nil, fmt.Errorf("get user from context: %w", err)
	}

	domainAgentGroup := s.toDomainModelAgentGroupForCreate(createCommand, requestedBy)

	err = s.agentgroupUsecase.SaveAgentGroup(ctx, createCommand.Name, domainAgentGroup)
	if err != nil {
		return nil, fmt.Errorf("create agent group: %w", err)
	}

	return toAPIModelAgentGroup(domainAgentGroup), nil
}

// UpdateAgentGroup updates an existing agent group.
func (s *ManageService) UpdateAgentGroup(
	ctx context.Context,
	name string,
	apiAgentGroup *v1agentgroup.AgentGroup,
) (*v1agentgroup.AgentGroup, error) {
	domainAgentGroup := toDomainModelAgentGroupFromAPI(apiAgentGroup)

	err := s.agentgroupUsecase.SaveAgentGroup(ctx, name, domainAgentGroup)
	if err != nil {
		return nil, fmt.Errorf("update agent group: %w", err)
	}

	return toAPIModelAgentGroup(domainAgentGroup), nil
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

func toAPIModelAgentGroup(domain *domainagentgroup.AgentGroup) *v1agentgroup.AgentGroup {
	if domain == nil {
		return nil
	}

	return &v1agentgroup.AgentGroup{
		UID:        domain.UID,
		Name:       domain.Name,
		Attributes: v1agentgroup.Attributes(domain.Attributes),
		Selector: v1agentgroup.AgentSelector{
			IdentifyingAttributes:    domain.Selector.IdentifyingAttributes,
			NonIdentifyingAttributes: domain.Selector.NonIdentifyingAttributes,
		},
		CreatedAt: domain.CreatedAt,
		CreatedBy: domain.CreatedBy,
		DeletedAt: domain.DeletedAt,
		DeletedBy: domain.DeletedBy,
	}
}

func (s *ManageService) toDomainModelAgentGroupForCreate(
	cmd *port.CreateAgentGroupCommand,
	requestedBy *security.User,
) *domainagentgroup.AgentGroup {
	return &domainagentgroup.AgentGroup{
		UID:        uuid.New(),
		Name:       cmd.Name,
		Attributes: domainagentgroup.Attributes(cmd.Attributes),
		Selector: model.AgentSelector{
			IdentifyingAttributes:    cmd.Selector.IdentifyingAttributes,
			NonIdentifyingAttributes: cmd.Selector.NonIdentifyingAttributes,
		},
		CreatedAt: s.clock.Now(),
		CreatedBy: requestedBy.String(),
		DeletedAt: nil,
		DeletedBy: nil,
	}
}

func toDomainModelAgentGroupFromAPI(api *v1agentgroup.AgentGroup) *domainagentgroup.AgentGroup {
	if api == nil {
		return nil
	}

	return &domainagentgroup.AgentGroup{
		UID:        api.UID,
		Name:       api.Name,
		Attributes: domainagentgroup.Attributes(api.Attributes),
		Selector: model.AgentSelector{
			IdentifyingAttributes:    api.Selector.IdentifyingAttributes,
			NonIdentifyingAttributes: api.Selector.NonIdentifyingAttributes,
		},
		CreatedAt: api.CreatedAt,
		CreatedBy: api.CreatedBy,
		DeletedAt: api.DeletedAt,
		DeletedBy: api.DeletedBy,
	}
}
