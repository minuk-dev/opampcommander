// Package role provides application services for role management.
package role

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"k8s.io/utils/clock"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/helper"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user"
	userport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/port"
)

var _ applicationport.RoleManageUsecase = (*Service)(nil)

// Service is a struct that implements the RoleManageUsecase interface.
type Service struct {
	roleUsecase userport.RoleUsecase
	mapper      *helper.Mapper
	logger      *slog.Logger
}

// New creates a new instance of the Service struct.
func New(roleUsecase userport.RoleUsecase, logger *slog.Logger) *Service {
	return &Service{
		roleUsecase: roleUsecase,
		mapper:      helper.NewMapper(clock.RealClock{}, 0),
		logger:      logger,
	}
}

// GetRole implements [applicationport.RoleManageUsecase].
func (s *Service) GetRole(ctx context.Context, uid uuid.UUID, options *applicationport.GetOptions) (*v1.Role, error) {
	role, err := s.roleUsecase.GetRole(ctx, uid, options.ToDomain())
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return s.mapper.MapRoleToAPI(role), nil
}

// ListRoles implements [applicationport.RoleManageUsecase].
func (s *Service) ListRoles(
	ctx context.Context,
	options *applicationport.ListOptions,
) (*v1.ListResponse[v1.Role], error) {
	response, err := s.roleUsecase.ListRoles(ctx, options.ToDomain())
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
		Items: lo.Map(response.Items, func(role *usermodel.Role, _ int) v1.Role {
			return *s.mapper.MapRoleToAPI(role)
		}),
	}, nil
}

// CreateRole implements [applicationport.RoleManageUsecase].
func (s *Service) CreateRole(ctx context.Context, apiRole *v1.Role) (*v1.Role, error) {
	domainRole := usermodel.NewRole(apiRole.Spec.DisplayName, false)
	domainRole.Spec.Description = apiRole.Spec.Description
	domainRole.Spec.Permissions = apiRole.Spec.Permissions

	created, err := s.roleUsecase.CreateRole(ctx, domainRole)
	if err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	return s.mapper.MapRoleToAPI(created), nil
}

// UpdateRole implements [applicationport.RoleManageUsecase].
func (s *Service) UpdateRole(ctx context.Context, uid uuid.UUID, apiRole *v1.Role) (*v1.Role, error) {
	domainRole := usermodel.NewRole(apiRole.Spec.DisplayName, false)
	domainRole.Spec.Description = apiRole.Spec.Description
	domainRole.Spec.Permissions = apiRole.Spec.Permissions

	updated, err := s.roleUsecase.UpdateRole(ctx, uid, domainRole)
	if err != nil {
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	return s.mapper.MapRoleToAPI(updated), nil
}

// DeleteRole implements [applicationport.RoleManageUsecase].
func (s *Service) DeleteRole(ctx context.Context, uid uuid.UUID) error {
	err := s.roleUsecase.DeleteRole(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	return nil
}
