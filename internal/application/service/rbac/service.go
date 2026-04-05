// Package rbac provides application services for RBAC authorization.
package rbac

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/internal/application/helper"
	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
)

var _ applicationport.RBACManageUsecase = (*Service)(nil)

// Service is a struct that implements the RBACManageUsecase interface.
type Service struct {
	userRoleUsecase   userport.UserRoleUsecase
	rbacUsecase       userport.RBACUsecase
	roleUsecase       userport.RoleUsecase
	permissionUsecase userport.PermissionUsecase
	userUsecase       userport.UserUsecase
	mapper            *helper.Mapper
	logger            *slog.Logger
}

// New creates a new instance of the Service struct.
func New(
	userRoleUsecase userport.UserRoleUsecase,
	rbacUsecase userport.RBACUsecase,
	roleUsecase userport.RoleUsecase,
	permissionUsecase userport.PermissionUsecase,
	userUsecase userport.UserUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		userRoleUsecase:   userRoleUsecase,
		rbacUsecase:       rbacUsecase,
		roleUsecase:       roleUsecase,
		permissionUsecase: permissionUsecase,
		userUsecase:       userUsecase,
		mapper:            helper.NewMapper(),
		logger:            logger,
	}
}

// AssignRole implements [applicationport.RBACManageUsecase].
func (s *Service) AssignRole(ctx context.Context, req *v1.AssignRoleRequest) error {
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return fmt.Errorf("failed to parse user ID: %w", err)
	}

	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		return fmt.Errorf("failed to parse role ID: %w", err)
	}

	// AssignedBy is now an email from the security context; resolve to user UID
	assigner, err := s.userUsecase.GetUserByEmail(ctx, req.AssignedBy)
	if err != nil {
		return fmt.Errorf("failed to resolve assigner identity: %w", err)
	}

	namespace := req.Namespace
	if namespace == "" {
		namespace = "*"
	}

	err = s.userRoleUsecase.AssignRole(ctx, userID, roleID, assigner.Metadata.UID, namespace)
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	return nil
}

// UnassignRole implements [applicationport.RBACManageUsecase].
func (s *Service) UnassignRole(ctx context.Context, userID, roleID uuid.UUID, namespace string) error {
	if namespace == "" {
		namespace = "*"
	}

	err := s.userRoleUsecase.UnassignRole(ctx, userID, roleID, namespace)
	if err != nil {
		return fmt.Errorf("failed to unassign role: %w", err)
	}

	return nil
}

// CheckPermission implements [applicationport.RBACManageUsecase].
func (s *Service) CheckPermission(
	ctx context.Context,
	req *v1.CheckPermissionRequest,
) (*v1.CheckPermissionResponse, error) {
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user ID: %w", err)
	}

	allowed, err := s.rbacUsecase.CheckPermission(ctx, userID, req.Namespace, req.Resource, req.Action)
	if err != nil {
		return nil, fmt.Errorf("failed to check permission: %w", err)
	}

	return &v1.CheckPermissionResponse{
		Allowed: allowed,
	}, nil
}

// GetUserRoles implements [applicationport.RBACManageUsecase].
func (s *Service) GetUserRoles(
	ctx context.Context,
	userID uuid.UUID,
) (*v1.ListResponse[v1.Role], error) {
	roles, err := s.userRoleUsecase.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	return &v1.ListResponse[v1.Role]{
		Kind:       v1.RoleKind,
		APIVersion: v1.APIVersion,
		Metadata:   v1.ListMeta{Continue: "", RemainingItemCount: 0},
		Items: lo.Map(roles, func(role *usermodel.Role, _ int) v1.Role {
			return *s.mapper.MapRoleToAPI(role)
		}),
	}, nil
}

// GetUserPermissions implements [applicationport.RBACManageUsecase].
func (s *Service) GetUserPermissions(
	ctx context.Context,
	userID uuid.UUID,
) (*v1.ListResponse[v1.Permission], error) {
	permissions, err := s.rbacUsecase.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	return &v1.ListResponse[v1.Permission]{
		Kind:       v1.PermissionKind,
		APIVersion: v1.APIVersion,
		Metadata:   v1.ListMeta{Continue: "", RemainingItemCount: 0},
		Items: lo.Map(permissions, func(permission *usermodel.Permission, _ int) v1.Permission {
			return *s.mapper.MapPermissionToAPI(permission)
		}),
	}, nil
}

// SyncPolicies implements [applicationport.RBACManageUsecase].
func (s *Service) SyncPolicies(ctx context.Context) error {
	err := s.rbacUsecase.SyncPolicies(ctx)
	if err != nil {
		return fmt.Errorf("failed to sync policies: %w", err)
	}

	return nil
}
