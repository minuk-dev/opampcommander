// Package infrastructure provides cross-cutting bootstrap wiring for the API server:
// the Casbin RBAC enforcer and the startup hooks that seed the default namespace,
// the built-in default role, and the RBAC policy sync. The HTTP/DB/messaging
// adapters live under module/adapter.
package infrastructure

import (
	"fmt"
	"log/slog"

	casbinModel "github.com/casbin/casbin/v2/model"
	mongodbadapter "github.com/casbin/mongodb-adapter/v4"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"

	casbinEnforcer "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/out/rbac/casbin"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	userport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/port"
)

// New creates the infrastructure bootstrap module: the Casbin RBAC enforcer and
// the startup hooks that seed the default namespace, default role, and RBAC policies.
func New() fx.Option {
	return fx.Module(
		"infrastructure",

		// RBAC (Casbin enforcer)
		provideRBACComponents(),

		// Default namespace initialization
		fx.Invoke(registerDefaultNamespaceHook),
		// Default role initialization
		fx.Invoke(registerDefaultRoleHook),
		// RBAC policy sync — must run after the default role is seeded.
		fx.Invoke(registerSyncPoliciesHook),
	)
}

// provideRBACComponents provides RBAC enforcer components.
func provideRBACComponents() fx.Option {
	return fx.Options(
		fx.Provide(
			provideCasbinEnforcer,
			fx.Annotate(
				Identity[*casbinEnforcer.Enforcer],
				fx.As(new(userport.RBACEnforcerPort)),
			),
		),
	)
}

func provideCasbinEnforcer(
	logger *slog.Logger,
	settings *config.ServerSettings,
	mongoClient *mongo.Client,
) (*casbinEnforcer.Enforcer, error) {
	rbacModel := defaultRBACModel()

	databaseName := settings.DatabaseSettings.DatabaseName
	if databaseName == "" {
		databaseName = "opampcommander"
	}

	adapter, err := mongodbadapter.NewAdapterByDB(
		mongoClient, &mongodbadapter.AdapterConfig{
			DatabaseName:   databaseName,
			CollectionName: "casbin_rules",
			Timeout:        0,
			IsFiltered:     false,
		},
	)
	if err != nil {
		logger.Warn("failed to create MongoDB adapter for Casbin, "+
			"falling back to in-memory",
			slog.Any("error", err),
		)

		fallback, fallbackErr := casbinEnforcer.NewEnforcerFromModel(
			logger, rbacModel,
		)
		if fallbackErr != nil {
			return nil, fmt.Errorf(
				"failed to create fallback casbin enforcer: %w", fallbackErr,
			)
		}

		return fallback, nil
	}

	enforcer, err := casbinEnforcer.NewEnforcerWithAdapter(
		logger, rbacModel, adapter,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	return enforcer, nil
}

func defaultRBACModel() casbinModel.Model {
	rbacModel, _ := casbinModel.NewModelFromString(`
[request_definition]
r = sub, dom, obj, act

[role_definition]
g = _, _, _

[policy_definition]
p = sub, dom, obj, act

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = (g(r.sub, p.sub, r.dom) || g(r.sub, p.sub, "*")) && \
  (p.dom == "*" || r.dom == p.dom) && \
  (p.obj == "*" || r.obj == p.obj) && \
  (p.act == "*" || r.act == p.act)
`)

	return rbacModel
}

// Identity is a generic function that returns the input value.
// It is a helper function to generate a function that returns the input value.
// It is used to provide a function as a interface.
func Identity[T any](a T) T {
	return a
}
