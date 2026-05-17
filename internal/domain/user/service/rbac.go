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
// Returns the union of permissions from (a) the built-in default role, granted to every user
// regardless of bindings, and (b) every RoleBinding whose Subjects name this user.
func (s *RBACService) GetUserPermissions(
	ctx context.Context,
	userID uuid.UUID,
) ([]*usermodel.Permission, error) {
	user, err := s.userPersistencePort.GetUser(ctx, userID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	allBindings, err := s.roleBindingPersistencePort.ListRoleBindings(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list role bindings: %w", err)
	}

	permissionMap := make(map[string]*usermodel.Permission)

	// Default role applies to every user even without a RoleBinding. Best-effort: log
	// and continue if the default role isn't seeded yet.
	defaultRole, defaultErr := s.rolePersistencePort.GetRoleByName(ctx, usermodel.RoleDefault)
	if defaultErr != nil {
		s.logger.WarnContext(ctx, "default role not found while resolving permissions",
			slog.Any("error", defaultErr),
		)
	} else {
		s.collectRolePermissions(ctx, defaultRole, permissionMap)
	}

	for _, binding := range allBindings.Items {
		if !binding.MatchesUser(user) {
			continue
		}

		role, err := s.rolePersistencePort.GetRoleByName(ctx, binding.Spec.RoleRef.Name)
		if err != nil {
			s.logger.WarnContext(ctx, "failed to get role for role binding",
				slog.String("roleRef", binding.Spec.RoleRef.Name),
				slog.Any("error", err),
			)

			continue
		}

		s.collectRolePermissions(ctx, role, permissionMap)
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

	// The Casbin MongoDB adapter rejects SavePolicy when there is nothing to insert
	// (it does an InsertMany that requires at least one document). On a fresh DB the
	// default role has no permissions and there are no users yet — skip persistence
	// in that case; the next mutation (user login, role/binding edit) will resync.
	if len(policies) == 0 && len(groupings) == 0 {
		return nil
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

	roleByName := make(map[string]*usermodel.Role, len(rolesResp.Items))
	for _, role := range rolesResp.Items {
		roleByName[role.Spec.DisplayName] = role
	}

	policies := s.collectNamedPolicies(ctx, rolesResp.Items)

	groupings, err := s.collectGroupingPolicies(ctx, roleByName)
	if err != nil {
		return nil, nil, err
	}

	return policies, groupings, nil
}

func (s *RBACService) collectNamedPolicies(
	ctx context.Context,
	roles []*usermodel.Role,
) []namedPolicy {
	var policies []namedPolicy

	for _, role := range roles {
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

	return policies
}

func (s *RBACService) collectGroupingPolicies(
	ctx context.Context,
	roleByName map[string]*usermodel.Role,
) ([]groupingPolicy, error) {
	bindingsResp, err := s.roleBindingPersistencePort.ListRoleBindings(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list role bindings from persistence: %w", err)
	}

	usersResp, err := s.userPersistencePort.ListUsers(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list users for policy sync: %w", err)
	}

	// Index users by email so each Subject can be resolved in O(1) instead of
	// scanning every user per binding.
	usersByEmail := make(map[string]*usermodel.User, len(usersResp.Items))
	for _, user := range usersResp.Items {
		if user.Spec.Email == "" {
			continue
		}

		usersByEmail[user.Spec.Email] = user
	}

	groupings := s.defaultRoleGroupings(roleByName, usersResp.Items)

	for _, binding := range bindingsResp.Items {
		groupings = append(groupings, s.bindingGroupings(ctx, binding, roleByName, usersByEmail)...)
	}

	return groupings, nil
}

// defaultRoleGroupings auto-assigns the built-in default role to every user in the "default" namespace.
func (s *RBACService) defaultRoleGroupings(
	roleByName map[string]*usermodel.Role,
	users []*usermodel.User,
) []groupingPolicy {
	defaultRole, ok := roleByName[usermodel.RoleDefault]
	if !ok {
		return nil
	}

	defaultRoleUID := defaultRole.Metadata.UID.String()
	groupings := make([]groupingPolicy, 0, len(users))

	for _, user := range users {
		groupings = append(groupings, groupingPolicy{
			userID:    user.Metadata.UID.String(),
			roleID:    defaultRoleUID,
			namespace: usermodel.DefaultNamespace,
		})
	}

	return groupings
}

// bindingGroupings produces grouping policies for one RoleBinding by resolving each Subject
// against the email→user index. Unknown roles or unresolvable subjects are logged and skipped.
func (s *RBACService) bindingGroupings(
	ctx context.Context,
	binding *usermodel.RoleBinding,
	roleByName map[string]*usermodel.Role,
	usersByEmail map[string]*usermodel.User,
) []groupingPolicy {
	role, ok := roleByName[binding.Spec.RoleRef.Name]
	if !ok {
		s.logger.WarnContext(ctx, "role not found for role binding during sync",
			slog.String("namespace", binding.Metadata.Namespace),
			slog.String("name", binding.Metadata.Name),
			slog.String("roleRef", binding.Spec.RoleRef.Name),
		)

		return nil
	}

	namespace := binding.Metadata.Namespace
	roleUID := role.Metadata.UID.String()
	groupings := make([]groupingPolicy, 0, len(binding.Spec.Subjects))

	for _, subject := range binding.Spec.Subjects {
		if subject.Kind != usermodel.SubjectKindUser || subject.Name == "" {
			continue
		}

		user, found := usersByEmail[subject.Name]
		if !found {
			continue
		}

		groupings = append(groupings, groupingPolicy{
			userID:    user.Metadata.UID.String(),
			roleID:    roleUID,
			namespace: namespace,
		})
	}

	return groupings
}

// collectRolePermissions resolves every permission of role and inserts it into out (deduped by ID).
func (s *RBACService) collectRolePermissions(
	ctx context.Context,
	role *usermodel.Role,
	out map[string]*usermodel.Permission,
) {
	for _, permissionID := range role.Spec.Permissions {
		if _, exists := out[permissionID]; exists {
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

		out[permissionID] = permission
	}
}
