package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/internal/domain/port"
	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
)

// ensureDefaultMemberRole creates the built-in "Member" role if it does not exist.
// The Member role is assigned to all new users automatically on login.
// It is marked IsBuiltIn=true so it cannot be deleted, but its permissions can be changed.
func ensureDefaultMemberRole(
	ctx context.Context,
	rolePersistencePort userport.RolePersistencePort,
	logger *slog.Logger,
) error {
	_, err := rolePersistencePort.GetRoleByName(ctx, usermodel.RoleMember)
	if err == nil {
		return nil
	}

	if !errors.Is(err, port.ErrResourceNotExist) {
		return fmt.Errorf("check default member role: %w", err)
	}

	logger.Info("creating default Member role")

	memberRole := usermodel.NewRole(usermodel.RoleMember, true)
	memberRole.Spec.Description = "Default role assigned to all new users on first login"

	_, err = rolePersistencePort.PutRole(ctx, memberRole)
	if err != nil {
		return fmt.Errorf("create default member role: %w", err)
	}

	return nil
}

// registerDefaultRoleHook registers a lifecycle hook to ensure
// the default Member role exists on startup.
func registerDefaultRoleHook(
	lifecycle fx.Lifecycle,
	rolePersistencePort userport.RolePersistencePort,
	logger *slog.Logger,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return ensureDefaultMemberRole(ctx, rolePersistencePort, logger)
		},
		OnStop: nil,
	})
}
