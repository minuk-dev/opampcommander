// Package rbac provides application services for RBAC authorization.
package rbac

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/google/uuid"
	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/internal/application/helper"
	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
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
	userUsecase                userport.UserUsecase
	mapper                     *helper.Mapper
	logger                     *slog.Logger
}

// New creates a new instance of the Service struct.
func New(
	rbacUsecase userport.RBACUsecase,
	roleUsecase userport.RoleUsecase,
	permissionUsecase userport.PermissionUsecase,
	roleBindingPersistencePort userport.RoleBindingPersistencePort,
	userUsecase userport.UserUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		rbacUsecase:                rbacUsecase,
		roleUsecase:                roleUsecase,
		permissionUsecase:          permissionUsecase,
		roleBindingPersistencePort: roleBindingPersistencePort,
		userUsecase:                userUsecase,
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
func (s *Service) GetUserRoles(
	ctx context.Context,
	userID uuid.UUID,
) (*v1.ListResponse[v1.Role], error) {
	user, err := s.userUsecase.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return s.buildRolesResponse(ctx, user), nil
}

// GetMyRoles implements [applicationport.RBACManageUsecase].
// Looks up the user by email — avoids an extra DB round-trip when the caller already holds the email.
func (s *Service) GetMyRoles(ctx context.Context, email string) (*v1.ListResponse[v1.Role], error) {
	user, err := s.userUsecase.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return s.buildRolesResponse(ctx, user), nil
}

// GetUserRoleBindings implements [applicationport.RBACManageUsecase].
func (s *Service) GetUserRoleBindings(
	ctx context.Context,
	userID uuid.UUID,
) (*v1.ListResponse[v1.RoleBinding], error) {
	user, err := s.userUsecase.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return s.buildRoleBindingsResponse(ctx, user), nil
}

// GetMyRoleBindings implements [applicationport.RBACManageUsecase].
// Looks up the user by email — avoids an extra DB round-trip when the caller already holds the email.
func (s *Service) GetMyRoleBindings(ctx context.Context, email string) (*v1.ListResponse[v1.RoleBinding], error) {
	user, err := s.userUsecase.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return s.buildRoleBindingsResponse(ctx, user), nil
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

// buildRolesResponse builds a sorted role list for the given user, always including the default role.
//
//nolint:funlen // Multi-step: list bindings, deduplicate roles, always append default, sort.
func (s *Service) buildRolesResponse(ctx context.Context, user *usermodel.User) *v1.ListResponse[v1.Role] {
	allBindings, err := s.roleBindingPersistencePort.ListRoleBindings(ctx, nil)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to list role bindings for user roles", slog.Any("error", err))

		allBindings = &model.ListResponse[*usermodel.RoleBinding]{
			RemainingItemCount: 0,
			Continue:           "",
			Items:              nil,
		}
	}

	roleMap := make(map[string]*usermodel.Role)

	for _, binding := range allBindings.Items {
		if !roleBindingMatchesUser(binding, user) {
			continue
		}

		if binding.Spec.RoleRef.Name == "" {
			s.logger.WarnContext(ctx, "skipping role binding with empty role ref name",
				slog.String("namespace", binding.Metadata.Namespace),
				slog.String("name", binding.Metadata.Name),
			)

			continue
		}

		if _, exists := roleMap[binding.Spec.RoleRef.Name]; exists {
			continue
		}

		role, roleErr := s.roleUsecase.GetRoleByName(ctx, binding.Spec.RoleRef.Name)
		if roleErr != nil {
			s.logger.WarnContext(ctx, "failed to get role for role binding",
				slog.String("roleRef", binding.Spec.RoleRef.Name),
				slog.Any("error", roleErr),
			)

			continue
		}

		roleMap[binding.Spec.RoleRef.Name] = role
	}

	// Always include the built-in default role — auto-assigned to all users via SyncPolicies.
	if _, exists := roleMap[usermodel.RoleDefault]; !exists {
		defaultRole, defaultErr := s.roleUsecase.GetRoleByName(ctx, usermodel.RoleDefault)
		if defaultErr != nil {
			s.logger.WarnContext(ctx, "failed to get default role", slog.Any("error", defaultErr))
		} else {
			roleMap[usermodel.RoleDefault] = defaultRole
		}
	}

	roles := make([]*usermodel.Role, 0, len(roleMap))
	for _, role := range roleMap {
		roles = append(roles, role)
	}

	sort.Slice(roles, func(i, j int) bool {
		return roles[i].Spec.DisplayName < roles[j].Spec.DisplayName
	})

	return &v1.ListResponse[v1.Role]{
		Kind:       v1.RoleKind,
		APIVersion: v1.APIVersion,
		Metadata:   v1.ListMeta{Continue: "", RemainingItemCount: 0},
		Items: lo.Map(roles, func(role *usermodel.Role, _ int) v1.Role {
			return *s.mapper.MapRoleToAPI(role)
		}),
	}
}

// buildRoleBindingsResponse returns RoleBindings whose LabelSelector matches the user's labels,
// sorted by namespace then name.
func (s *Service) buildRoleBindingsResponse(
	ctx context.Context,
	user *usermodel.User,
) *v1.ListResponse[v1.RoleBinding] {
	allBindings, err := s.roleBindingPersistencePort.ListRoleBindings(ctx, nil)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to list role bindings", slog.Any("error", err))

		return &v1.ListResponse[v1.RoleBinding]{
			Kind:       v1.RoleBindingKind,
			APIVersion: v1.APIVersion,
			Metadata:   v1.ListMeta{Continue: "", RemainingItemCount: 0},
			Items:      []v1.RoleBinding{},
		}
	}

	matching := make([]*usermodel.RoleBinding, 0)

	for _, binding := range allBindings.Items {
		if roleBindingMatchesUser(binding, user) {
			matching = append(matching, binding)
		}
	}

	sort.Slice(matching, func(i, j int) bool {
		if matching[i].Metadata.Namespace != matching[j].Metadata.Namespace {
			return matching[i].Metadata.Namespace < matching[j].Metadata.Namespace
		}

		return matching[i].Metadata.Name < matching[j].Metadata.Name
	})

	return &v1.ListResponse[v1.RoleBinding]{
		Kind:       v1.RoleBindingKind,
		APIVersion: v1.APIVersion,
		Metadata:   v1.ListMeta{Continue: "", RemainingItemCount: 0},
		Items: lo.Map(matching, func(binding *usermodel.RoleBinding, _ int) v1.RoleBinding {
			return *s.mapper.MapRoleBindingToAPI(binding)
		}),
	}
}

// roleBindingMatchesUser returns true if the binding's labelSelector matches the user's labels.
// An empty selector never matches (deny-by-default safety).
func roleBindingMatchesUser(binding *usermodel.RoleBinding, user *usermodel.User) bool {
	if len(binding.Spec.LabelSelector) == 0 {
		return false
	}

	for key, value := range binding.Spec.LabelSelector {
		if user.Metadata.Labels[key] != value {
			return false
		}
	}

	return true
}
