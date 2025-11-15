// Package infrastructure provides infrastructure components module for the API server.
package infrastructure

import (
	"context"
	"net"

	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/auth/basic"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/auth/github"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agent"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agentgroup"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/connection"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/opamp"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/ping"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/server"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/version"
	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
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
			server.NewController, helper.AsController(Identity[*server.Controller]),
			github.NewController, helper.AsController(Identity[*github.Controller]),
			basic.NewController, helper.AsController(Identity[*basic.Controller]),
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
			fx.Annotate(mongodb.NewAgentRepository, fx.As(new(port.AgentPersistencePort))),
			fx.Annotate(mongodb.NewAgentGroupRepository, fx.As(new(port.AgentGroupPersistencePort))),
			fx.Annotate(mongodb.NewServerAdapter, fx.As(new(port.ServerPersistencePort))),
		),
	)
}

// provideMessagingComponents provides messaging-related components (Kafka/in-memory).
func provideMessagingComponents() fx.Option {
	return fx.Options(
		// Provide the event hub adapter
		fx.Provide(
			fx.Annotate(
				NewEventhubAdapter,
				fx.As(new(port.ServerEventSenderPort)),
				fx.As(new(port.ServerEventReceiverPort)),
			),
		),
	)
}

// Identity is a generic function that returns the input value.
// It is a helper function to generate a function that returns the input value.
// It is used to provide a function as a interface.
func Identity[T any](a T) T {
	return a
}
