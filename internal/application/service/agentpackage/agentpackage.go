// Package agentpackage provides the AgentPackageService for managing agent packages.
package agentpackage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/internal/application/helper"
	"github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/internal/security"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ port.AgentPackageManageUsecase = (*Service)(nil)

// Service is a service for managing agent packages.
type Service struct {
	agentpackageUsecase domainport.AgentPackageUsecase
	mapper              *helper.Mapper
	clock               clock.Clock
	logger              *slog.Logger
}

// NewAgentPackageService creates a new AgentPackageService.
func NewAgentPackageService(
	agentpackageUsecase domainport.AgentPackageUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		agentpackageUsecase: agentpackageUsecase,
		mapper:              helper.NewMapper(),
		clock:               clock.NewRealClock(),
		logger:              logger,
	}
}

// GetAgentPackage implements [port.AgentPackageManageUsecase].
func (a *Service) GetAgentPackage(
	ctx context.Context,
	name string,
) (*v1.AgentPackage, error) {
	agentPackage, err := a.agentpackageUsecase.GetAgentPackage(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get agent package: %w", err)
	}

	return a.mapper.MapAgentPackageToAPI(agentPackage), nil
}

// ListAgentPackages implements [port.AgentPackageManageUsecase].
func (a *Service) ListAgentPackages(
	ctx context.Context,
	options *model.ListOptions,
) (*v1.ListResponse[v1.AgentPackage], error) {
	agentPackages, err := a.agentpackageUsecase.ListAgentPackages(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("list agent packages: %w", err)
	}

	return &v1.ListResponse[v1.AgentPackage]{
		Kind:       v1.AgentPackageKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.ListMeta{
			Continue:           agentPackages.Continue,
			RemainingItemCount: agentPackages.RemainingItemCount,
		},
		Items: lo.Map(agentPackages.Items, func(item *model.AgentPackage, _ int) v1.AgentPackage {
			return *a.mapper.MapAgentPackageToAPI(item)
		}),
	}, nil
}

// CreateAgentPackage implements [port.AgentPackageManageUsecase].
func (a *Service) CreateAgentPackage(
	ctx context.Context,
	apiModel *v1.AgentPackage,
) (*v1.AgentPackage, error) {
	domainModel := a.mapper.MapAPIToAgentPackage(apiModel)
	// Set the created condition with createdBy information
	now := a.clock.Now()

	createdBy, err := security.GetUser(ctx)
	if err != nil {
		a.logger.Warn("failed to get user from context", slog.String("error", err.Error()))

		createdBy = security.NewAnonymousUser()
	}

	domainModel.Status = model.AgentPackageStatus{
		Conditions: []model.Condition{
			{
				Type:               model.ConditionTypeCreated,
				LastTransitionTime: now,
				Status:             model.ConditionStatusTrue,
				Reason:             createdBy.String(),
				Message:            "Agent group created",
			},
		},
	}

	created, err := a.agentpackageUsecase.SaveAgentPackage(ctx, domainModel)
	if err != nil {
		return nil, fmt.Errorf("create agent package: %w", err)
	}

	return a.mapper.MapAgentPackageToAPI(created), nil
}

// UpdateAgentPackage implements [port.AgentPackageManageUsecase].
func (a *Service) UpdateAgentPackage(
	ctx context.Context,
	name string,
	agentPackage *v1.AgentPackage,
) (*v1.AgentPackage, error) {
	existingDomainModel, err := a.agentpackageUsecase.GetAgentPackage(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get existing agent package: %w", err)
	}

	domainModel := a.mapper.MapAPIToAgentPackage(agentPackage)
	domainModel.Status = existingDomainModel.Status

	updated, err := a.agentpackageUsecase.SaveAgentPackage(ctx, domainModel)
	if err != nil {
		return nil, fmt.Errorf("update agent package: %w", err)
	}

	return a.mapper.MapAgentPackageToAPI(updated), nil
}

// DeleteAgentPackage implements [port.AgentPackageManageUsecase].
func (a *Service) DeleteAgentPackage(
	ctx context.Context,
	name string,
) error {
	deletedBy, err := security.GetUser(ctx)
	if err != nil {
		a.logger.Warn("failed to get user from context", slog.String("error", err.Error()))

		deletedBy = security.NewAnonymousUser()
	}

	err = a.agentpackageUsecase.DeleteAgentPackage(ctx, name, a.clock.Now(), deletedBy.String())
	if err != nil {
		return fmt.Errorf("delete agent package: %w", err)
	}

	return nil
}
