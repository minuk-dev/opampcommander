// Package infrastructure provides infrastructure components module for the API server.
package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	casbinModel "github.com/casbin/casbin/v2/model"
	mongodbadapter "github.com/casbin/mongodb-adapter/v4"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/auth/basic"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/auth/github"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agent"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agentgroup"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agentpackage"
	agentremoteconfigcontroller "github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agentremoteconfig"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/certificate"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/connection"
	namespacecontroller "github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/namespace"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/opamp"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/ping"
	rbaccontroller "github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/rbac"
	rolecontroller "github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/role"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/server"
	usercontroller "github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/user"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/version"
	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb"
	casbinEnforcer "github.com/minuk-dev/opampcommander/internal/adapter/out/rbac/casbin"
	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/module/helper"
)

// New creates a new module for infrastructure components.
// This includes HTTP server, database, messaging, and WebSocket registry.
func New() fx.Option {
	return fx.Module(
		"infrastructure",
		// HTTP Server & Controllers
		provideHTTPComponents(),

		// Database (MongoDB)
		provideDatabaseComponents(),

		// Messaging (NATS or in-memory)
		provideMessagingComponents(),

		// RBAC (Casbin enforcer)
		provideRBACComponents(),

		// Default namespace initialization
		fx.Invoke(registerDefaultNamespaceHook),
	)
}

// provideHTTPComponents provides HTTP server and controller components.
func provideHTTPComponents() fx.Option {
	return fx.Options(
		// HTTP Server & Engine
		fx.Provide(
			NewHTTPServer,
			fx.Annotate(NewEngine, fx.ParamTags(`group:"controllers"`)),
		),
		// Controllers
		fx.Provide(
			ping.NewController, helper.AsController(Identity[*ping.Controller]),
			opamp.NewController, helper.AsController(Identity[*opamp.Controller]),
			version.NewController, helper.AsController(Identity[*version.Controller]),
			connection.NewController, helper.AsController(Identity[*connection.Controller]),
			agent.NewController, helper.AsController(Identity[*agent.Controller]),
			agentgroup.NewController, helper.AsController(Identity[*agentgroup.Controller]),
			agentpackage.NewController, helper.AsController(Identity[*agentpackage.Controller]),
			agentremoteconfigcontroller.NewController,
			helper.AsController(Identity[*agentremoteconfigcontroller.Controller]),
			namespacecontroller.NewController, helper.AsController(Identity[*namespacecontroller.Controller]),
			certificate.NewController, helper.AsController(Identity[*certificate.Controller]),
			server.NewController, helper.AsController(Identity[*server.Controller]),
			github.NewController, helper.AsController(Identity[*github.Controller]),
			basic.NewController, helper.AsController(Identity[*basic.Controller]),
			// RBAC controllers
			usercontroller.NewController, helper.AsController(Identity[*usercontroller.Controller]),
			rolecontroller.NewController, helper.AsController(Identity[*rolecontroller.Controller]),
			rbaccontroller.NewController, helper.AsController(Identity[*rbaccontroller.Controller]),
		),
		// OpAMP specific connection context
		fx.Provide(
			func(opampController *opamp.Controller) func(context.Context, net.Conn) context.Context {
				return opampController.ConnContext
			},
		),
	)
}

// provideDatabaseComponents provides database-related components.
func provideDatabaseComponents() fx.Option {
	return fx.Options(
		fx.Provide(
			NewMongoDBClient,
			NewMongoDatabase,
			helper.AsHealthIndicator(NewMongoDBHealthIndicator),
			fx.Annotate(mongodb.NewAgentRepository, fx.As(new(agentport.AgentPersistencePort))),
			fx.Annotate(mongodb.NewAgentGroupRepository, fx.As(new(agentport.AgentGroupPersistencePort))),
			fx.Annotate(mongodb.NewServerAdapter, fx.As(new(agentport.ServerPersistencePort))),
			fx.Annotate(mongodb.NewAgentPackageRepository, fx.As(new(agentport.AgentPackagePersistencePort))),
			fx.Annotate(mongodb.NewNamespaceRepository, fx.As(new(agentport.NamespacePersistencePort))),
			fx.Annotate(mongodb.NewAgentRemoteConfigRepository, fx.As(new(agentport.AgentRemoteConfigPersistencePort))),
			fx.Annotate(mongodb.NewCertificateRepository, fx.As(new(agentport.CertificatePersistencePort))),
			// RBAC MongoDB adapters
			fx.Annotate(mongodb.NewUserRepository, fx.As(new(userport.UserPersistencePort))),
			fx.Annotate(mongodb.NewRoleRepository, fx.As(new(userport.RolePersistencePort))),
			fx.Annotate(mongodb.NewPermissionRepository, fx.As(new(userport.PermissionPersistencePort))),
			fx.Annotate(mongodb.NewUserRoleRepository, fx.As(new(userport.UserRolePersistencePort))),
			fx.Annotate(mongodb.NewOrgRoleMappingRepository, fx.As(new(userport.OrgRoleMappingPersistencePort))),
		),
	)
}

// provideMessagingComponents provides messaging-related components (Kafka/in-memory).
func provideMessagingComponents() fx.Option {
	return fx.Options(
		// Provide the event hub adapter
		fx.Provide(newEventSenderAndReceiver),
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
