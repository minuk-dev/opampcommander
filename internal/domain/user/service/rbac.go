package userservice

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
)

var _ userport.RBACUsecase = (*RBACService)(nil)

// RBACService implements the RBACUsecase interface.
type RBACService struct {
	rbacEnforcerPort          userport.RBACEnforcerPort
	userRolePersistencePort   userport.UserRolePersistencePort
	rolePersistencePort       userport.RolePersistencePort
	permissionPersistencePort userport.PermissionPersistencePort
	logger                    *slog.Logger
}

// NewRBACService creates a new instance of RBACService.
func NewRBACService(
	rbacEnforcerPort userport.RBACEnforcerPort,
	userRolePersistencePort userport.UserRolePersistencePort,
	rolePersistencePort userport.RolePersistencePort,
	permissionPersistencePort userport.PermissionPersistencePort,
	logger *slog.Logger,
) *RBACService {
	return &RBACService{
		rbacEnforcerPort:          rbacEnforcerPort,
		userRolePersistencePort:   userRolePersistencePort,
		rolePersistencePort:       rolePersistencePort,
		permissionPersistencePort: permissionPersistencePort,
		logger:                    logger,
	}
}

// CheckPermission implements [userport.RBACUsecase].
func (s *RBACService) CheckPermission(
	ctx context.Context,
	userID uuid.UUID,
	resource, action string,
) (bool, error) {
	allowed, err := s.rbacEnforcerPort.CheckPermission(ctx, userID, resource, action)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}

	return allowed, nil
}

// GetUserPermissions implements [userport.RBACUsecase].
func (s *RBACService) GetUserPermissions(
	ctx context.Context,
	userID uuid.UUID,
) ([]*usermodel.Permission, error) {
	roles, err := s.userRolePersistencePort.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles from persistence: %w", err)
	}

	permissionMap := make(map[string]*usermodel.Permission)

	for _, role := range roles {
		for _, permissionID := range role.Spec.Permissions {
			if _, exists := permissionMap[permissionID]; exists {
				continue
			}

			permission, err := s.permissionPersistencePort.GetPermissionByName(ctx, permissionID)
			if err != nil {
				s.logger.WarnContext(ctx, "failed to get permission by name",
					slog.String("permissionID", permissionID),
					slog.Any("error", err),
				)

				continue
			}

			permissionMap[permissionID] = permission
		}
	}

	permissions := make([]*usermodel.Permission, 0, len(permissionMap))
	for _, permission := range permissionMap {
		permissions = append(permissions, permission)
	}

	return permissions, nil
}

// GetEffectivePermissions implements [userport.RBACUsecase].
func (s *RBACService) GetEffectivePermissions(
	ctx context.Context,
	userID uuid.UUID,
) ([]*usermodel.Permission, error) {
	return s.GetUserPermissions(ctx, userID)
}

// SyncPolicies implements [userport.RBACUsecase].
func (s *RBACService) SyncPolicies(ctx context.Context) error {
	// Clear existing policies to prevent duplicate accumulation
	s.rbacEnforcerPort.ClearPolicy(ctx)

	// Load all roles with their permissions
	rolesResp, err := s.rolePersistencePort.ListRoles(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to list roles from persistence: %w", err)
	}

	// Add role-permission policies (named policies with ptype "p")
	for _, role := range rolesResp.Items {
		for _, permissionID := range role.Spec.Permissions {
			permission, err := s.permissionPersistencePort.GetPermissionByName(ctx, permissionID)
			if err != nil {
				s.logger.WarnContext(ctx, "failed to get permission during sync",
					slog.String("permissionID", permissionID),
					slog.Any("error", err),
				)

				continue
			}

			_, err = s.rbacEnforcerPort.AddNamedPolicy(ctx, "p",
				role.Metadata.UID.String(), permission.Spec.Resource, permission.Spec.Action)
			if err != nil {
				return fmt.Errorf("failed to add named policy: %w", err)
			}
		}
	}

	// Load all user-role assignments
	userRolesResp, err := s.userRolePersistencePort.ListUserRoles(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to list user roles from persistence: %w", err)
	}

	// Add user-role grouping policies
	for _, userRole := range userRolesResp.Items {
		_, err = s.rbacEnforcerPort.AddGroupingPolicy(ctx,
			userRole.Spec.UserID.String(), userRole.Spec.RoleID.String())
		if err != nil {
			return fmt.Errorf("failed to add grouping policy: %w", err)
		}
	}

	// Save policies to persistence
	err = s.rbacEnforcerPort.SavePolicy(ctx)
	if err != nil {
		return fmt.Errorf("failed to save policies: %w", err)
	}

	return nil
}
