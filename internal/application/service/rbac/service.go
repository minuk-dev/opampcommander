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
	rbacUsecase                userport.RBACUsecase
	roleUsecase                userport.RoleUsecase
	permissionUsecase          userport.PermissionUsecase
	roleBindingPersistencePort userport.RoleBindingPersistencePort
	mapper                     *helper.Mapper
	logger                     *slog.Logger
}

// New creates a new instance of the Service struct.
func New(
	rbacUsecase userport.RBACUsecase,
	roleUsecase userport.RoleUsecase,
	permissionUsecase userport.PermissionUsecase,
	roleBindingPersistencePort userport.RoleBindingPersistencePort,
	logger *slog.Logger,
) *Service {
	return &Service{
		rbacUsecase:                rbacUsecase,
		roleUsecase:                roleUsecase,
		permissionUsecase:          permissionUsecase,
		roleBindingPersistencePort: roleBindingPersistencePort,
		mapper:                     helper.NewMapper(),
		logger:                     logger,
	}
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
//
//nolint:funlen // Nil-UID validation adds necessary safety checks that push length over the limit.
func (s *Service) GetUserRoles(
	ctx context.Context,
	userID uuid.UUID,
) (*v1.ListResponse[v1.Role], error) {
	// Find role bindings for this user
	allBindings, err := s.roleBindingPersistencePort.ListRoleBindings(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list role bindings: %w", err)
	}

	roleMap := make(map[uuid.UUID]*usermodel.Role)

	for _, binding := range allBindings.Items {
		if binding.Spec.Subject.UID == uuid.Nil {
			s.logger.WarnContext(ctx, "skipping role binding with nil subject UID",
				slog.String("namespace", binding.Metadata.Namespace),
				slog.String("name", binding.Metadata.Name),
			)

			continue
		}

		if binding.Spec.Subject.UID != userID {
			continue
		}

		if binding.Spec.RoleRef.UID == uuid.Nil {
			s.logger.WarnContext(ctx, "skipping role binding with nil role ref UID",
				slog.String("namespace", binding.Metadata.Namespace),
				slog.String("name", binding.Metadata.Name),
			)

			continue
		}

		if _, exists := roleMap[binding.Spec.RoleRef.UID]; exists {
			continue
		}

		role, roleErr := s.roleUsecase.GetRole(ctx, binding.Spec.RoleRef.UID)
		if roleErr != nil {
			s.logger.WarnContext(ctx, "failed to get role for role binding",
				slog.String("roleUID", binding.Spec.RoleRef.UID.String()),
				slog.Any("error", roleErr),
			)

			continue
		}

		roleMap[binding.Spec.RoleRef.UID] = role
	}

	roles := make([]*usermodel.Role, 0, len(roleMap))
	for _, role := range roleMap {
		roles = append(roles, role)
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
