// Package agentgroup provides the AgentGroupManageService for managing agent groups.
package agentgroup

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	k8sclock "k8s.io/utils/clock"

	v1agentgroup "github.com/minuk-dev/opampcommander/api/v1/agentgroup"
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
	clock             clock.Clock
	logger            *slog.Logger
}

// NewManageService returns a new ManageService
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
) (
	*model.ListResponse[*v1agentgroup.AgentGroup],
	error,
) {
	domainResp, err := s.agentgroupUsecase.ListAgentGroups(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("list agent groups: %w", err)
	}

	apiItems := make([]*v1agentgroup.AgentGroup, 0, len(domainResp.Items))
	for _, agentGroup := range domainResp.Items {
		apiItems = append(apiItems, toAPIModelAgentGroup(agentGroup))
	}

	return &model.ListResponse[*v1agentgroup.AgentGroup]{
		RemainingItemCount: domainResp.RemainingItemCount,
		Continue:           domainResp.Continue,
		Items:              apiItems,
	}, nil
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
		Version:    domainagentgroup.Version1,
		UID:        uuid.New(),
		Name:       cmd.Name,
		Attributes: domainagentgroup.Attributes(cmd.Attributes),
		Selector: domainagentgroup.AgentSelector{
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
		Version:    domainagentgroup.Version1,
		UID:        api.UID,
		Name:       api.Name,
		Attributes: domainagentgroup.Attributes(api.Attributes),
		Selector: domainagentgroup.AgentSelector{
			IdentifyingAttributes:    api.Selector.IdentifyingAttributes,
			NonIdentifyingAttributes: api.Selector.NonIdentifyingAttributes,
		},
		CreatedAt: api.CreatedAt,
		CreatedBy: api.CreatedBy,
		DeletedAt: api.DeletedAt,
		DeletedBy: api.DeletedBy,
	}
}
