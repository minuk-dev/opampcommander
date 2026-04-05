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
	namespace, resource, action string,
) (bool, error) {
	allowed, err := s.rbacEnforcerPort.CheckPermission(ctx, userID.String(), namespace, resource, action)
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
	// Phase 1: Build all policies in memory before modifying the enforcer
	policies, groupings, err := s.collectPolicies(ctx)
	if err != nil {
		return err
	}

	// Phase 2: All data loaded successfully — now clear and apply atomically
	s.rbacEnforcerPort.ClearPolicy(ctx)

	// Rebuild role links after clearing to reset the role inheritance graph.
	err = s.rbacEnforcerPort.BuildRoleLinks(ctx)
	if err != nil {
		return fmt.Errorf("failed to rebuild role links: %w", err)
	}

	for _, p := range policies {
		_, err = s.rbacEnforcerPort.AddNamedPolicy(ctx, "p", p.roleID, p.dom, p.resource, p.action)
		if err != nil {
			return fmt.Errorf("failed to add named policy: %w", err)
		}
	}

	for _, g := range groupings {
		_, err = s.rbacEnforcerPort.AddGroupingPolicy(ctx, g.userID, g.roleID, g.namespace)
		if err != nil {
			return fmt.Errorf("failed to add grouping policy: %w", err)
		}
	}

	err = s.rbacEnforcerPort.SavePolicy(ctx)
	if err != nil {
		return fmt.Errorf("failed to save policies: %w", err)
	}

	return nil
}

type namedPolicy struct {
	roleID, dom, resource, action string
}

type groupingPolicy struct {
	userID, roleID, namespace string
}

func (s *RBACService) collectPolicies(ctx context.Context) ([]namedPolicy, []groupingPolicy, error) {
	rolesResp, err := s.rolePersistencePort.ListRoles(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list roles from persistence: %w", err)
	}

	var policies []namedPolicy

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

			policies = append(policies, namedPolicy{
				roleID:   role.Metadata.UID.String(),
				dom:      usermodel.WildcardAll,
				resource: permission.Spec.Resource,
				action:   permission.Spec.Action,
			})
		}
	}

	userRolesResp, err := s.userRolePersistencePort.ListUserRoles(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list user roles from persistence: %w", err)
	}

	groupings := make([]groupingPolicy, 0, len(userRolesResp.Items))
	for _, userRole := range userRolesResp.Items {
		groupings = append(groupings, groupingPolicy{
			userID:    userRole.Spec.UserID.String(),
			roleID:    userRole.Spec.RoleID.String(),
			namespace: userRole.Spec.Namespace,
		})
	}

	return policies, groupings, nil
}
