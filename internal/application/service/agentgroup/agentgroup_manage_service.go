// Package agentgroup provides the AgentGroupManageService for managing agent groups.
package agentgroup

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	v1agentgroup "github.com/minuk-dev/opampcommander/api/v1/agentgroup"
	"github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainagentgroup "github.com/minuk-dev/opampcommander/internal/domain/model/agentgroup"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

// ManageService implements port.AgentGroupManageUsecase. You can inject repository or other dependencies as needed.
type ManageService struct {
	persistence domainport.AgentGroupPersistencePort
	clock       clock.Clock
}

// NewManageService returns a new ManageService.
func NewManageService(persistence domainport.AgentGroupPersistencePort) *ManageService {
	return &ManageService{
		persistence: persistence,
		clock:       clock.NewRealClock(),
	}
}

// NewManageServiceWithClock returns a new ManageService with custom clock (for testing).
func NewManageServiceWithClock(persistence domainport.AgentGroupPersistencePort, clk clock.Clock) *ManageService {
	return &ManageService{
		persistence: persistence,
		clock:       clk,
	}
}

// GetAgentGroup returns an agent group by its UUID.
func (s *ManageService) GetAgentGroup(
	ctx context.Context,
	id uuid.UUID,
) (*v1agentgroup.AgentGroup, error) {
	agentGroup, err := s.persistence.GetAgentGroup(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get agent group: %w", err)
	}

	return toAPIModelAgentGroup(agentGroup), nil
}

// ListAgentGroups returns a paginated list of agent groups.
// ListAgentGroups returns a paginated list of agent groups.
func (s *ManageService) ListAgentGroups(
	ctx context.Context,
	options *model.ListOptions,
) (
	*model.ListResponse[*v1agentgroup.AgentGroup],
	error,
) {
	domainResp, err := s.persistence.ListAgentGroups(ctx, options)
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
// CreateAgentGroup creates a new agent group.
func (s *ManageService) CreateAgentGroup(
	ctx context.Context,
	createCommand *port.CreateAgentGroupCommand,
) (*v1agentgroup.AgentGroup, error) {
	domainAgentGroup := s.toDomainModelAgentGroupForCreate(createCommand)

	var err = s.persistence.PutAgentGroup(ctx, domainAgentGroup)
	if err != nil {
		return nil, fmt.Errorf("create agent group: %w", err)
	}

	return toAPIModelAgentGroup(domainAgentGroup), nil
}

// UpdateAgentGroup updates an existing agent group.
// UpdateAgentGroup updates an existing agent group.
func (s *ManageService) UpdateAgentGroup(
	ctx context.Context,
	uid uuid.UUID,
	apiAgentGroup *v1agentgroup.AgentGroup,
) (*v1agentgroup.AgentGroup, error) {
	domainAgentGroup := toDomainModelAgentGroupFromAPI(apiAgentGroup)
	domainAgentGroup.UID = uid

	var err = s.persistence.PutAgentGroup(ctx, domainAgentGroup)
	if err != nil {
		return nil, fmt.Errorf("update agent group: %w", err)
	}

	return toAPIModelAgentGroup(domainAgentGroup), nil
}

// DeleteAgentGroup marks an agent group as deleted.
// DeleteAgentGroup marks an agent group as deleted.
func (s *ManageService) DeleteAgentGroup(
	ctx context.Context,
	id uuid.UUID,
	deletedBy string,
) error {
	agentGroup, err := s.persistence.GetAgentGroup(ctx, id)
	if err != nil {
		return fmt.Errorf("get agent group for delete: %w", err)
	}

	if agentGroup.IsDeleted() {
		return nil
	}

	agentGroup.MarkDeleted(s.clock.Now(), deletedBy)

	err = s.persistence.PutAgentGroup(ctx, agentGroup)
	if err != nil {
		return fmt.Errorf("put agent group for delete: %w", err)
	}

	return nil
}

// Model 변환 함수 (domain <-> api).
func toAPIModelAgentGroup(domain *domainagentgroup.AgentGroup) *v1agentgroup.AgentGroup {
	if domain == nil {
		return nil
	}

	var deletedAt, deletedBy *string

	if domain.DeletedAt != nil {
		dt := domain.DeletedAt.Format("2006-01-02T15:04:05Z07:00")
		deletedAt = &dt
	}

	if domain.DeletedBy != nil {
		deletedBy = domain.DeletedBy
	}

	return &v1agentgroup.AgentGroup{
		UID:        domain.UID,
		Name:       domain.Name,
		Attributes: v1agentgroup.Attributes(domain.Attributes),
		Selector: v1agentgroup.AgentSelector{
			IdentifyingAttributes:    domain.Selector.IdentifyingAttributes,
			NonIdentifyingAttributes: domain.Selector.NonIdentifyingAttributes,
		},
		CreatedAt: domain.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		CreatedBy: domain.CreatedBy,
		DeletedAt: deletedAt,
		DeletedBy: deletedBy,
	}
}

func (s *ManageService) toDomainModelAgentGroupForCreate(
	cmd *port.CreateAgentGroupCommand,
) *domainagentgroup.AgentGroup {
	// NOTE: Replace "system" with actual user info in production.
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
		CreatedBy: "system",
		DeletedAt: nil,
		DeletedBy: nil,
	}
}

func toDomainModelAgentGroupFromAPI(api *v1agentgroup.AgentGroup) *domainagentgroup.AgentGroup {
	if api == nil {
		return nil
	}

	var deletedAt *time.Time

	if api.DeletedAt != nil {
		t, _ := time.Parse("2006-01-02T15:04:05Z07:00", *api.DeletedAt)
		deletedAt = &t
	}

	createdAt, _ := time.Parse("2006-01-02T15:04:05Z07:00", api.CreatedAt)

	return &domainagentgroup.AgentGroup{
		Version:    domainagentgroup.Version1,
		UID:        api.UID,
		Name:       api.Name,
		Attributes: domainagentgroup.Attributes(api.Attributes),
		Selector: domainagentgroup.AgentSelector{
			IdentifyingAttributes:    api.Selector.IdentifyingAttributes,
			NonIdentifyingAttributes: api.Selector.NonIdentifyingAttributes,
		},
		CreatedAt: createdAt,
		CreatedBy: api.CreatedBy,
		DeletedAt: deletedAt,
		DeletedBy: api.DeletedBy,
	}
}

var _ port.AgentGroupManageUsecase = (*ManageService)(nil)
