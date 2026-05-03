package infrastructure

import (
	"context"
	"fmt"
	"log/slog"

	"go.uber.org/fx"

	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
)

// registerSyncPoliciesHook re-applies RBAC policies to the Casbin enforcer on startup,
// so existing users persisted across restarts are re-granted the built-in default role
// (and any matching RoleBindings) without waiting for a fresh login.
//
// Runs after registerDefaultRoleHook so the default role is present when sync executes.
func registerSyncPoliciesHook(
	lifecycle fx.Lifecycle,
	rbacUsecase userport.RBACUsecase,
	logger *slog.Logger,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("synchronizing RBAC policies on startup")

			err := rbacUsecase.SyncPolicies(ctx)
			if err != nil {
				return fmt.Errorf("sync RBAC policies on startup: %w", err)
			}

			return nil
		},
		OnStop: nil,
	})
}
