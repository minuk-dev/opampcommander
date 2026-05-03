// Package user provides application services for user management.
package user

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

var _ applicationport.UserManageUsecase = (*Service)(nil)

// Service is a struct that implements the UserManageUsecase interface.
type Service struct {
	userUsecase                userport.UserUsecase
	roleUsecase                userport.RoleUsecase
	roleBindingPersistencePort userport.RoleBindingPersistencePort
	rbacEnforcerPort           userport.RBACEnforcerPort
	mapper                     *helper.Mapper
	logger                     *slog.Logger
}

// New creates a new instance of the Service struct.
func New(
	userUsecase userport.UserUsecase,
	roleUsecase userport.RoleUsecase,
	roleBindingPersistencePort userport.RoleBindingPersistencePort,
	rbacEnforcerPort userport.RBACEnforcerPort,
	logger *slog.Logger,
) *Service {
	return &Service{
		userUsecase:                userUsecase,
		roleUsecase:                roleUsecase,
		roleBindingPersistencePort: roleBindingPersistencePort,
		rbacEnforcerPort:           rbacEnforcerPort,
		mapper:                     helper.NewMapper(),
		logger:                     logger,
	}
}

// GetUser implements [applicationport.UserManageUsecase].
func (s *Service) GetUser(ctx context.Context, uid uuid.UUID) (*v1.User, error) {
	user, err := s.userUsecase.GetUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return s.mapper.MapUserToAPI(user), nil
}

// GetUserByEmail implements [applicationport.UserManageUsecase].
func (s *Service) GetUserByEmail(ctx context.Context, email string) (*v1.User, error) {
	user, err := s.userUsecase.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return s.mapper.MapUserToAPI(user), nil
}

// ListUsers implements [applicationport.UserManageUsecase].
func (s *Service) ListUsers(
	ctx context.Context,
	options *model.ListOptions,
) (*v1.ListResponse[v1.User], error) {
	response, err := s.userUsecase.ListUsers(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return &v1.ListResponse[v1.User]{
		Kind:       v1.UserKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.ListMeta{
			Continue:           response.Continue,
			RemainingItemCount: response.RemainingItemCount,
		},
		Items: lo.Map(response.Items, func(user *usermodel.User, _ int) v1.User {
			return *s.mapper.MapUserToAPI(user)
		}),
	}, nil
}

// CreateUser implements [applicationport.UserManageUsecase].
func (s *Service) CreateUser(ctx context.Context, apiUser *v1.User) (*v1.User, error) {
	domainUser := usermodel.NewUser(apiUser.Spec.Email, apiUser.Spec.Username)

	for key, value := range apiUser.Metadata.Labels {
		domainUser.SetLabel(key, value)
	}

	err := s.userUsecase.SaveUser(ctx, domainUser)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return s.mapper.MapUserToAPI(domainUser), nil
}

// DeleteUser implements [applicationport.UserManageUsecase].
func (s *Service) DeleteUser(ctx context.Context, uid uuid.UUID) error {
	err := s.userUsecase.DeleteUser(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// GetMyProfile implements [applicationport.UserManageUsecase].
// Returns the user's profile together with all roles and the binding that granted each role.
func (s *Service) GetMyProfile(ctx context.Context, email string) (*v1.UserProfileResponse, error) {
	user, err := s.userUsecase.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	roles, err := s.buildRoleEntries(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to build role entries: %w", err)
	}

	return &v1.UserProfileResponse{
		User:  *s.mapper.MapUserToAPI(user),
		Roles: roles,
	}, nil
}

// buildRoleEntries returns the roles actually loaded into the RBAC enforcer for the user,
// each paired (best-effort) with the source RoleBinding that produced the assignment.
//
// The enforcer (Casbin) is the source of truth for "what roles is this user enforced to have".
// The RoleBinding lookup is best-effort context: if labels or bindings have changed since the
// last policy sync, no binding may match — we still surface the role so the divergence is visible.
func (s *Service) buildRoleEntries(ctx context.Context, user *usermodel.User) ([]v1.UserRoleEntry, error) {
	assignments, err := s.rbacEnforcerPort.GetRoleAssignmentsForUser(ctx, user.Metadata.UID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get role assignments from enforcer: %w", err)
	}

	allBindings, err := s.roleBindingPersistencePort.ListRoleBindings(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list role bindings: %w", err)
	}

	// Sort bindings so the matching binding lookup is deterministic when multiple match.
	sort.Slice(allBindings.Items, func(i, j int) bool {
		a, b := allBindings.Items[i], allBindings.Items[j]
		if a.Metadata.Namespace != b.Metadata.Namespace {
			return a.Metadata.Namespace < b.Metadata.Namespace
		}

		return a.Metadata.Name < b.Metadata.Name
	})

	entries := make([]v1.UserRoleEntry, 0, len(assignments))

	for _, assignment := range assignments {
		role, lookupErr := s.lookupRoleByUIDString(ctx, assignment.RoleUID)
		if lookupErr != nil {
			s.logger.WarnContext(ctx, "role from enforcer not found in store — skipping",
				slog.String("roleUID", assignment.RoleUID),
				slog.String("namespace", assignment.Namespace),
				slog.Any("error", lookupErr),
			)

			continue
		}

		matched := findMatchingBinding(allBindings.Items, role.Spec.DisplayName, assignment.Namespace, user)

		roleAPI := s.mapper.MapRoleToAPI(role)
		//exhaustruct:ignore
		entry := v1.UserRoleEntry{Role: *roleAPI}

		if matched != nil {
			entry.RoleBinding = s.mapper.MapRoleBindingToAPI(matched)
		}

		entries = append(entries, entry)
	}

	sort.Slice(entries, func(left, right int) bool {
		if entries[left].Role.Spec.DisplayName != entries[right].Role.Spec.DisplayName {
			return entries[left].Role.Spec.DisplayName < entries[right].Role.Spec.DisplayName
		}

		// Stable secondary key: namespace of the binding (empty for unmatched).
		return bindingNamespace(entries[left].RoleBinding) < bindingNamespace(entries[right].RoleBinding)
	})

	return entries, nil
}

func (s *Service) lookupRoleByUIDString(ctx context.Context, raw string) (*usermodel.Role, error) {
	uid, err := uuid.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid role UID %q: %w", raw, err)
	}

	role, err := s.roleUsecase.GetRole(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("get role by UID: %w", err)
	}

	return role, nil
}

// findMatchingBinding returns the first binding whose namespace, role reference, and label
// selector all match — i.e. the binding that would currently produce this assignment.
// Returns nil when no binding matches (assignment came from a stale sync or default-role injection).
func findMatchingBinding(
	bindings []*usermodel.RoleBinding,
	roleName, namespace string,
	user *usermodel.User,
) *usermodel.RoleBinding {
	for _, binding := range bindings {
		if binding.Metadata.Namespace != namespace {
			continue
		}

		if binding.Spec.RoleRef.Name != roleName {
			continue
		}

		if !labelSelectorMatches(binding.Spec.LabelSelector, user.Metadata.Labels) {
			continue
		}

		return binding
	}

	return nil
}

func labelSelectorMatches(selector, labels map[string]string) bool {
	if len(selector) == 0 {
		return false
	}

	return lo.EveryBy(lo.Entries(selector), func(kv lo.Entry[string, string]) bool {
		return labels[kv.Key] == kv.Value
	})
}

func bindingNamespace(b *v1.RoleBinding) string {
	if b == nil {
		return ""
	}

	return b.Metadata.Namespace
}
