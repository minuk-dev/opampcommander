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

	casbinEnforcer "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/rbac/casbin"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	userport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/port"
)

// New creates the infrastructure bootstrap module: the Casbin RBAC enforcer and
// the startup hooks that seed the default namespace, default role, and RBAC policies.
//
// The database type selects how the Casbin enforcer stores its policies: the
// explicit "mongodb" persists them in a "casbin_rules" collection, while any
// other value (including the default/empty, i.e. standalone) keeps them in
// process memory and does not depend on a *mongo.Client.
func New(databaseType config.DatabaseType) fx.Option {
	return fx.Module(
		"infrastructure",

		// RBAC (Casbin enforcer)
		provideRBACComponents(databaseType),

		// Initial manifest reconciliation — seeds the default namespace, built-in
		// roles, and their permissions declaratively from BootstrapSettings.Dir.
		fx.Invoke(registerBootstrapHook),
		// RBAC policy sync — must run after the bootstrap manifests are applied.
		fx.Invoke(registerSyncPoliciesHook),
	)
}

// provideRBACComponents provides RBAC enforcer components.
func provideRBACComponents(databaseType config.DatabaseType) fx.Option {
	enforcerProvider := any(provideInMemoryCasbinEnforcer)
	if databaseType == config.DatabaseTypeMongoDB {
		enforcerProvider = provideCasbinEnforcer
	}

	return fx.Options(
		fx.Provide(
			enforcerProvider,
			fx.Annotate(
				Identity[*casbinEnforcer.Enforcer],
				fx.As(new(userport.RBACEnforcerPort)),
			),
		),
	)
}

// provideInMemoryCasbinEnforcer builds a Casbin enforcer whose policies live only
// in process memory. Used in standalone mode where there is no *mongo.Client.
func provideInMemoryCasbinEnforcer(
	logger *slog.Logger,
) (*casbinEnforcer.Enforcer, error) {
	enforcer, err := casbinEnforcer.NewInMemoryEnforcer(logger, defaultRBACModel())
	if err != nil {
		return nil, fmt.Errorf("failed to create in-memory casbin enforcer: %w", err)
	}

	return enforcer, nil
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
