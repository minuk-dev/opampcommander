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

// ensureDefaultRole creates the built-in "default" role if it does not exist.
// The default role is assigned to all new users automatically on login.
// It is marked IsBuiltIn=true so it cannot be deleted, but its permissions can be changed.
func ensureDefaultRole(
	ctx context.Context,
	rolePersistencePort userport.RolePersistencePort,
	logger *slog.Logger,
) error {
	_, err := rolePersistencePort.GetRoleByName(ctx, usermodel.RoleDefault)
	if err == nil {
		return nil
	}

	if !errors.Is(err, port.ErrResourceNotExist) {
		return fmt.Errorf("check default role: %w", err)
	}

	logger.Info("creating built-in default role")

	defaultRole := usermodel.NewRole(usermodel.RoleDefault, true)
	defaultRole.Spec.Description = "Default role assigned to all new users on first login"

	_, err = rolePersistencePort.PutRole(ctx, defaultRole)
	if err != nil {
		return fmt.Errorf("create default role: %w", err)
	}

	return nil
}

// registerDefaultRoleHook registers a lifecycle hook to ensure
// the built-in default role exists on startup.
func registerDefaultRoleHook(
	lifecycle fx.Lifecycle,
	rolePersistencePort userport.RolePersistencePort,
	logger *slog.Logger,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return ensureDefaultRole(ctx, rolePersistencePort, logger)
		},
		OnStop: nil,
	})
}
