// Package user provides application services for user management.
package user

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"k8s.io/utils/clock"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/helper"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user"
	userport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/security"
)

var _ applicationport.UserManageUsecase = (*Service)(nil)

// Service is a struct that implements the UserManageUsecase interface.
type Service struct {
	userUsecase                userport.UserUsecase
	roleUsecase                userport.RoleUsecase
	roleBindingPersistencePort userport.RoleBindingPersistencePort
	rbacEnforcerPort           userport.RBACEnforcerPort
	rbacUsecase                userport.RBACUsecase
	passwordHasher             *security.PasswordHasher
	mapper                     *helper.Mapper
	logger                     *slog.Logger
}

// New creates a new instance of the Service struct.
func New(
	userUsecase userport.UserUsecase,
	roleUsecase userport.RoleUsecase,
	roleBindingPersistencePort userport.RoleBindingPersistencePort,
	rbacEnforcerPort userport.RBACEnforcerPort,
	rbacUsecase userport.RBACUsecase,
	passwordHasher *security.PasswordHasher,
	logger *slog.Logger,
) *Service {
	return &Service{
		userUsecase:                userUsecase,
		roleUsecase:                roleUsecase,
		roleBindingPersistencePort: roleBindingPersistencePort,
		rbacEnforcerPort:           rbacEnforcerPort,
		rbacUsecase:                rbacUsecase,
		passwordHasher:             passwordHasher,
		mapper:                     helper.NewMapper(clock.RealClock{}, 0),
		logger:                     logger,
	}
}

// GetUser implements [applicationport.UserManageUsecase].
func (s *Service) GetUser(ctx context.Context, uid uuid.UUID, options *applicationport.GetOptions) (*v1.User, error) {
	user, err := s.userUsecase.GetUser(ctx, uid, options.ToDomain())
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
	options *applicationport.ListOptions,
) (*v1.ListResponse[v1.User], error) {
	response, err := s.userUsecase.ListUsers(ctx, options.ToDomain())
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
// When a password is supplied, the user is provisioned for basic (username/password) login:
// the plaintext is hashed (peppered + salted) and only the hash is persisted, and a "basic"
// identity + login-type label is attached so RBAC default-role enrollment works.
func (s *Service) CreateUser(ctx context.Context, apiUser *v1.User) (*v1.User, error) {
	domainUser := usermodel.NewUser(apiUser.Spec.Email, apiUser.Spec.Username)

	for key, value := range apiUser.Metadata.Labels {
		domainUser.SetLabel(key, value)
	}

	if apiUser.Spec.Password != "" {
		err := s.applyBasicAuth(ctx, domainUser, apiUser.Spec.Username, apiUser.Spec.Email, apiUser.Spec.Password)
		if err != nil {
			return nil, err
		}
	}

	err := s.userUsecase.SaveUser(ctx, domainUser)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Re-sync Casbin so the new user is enrolled in the built-in default role
	// grouping. Without this, admin-API-created users would have no Casbin
	// groupings and every RBAC-gated request would 403 until the next event
	// that triggers SyncPolicies (e.g. their first login or a rolebinding edit).
	syncErr := s.rbacUsecase.SyncPolicies(ctx)
	if syncErr != nil {
		s.logger.WarnContext(ctx, "failed to sync RBAC policies after user create — default role grouping may be delayed",
			slog.String("email", domainUser.Spec.Email),
			slog.Any("error", syncErr),
		)
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

	bindingsResp, err := s.roleBindingPersistencePort.ListRoleBindings(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list role bindings: %w", err)
	}

	// Pre-sort so binding lookup is deterministic when multiple bindings could match.
	bindings := bindingsResp.Items
	slices.SortFunc(bindings, compareBindings)

	entries := lo.FilterMap(assignments,
		func(assignment userport.UserRoleAssignment, _ int) (v1.UserRoleEntry, bool) {
			role, lookupErr := s.resolveRole(ctx, assignment.RoleUID)
			if lookupErr != nil {
				s.logger.WarnContext(ctx, "role from enforcer not found in store — skipping",
					slog.String("roleUID", assignment.RoleUID),
					slog.String("namespace", assignment.Namespace),
					slog.Any("error", lookupErr),
				)

				//exhaustruct:ignore
				return v1.UserRoleEntry{}, false
			}

			binding := findMatchingBinding(bindings, role.Spec.DisplayName, assignment.Namespace, user)

			return s.toUserRoleEntry(role, binding), true
		})

	slices.SortFunc(entries, compareEntries)

	return entries, nil
}

// applyBasicAuth validates the request, hashes the plaintext password, and attaches the basic
// credential, a "basic" identity, and the login-type label to the domain user.
// Basic-auth login is keyed on username and identified by email, so both must be present and the
// username must be unique among active users. Bad input (empty email/username, duplicate username,
// or basic auth disabled because no pepper is configured) is wrapped in [model.ErrInvalidArgument]
// so the controller surfaces it as a 400.
func (s *Service) applyBasicAuth(ctx context.Context, user *usermodel.User, username, email, password string) error {
	usernameErr := usermodel.ValidateUsername(username)
	if usernameErr != nil {
		return fmt.Errorf("%w: %w", model.ErrInvalidArgument, usernameErr)
	}

	if email == "" {
		return fmt.Errorf("%w: email is required for a basic-auth user", model.ErrInvalidArgument)
	}

	err := s.ensureUsernameAvailable(ctx, username)
	if err != nil {
		return err
	}

	hash, err := s.passwordHasher.Hash(password)
	if err != nil {
		if errors.Is(err, security.ErrBasicAuthDisabled) {
			return fmt.Errorf("%w: %w", model.ErrInvalidArgument, err)
		}

		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.SetPasswordHash(hash)
	user.AddIdentity(usermodel.UserIdentity{
		Provider:       usermodel.IdentityProviderBasic,
		ProviderUserID: username,
		Email:          email,
		DisplayName:    username,
	})
	user.SetLabel(usermodel.LabelLoginType, usermodel.IdentityProviderBasic)

	return nil
}

// ensureUsernameAvailable returns [model.ErrInvalidArgument] when an active user already has the
// given username. Basic-auth login resolves a user by username, so duplicates would make login
// ambiguous (and could shadow a legitimate user). A not-found lookup means the username is free.
func (s *Service) ensureUsernameAvailable(ctx context.Context, username string) error {
	_, err := s.userUsecase.GetUserByUsername(ctx, username)
	if err == nil {
		return fmt.Errorf("%w: username %q is already in use", model.ErrInvalidArgument, username)
	}

	if !errors.Is(err, model.ErrResourceNotExist) {
		return fmt.Errorf("failed to check username availability: %w", err)
	}

	return nil
}

func (s *Service) resolveRole(ctx context.Context, raw string) (*usermodel.Role, error) {
	uid, err := uuid.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid role UID %q: %w", raw, err)
	}

	role, err := s.roleUsecase.GetRole(ctx, uid, nil)
	if err != nil {
		return nil, fmt.Errorf("get role by UID: %w", err)
	}

	return role, nil
}

func (s *Service) toUserRoleEntry(role *usermodel.Role, binding *usermodel.RoleBinding) v1.UserRoleEntry {
	//exhaustruct:ignore
	entry := v1.UserRoleEntry{Role: *s.mapper.MapRoleToAPI(role)}
	if binding != nil {
		entry.RoleBinding = s.mapper.MapRoleBindingToAPI(binding)
	}

	return entry
}

// findMatchingBinding returns the first binding whose namespace, role reference, and subjects
// all match — i.e. the binding that would currently produce this assignment.
// Returns nil when no binding matches (assignment came from a stale sync or default-role injection).
func findMatchingBinding(
	bindings []*usermodel.RoleBinding,
	roleName, namespace string,
	user *usermodel.User,
) *usermodel.RoleBinding {
	matches := func(b *usermodel.RoleBinding) bool {
		return b.Metadata.Namespace == namespace &&
			b.Spec.RoleRef.Name == roleName &&
			b.MatchesUser(user)
	}

	binding, ok := lo.Find(bindings, matches)
	if !ok {
		return nil
	}

	return binding
}

func compareBindings(a, b *usermodel.RoleBinding) int {
	return cmp.Or(
		cmp.Compare(a.Metadata.Namespace, b.Metadata.Namespace),
		cmp.Compare(a.Metadata.Name, b.Metadata.Name),
	)
}

func compareEntries(a, b v1.UserRoleEntry) int {
	return cmp.Or(
		cmp.Compare(a.Role.Spec.DisplayName, b.Role.Spec.DisplayName),
		cmp.Compare(bindingNamespace(a.RoleBinding), bindingNamespace(b.RoleBinding)),
	)
}

func bindingNamespace(b *v1.RoleBinding) string {
	if b == nil {
		return ""
	}

	return b.Metadata.Namespace
}
