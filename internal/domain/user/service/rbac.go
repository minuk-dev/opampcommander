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
	rbacEnforcerPort           userport.RBACEnforcerPort
	roleBindingPersistencePort userport.RoleBindingPersistencePort
	rolePersistencePort        userport.RolePersistencePort
	permissionPersistencePort  userport.PermissionPersistencePort
	userPersistencePort        userport.UserPersistencePort
	logger                     *slog.Logger
}

// NewRBACService creates a new instance of RBACService.
func NewRBACService(
	rbacEnforcerPort userport.RBACEnforcerPort,
	roleBindingPersistencePort userport.RoleBindingPersistencePort,
	rolePersistencePort userport.RolePersistencePort,
	permissionPersistencePort userport.PermissionPersistencePort,
	userPersistencePort userport.UserPersistencePort,
	logger *slog.Logger,
) *RBACService {
	return &RBACService{
		rbacEnforcerPort:           rbacEnforcerPort,
		roleBindingPersistencePort: roleBindingPersistencePort,
		rolePersistencePort:        rolePersistencePort,
		permissionPersistencePort:  permissionPersistencePort,
		userPersistencePort:        userPersistencePort,
		logger:                     logger,
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
	user, err := s.userPersistencePort.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	allBindings, err := s.roleBindingPersistencePort.ListRoleBindings(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list role bindings: %w", err)
	}

	permissionMap := make(map[string]*usermodel.Permission)

	for _, binding := range allBindings.Items {
		if !bindingMatchesUser(binding, user) {
			continue
		}

		role, err := s.rolePersistencePort.GetRole(ctx, binding.Spec.RoleRef.UID)
		if err != nil {
			s.logger.WarnContext(ctx, "failed to get role for role binding",
				slog.String("roleUID", binding.Spec.RoleRef.UID.String()),
				slog.Any("error", err),
			)

			continue
		}

		for _, permissionID := range role.Spec.Permissions {
			if _, exists := permissionMap[permissionID]; exists {
				continue
			}

			permission, permErr := s.permissionPersistencePort.GetPermissionByName(ctx, permissionID)
			if permErr != nil {
				s.logger.WarnContext(ctx, "failed to get permission by name",
					slog.String("permissionID", permissionID),
					slog.Any("error", permErr),
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

	bindingsResp, err := s.roleBindingPersistencePort.ListRoleBindings(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list role bindings from persistence: %w", err)
	}

	usersResp, err := s.userPersistencePort.ListUsers(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list users for policy sync: %w", err)
	}

	var groupings []groupingPolicy

	for _, binding := range bindingsResp.Items {
		roleID := binding.Spec.RoleRef.UID.String()
		namespace := binding.Metadata.Namespace

		if binding.Spec.Subject.Name != "" {
			// Subject-based: resolve the specific user by email
			user, userErr := s.userPersistencePort.GetUserByEmail(ctx, binding.Spec.Subject.Name)
			if userErr != nil {
				s.logger.WarnContext(ctx, "failed to resolve subject for role binding during sync",
					slog.String("namespace", namespace),
					slog.String("name", binding.Metadata.Name),
					slog.String("subject", binding.Spec.Subject.Name),
					slog.Any("error", userErr),
				)

				continue
			}

			groupings = append(groupings, groupingPolicy{
				userID:    user.Metadata.UID.String(),
				roleID:    roleID,
				namespace: namespace,
			})

			continue
		}

		// LabelSelector-based: match against all users
		for _, user := range usersResp.Items {
			if labelsMatch(binding.Spec.LabelSelector, user.Metadata.Labels) {
				groupings = append(groupings, groupingPolicy{
					userID:    user.Metadata.UID.String(),
					roleID:    roleID,
					namespace: namespace,
				})
			}
		}
	}

	return policies, groupings, nil
}

// bindingMatchesUser returns true if the binding applies to the given user.
// A binding matches if:
//   - Subject.Name matches the user's email (subject-based), or
//   - all LabelSelector key/value pairs are present in the user's labels (label-selector-based).
func bindingMatchesUser(binding *usermodel.RoleBinding, user *usermodel.User) bool {
	if binding.Spec.Subject.Name != "" {
		return user.Spec.Email == binding.Spec.Subject.Name
	}

	return labelsMatch(binding.Spec.LabelSelector, user.Metadata.Labels)
}

// labelsMatch returns true if all selector pairs are present in labels.
func labelsMatch(selector, labels map[string]string) bool {
	if len(selector) == 0 {
		return false
	}

	for key, value := range selector {
		if labels[key] != value {
			return false
		}
	}

	return true
}
