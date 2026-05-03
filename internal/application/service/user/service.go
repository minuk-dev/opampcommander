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
	mapper                     *helper.Mapper
	logger                     *slog.Logger
}

// New creates a new instance of the Service struct.
func New(
	userUsecase userport.UserUsecase,
	roleUsecase userport.RoleUsecase,
	roleBindingPersistencePort userport.RoleBindingPersistencePort,
	logger *slog.Logger,
) *Service {
	return &Service{
		userUsecase:                userUsecase,
		roleUsecase:                roleUsecase,
		roleBindingPersistencePort: roleBindingPersistencePort,
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

	return &v1.UserProfileResponse{
		User:  *s.mapper.MapUserToAPI(user),
		Roles: s.buildRoleEntries(ctx, user),
	}, nil
}

// buildRoleEntries returns a sorted slice of roles with their associated bindings.
// The built-in default role is always included even when no binding matches.
//
//nolint:funlen // Multi-step: list bindings, deduplicate roles, always append default, sort.
func (s *Service) buildRoleEntries(ctx context.Context, user *usermodel.User) []v1.UserRoleEntry {
	allBindings, err := s.roleBindingPersistencePort.ListRoleBindings(ctx, nil)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to list role bindings for user profile", slog.Any("error", err))

		//exhaustruct:ignore
		allBindings = &model.ListResponse[*usermodel.RoleBinding]{Items: nil}
	}

	type entry struct {
		role    *usermodel.Role
		binding *usermodel.RoleBinding
	}

	entryMap := make(map[string]entry)

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

		if _, exists := entryMap[binding.Spec.RoleRef.Name]; exists {
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

		b := binding
		entryMap[binding.Spec.RoleRef.Name] = entry{role: role, binding: b}
	}

	// Always include the built-in default role.
	if _, exists := entryMap[usermodel.RoleDefault]; !exists {
		defaultRole, defaultErr := s.roleUsecase.GetRoleByName(ctx, usermodel.RoleDefault)
		if defaultErr != nil {
			s.logger.WarnContext(ctx, "failed to get default role", slog.Any("error", defaultErr))
		} else {
			entryMap[usermodel.RoleDefault] = entry{role: defaultRole, binding: nil}
		}
	}

	entries := lo.Values(entryMap)

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].role.Spec.DisplayName < entries[j].role.Spec.DisplayName
	})

	return lo.Map(entries, func(roleEntry entry, _ int) v1.UserRoleEntry {
		roleAPI := s.mapper.MapRoleToAPI(roleEntry.role)
		//exhaustruct:ignore
		result := v1.UserRoleEntry{Role: *roleAPI}

		if roleEntry.binding != nil {
			bindingAPI := s.mapper.MapRoleBindingToAPI(roleEntry.binding)
			result.RoleBinding = bindingAPI
		}

		return result
	})
}

func roleBindingMatchesUser(binding *usermodel.RoleBinding, user *usermodel.User) bool {
	if len(binding.Spec.LabelSelector) == 0 {
		return false
	}

	return lo.EveryBy(lo.Entries(binding.Spec.LabelSelector), func(kv lo.Entry[string, string]) bool {
		return user.Metadata.Labels[kv.Key] == kv.Value
	})
}
