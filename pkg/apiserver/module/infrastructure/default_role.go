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

// defaultRoleBuiltInPermissions lists the permission names every user should hold
// in the default namespace via the built-in default role: GET on every
// namespace-scoped resource. Admins can still customize the default role's
// permissions via the API; this set is treated as a floor that startup re-applies.
func defaultRoleBuiltInPermissions() []usermodel.PermissionSpec {
	specs := make([]usermodel.PermissionSpec, 0, len(usermodel.NamespaceScopedResources()))

	for _, resource := range usermodel.NamespaceScopedResources() {
		specs = append(specs, usermodel.PermissionSpec{
			Name:        resource + ":" + usermodel.ActionGet,
			Description: "Built-in: read access to " + resource + " in the default namespace",
			Resource:    resource,
			Action:      usermodel.ActionGet,
			IsBuiltIn:   true,
		})
	}

	return specs
}

// ensureDefaultPermissions persists the built-in permission docs the default role refers to,
// so RBACService.SyncPolicies can resolve them via PermissionPersistencePort.GetPermissionByName.
func ensureDefaultPermissions(
	ctx context.Context,
	permissionPersistencePort userport.PermissionPersistencePort,
	logger *slog.Logger,
) error {
	for _, spec := range defaultRoleBuiltInPermissions() {
		_, err := permissionPersistencePort.GetPermissionByName(ctx, spec.Name)
		if err == nil {
			continue
		}

		if !errors.Is(err, port.ErrResourceNotExist) {
			return fmt.Errorf("check built-in permission %q: %w", spec.Name, err)
		}

		logger.Info("creating built-in permission", slog.String("name", spec.Name))

		permission := usermodel.NewPermission(spec.Resource, spec.Action, spec.IsBuiltIn)
		permission.Spec.Description = spec.Description

		_, err = permissionPersistencePort.PutPermission(ctx, permission)
		if err != nil {
			return fmt.Errorf("create built-in permission %q: %w", spec.Name, err)
		}
	}

	return nil
}

// ensureDefaultRole creates the built-in "default" role if it does not exist and ensures it
// references every built-in permission from defaultRoleBuiltInPermissions. New permissions
// added to that list in later releases are picked up on next startup; admin-added permissions
// outside that list are preserved.
//
// The role is marked IsBuiltIn=true so it cannot be deleted, but its permissions can be
// further customized via the API (this hook only enforces the built-in floor).
func ensureDefaultRole(
	ctx context.Context,
	rolePersistencePort userport.RolePersistencePort,
	logger *slog.Logger,
) error {
	builtInNames := make([]string, 0, len(defaultRoleBuiltInPermissions()))
	for _, spec := range defaultRoleBuiltInPermissions() {
		builtInNames = append(builtInNames, spec.Name)
	}

	existing, err := rolePersistencePort.GetRoleByName(ctx, usermodel.RoleDefault)
	if err == nil {
		added := false

		for _, name := range builtInNames {
			if !existing.HasPermission(name) {
				existing.AddPermission(name)

				added = true
			}
		}

		if !added {
			return nil
		}

		logger.Info("adding missing built-in permissions to default role")

		_, err = rolePersistencePort.PutRole(ctx, existing)
		if err != nil {
			return fmt.Errorf("update default role with built-in permissions: %w", err)
		}

		return nil
	}

	if !errors.Is(err, port.ErrResourceNotExist) {
		return fmt.Errorf("check default role: %w", err)
	}

	logger.Info("creating built-in default role")

	defaultRole := usermodel.NewRole(usermodel.RoleDefault, true)
	defaultRole.Spec.Description = "Default role assigned to all new users on first login"
	defaultRole.Spec.Permissions = builtInNames

	_, err = rolePersistencePort.PutRole(ctx, defaultRole)
	if err != nil {
		return fmt.Errorf("create default role: %w", err)
	}

	return nil
}

// registerDefaultRoleHook registers a lifecycle hook to ensure
// the built-in default role and its permissions exist on startup.
func registerDefaultRoleHook(
	lifecycle fx.Lifecycle,
	rolePersistencePort userport.RolePersistencePort,
	permissionPersistencePort userport.PermissionPersistencePort,
	logger *slog.Logger,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			err := ensureDefaultPermissions(ctx, permissionPersistencePort, logger)
			if err != nil {
				return err
			}

			return ensureDefaultRole(ctx, rolePersistencePort, logger)
		},
		OnStop: nil,
	})
}
