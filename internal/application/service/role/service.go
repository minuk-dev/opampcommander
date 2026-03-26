// Package role provides application services for role management.
package role

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/internal/application/helper"
	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ applicationport.RoleManageUsecase = (*Service)(nil)

// Service is a struct that implements the RoleManageUsecase interface.
type Service struct {
	roleUsecase domainport.RoleUsecase
	mapper      *helper.Mapper
	logger      *slog.Logger
}

// New creates a new instance of the Service struct.
func New(roleUsecase domainport.RoleUsecase, logger *slog.Logger) *Service {
	return &Service{
		roleUsecase: roleUsecase,
		mapper:      helper.NewMapper(),
		logger:      logger,
	}
}

// GetRole implements [applicationport.RoleManageUsecase].
func (s *Service) GetRole(ctx context.Context, uid uuid.UUID) (*v1.Role, error) {
	role, err := s.roleUsecase.GetRole(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return s.mapper.MapRoleToAPI(role), nil
}

// ListRoles implements [applicationport.RoleManageUsecase].
func (s *Service) ListRoles(
	ctx context.Context,
	options *model.ListOptions,
) (*v1.ListResponse[v1.Role], error) {
	response, err := s.roleUsecase.ListRoles(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	return &v1.ListResponse[v1.Role]{
		Kind:       v1.RoleKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.ListMeta{
			Continue:           response.Continue,
			RemainingItemCount: response.RemainingItemCount,
		},
		Items: lo.Map(response.Items, func(role *model.Role, _ int) v1.Role {
			return *s.mapper.MapRoleToAPI(role)
		}),
	}, nil
}

// CreateRole implements [applicationport.RoleManageUsecase].
func (s *Service) CreateRole(ctx context.Context, apiRole *v1.Role) (*v1.Role, error) {
	domainRole := model.NewRole(apiRole.Spec.DisplayName, apiRole.Spec.IsBuiltIn)
	domainRole.Spec.Description = apiRole.Spec.Description
	domainRole.Spec.Permissions = apiRole.Spec.Permissions

	err := s.roleUsecase.SaveRole(ctx, domainRole)
	if err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	return s.mapper.MapRoleToAPI(domainRole), nil
}

// UpdateRole implements [applicationport.RoleManageUsecase].
func (s *Service) UpdateRole(ctx context.Context, uid uuid.UUID, apiRole *v1.Role) (*v1.Role, error) {
	existing, err := s.roleUsecase.GetRole(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	existing.Spec.DisplayName = apiRole.Spec.DisplayName
	existing.Spec.Description = apiRole.Spec.Description
	existing.Spec.Permissions = apiRole.Spec.Permissions
	existing.Metadata.UpdatedAt = time.Now()

	err = s.roleUsecase.SaveRole(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	return s.mapper.MapRoleToAPI(existing), nil
}

// DeleteRole implements [applicationport.RoleManageUsecase].
func (s *Service) DeleteRole(ctx context.Context, uid uuid.UUID) error {
	err := s.roleUsecase.DeleteRole(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	return nil
}
