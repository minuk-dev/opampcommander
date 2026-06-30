// Package auth provides the implementation of the AuthProvisioningUsecase interface.
//
// It owns the user-provisioning logic that runs on every successful login (creating
// or updating the user record and re-syncing RBAC policies). This logic used to live
// in the HTTP auth controllers; it is kept here so those primary adapters depend only
// on the application layer, not on the domain.
package auth

import (
	"context"
	"errors"
	"log/slog"
	"time"

	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/usecase"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user"
	userport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/port"
)

var _ usecase.AuthProvisioningUsecase = (*Service)(nil)

// Service implements AuthProvisioningUsecase.
type Service struct {
	logger      *slog.Logger
	userUsecase userport.UserUsecase
	rbacUsecase userport.RBACUsecase
}

// New creates a new instance of the Service struct.
func New(
	userUsecase userport.UserUsecase,
	rbacUsecase userport.RBACUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		logger:      logger,
		userUsecase: userUsecase,
		rbacUsecase: rbacUsecase,
	}
}

// EnsureUserOnLogin creates or updates a user record on login.
// Always syncs provider labels and re-applies RBAC policies so the freshly-saved user
// picks up the built-in default role (and any matching bindings).
// Failures are logged but do not block the login flow.
func (s *Service) EnsureUserOnLogin(ctx context.Context, provisioning applicationport.LoginProvisioning) {
	existing, err := s.userUsecase.GetUserByEmail(ctx, provisioning.Email)

	switch {
	case err == nil && existing != nil:
		s.syncLabels(existing, provisioning.Provider, provisioning.Groups)

		existing.Metadata.UpdatedAt = time.Now()

		saveErr := s.userUsecase.SaveUser(ctx, existing)
		if saveErr != nil {
			s.logger.Warn("failed to update user on login",
				slog.String("email", provisioning.Email),
				slog.Any("error", saveErr),
			)

			return
		}

		s.syncRBACPolicies(ctx, provisioning.Email)

		return
	case err != nil && !errors.Is(err, model.ErrResourceNotExist):
		s.logger.Warn("failed to check user existence on login",
			slog.String("email", provisioning.Email),
			slog.Any("error", err),
		)

		return
	}

	// No active user with this email. If a soft-deleted record exists, the user was
	// deliberately deleted: do not resurrect or duplicate it. They can still authenticate
	// but stay without a record (RBAC denies) until an admin restores them.
	if s.isDeletedUser(ctx, provisioning.Email) {
		return
	}

	newUser := usermodel.NewUserWithIdentity(
		provisioning.Provider, provisioning.Username, provisioning.Email, provisioning.Username,
	)
	s.syncLabels(newUser, provisioning.Provider, provisioning.Groups)

	saveErr := s.userUsecase.SaveUser(ctx, newUser)
	if saveErr != nil {
		s.logger.Warn("failed to create user on login",
			slog.String("email", provisioning.Email),
			slog.Any("error", saveErr),
		)

		return
	}

	s.syncRBACPolicies(ctx, provisioning.Email)
}

// isDeletedUser reports whether a soft-deleted user record exists for the email.
// On lookup failure it returns true (fail-closed): recreating on a transient error is the
// behaviour that produced duplicate accounts, so we'd rather skip creation and let the user retry.
func (s *Service) isDeletedUser(ctx context.Context, email string) bool {
	deleted, err := s.userUsecase.GetUserByEmailIncludingDeleted(ctx, email)
	if err != nil {
		if errors.Is(err, model.ErrResourceNotExist) {
			return false
		}

		s.logger.Warn("failed to check for soft-deleted user on login; skipping user creation",
			slog.String("email", email),
			slog.Any("error", err),
		)

		return true
	}

	s.logger.Warn("not recreating soft-deleted user on login",
		slog.String("email", email),
		slog.String("uid", deleted.Metadata.UID.String()),
	)

	return true
}

// syncRBACPolicies re-runs the Casbin policy sync so newly persisted users/bindings take effect.
// Best-effort: failures are logged but do not block the login flow.
func (s *Service) syncRBACPolicies(ctx context.Context, email string) {
	err := s.rbacUsecase.SyncPolicies(ctx)
	if err != nil {
		s.logger.Warn("failed to sync RBAC policies after login",
			slog.String("email", email),
			slog.Any("error", err),
		)
	}
}

// syncLabels updates the user's metadata labels to reflect the current login session.
// Existing non-provider labels are preserved.
func (s *Service) syncLabels(user *usermodel.User, provider string, groups []string) {
	// Remove stale provider-specific labels before re-setting.
	for key := range user.Metadata.Labels {
		if len(key) > len(usermodel.LabelGitHubOrg) && key[:len(usermodel.LabelGitHubOrg)] == usermodel.LabelGitHubOrg {
			delete(user.Metadata.Labels, key)
		}
	}

	user.SetLabel(usermodel.LabelLoginType, provider)

	if provider == usermodel.IdentityProviderGitHub {
		for _, org := range groups {
			user.SetLabel(usermodel.LabelGitHubOrg+org, "true")
		}
	}
}
