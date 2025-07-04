package apiserver

import (
	"context"
	"net"

	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/auth/basic"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/auth/github"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agent"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/command"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/connection"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/opamp"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/ping"
	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/etcd"
	"github.com/minuk-dev/opampcommander/internal/application/port"
	adminApplicationService "github.com/minuk-dev/opampcommander/internal/application/service/admin"
	agentApplicationService "github.com/minuk-dev/opampcommander/internal/application/service/agent"
	commandApplicationService "github.com/minuk-dev/opampcommander/internal/application/service/command"
	opampApplicationService "github.com/minuk-dev/opampcommander/internal/application/service/opamp"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	domainservice "github.com/minuk-dev/opampcommander/internal/domain/service"
	"github.com/minuk-dev/opampcommander/internal/helper"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

// NewConfigModule creates a new module for configuration.
//
//nolint:ireturn
func NewConfigModule(settings *config.ServerSettings) fx.Option {
	return fx.Module(
		"config",
		// config
		fx.Provide(ValueFunc(settings)),
		fx.Provide(PointerFunc(settings.DatabaseSettings)),
		fx.Provide(PointerFunc(settings.AuthSettings)),
		fx.Provide(PointerFunc(settings.ObservabilitySettings)),
	)
}

// NewInPortModule creates a new module for controllers.
//
//nolint:ireturn
func NewInPortModule() fx.Option {
	return fx.Module(
		"inport",
		// base
		fx.Provide(
			NewHTTPServer,
			fx.Annotate(NewEngine, fx.ParamTags(`group:"controllers"`)),
		),
		fx.Provide(
			ping.NewController, AsController(Identity[*ping.Controller]),
			opamp.NewController, AsController(Identity[*opamp.Controller]),
			connection.NewController, AsController(Identity[*connection.Controller]),
			agent.NewController, AsController(Identity[*agent.Controller]),
			command.NewController, AsController(Identity[*command.Controller]),
			github.NewController, AsController(Identity[*github.Controller]),
			basic.NewController, AsController(Identity[*basic.Controller]),
		),
		// opamp specific spec for connection context
		fx.Provide(
			func(opampController *opamp.Controller) func(context.Context, net.Conn) context.Context {
				return opampController.ConnContext
			},
		),
	)
}

// NewApplicationServiceModule creates a new module for application services.
//
//nolint:ireturn
func NewApplicationServiceModule() fx.Option {
	return fx.Module(
		"application",
		// application
		fx.Provide(
			opampApplicationService.New,
			fx.Annotate(Identity[*opampApplicationService.Service], fx.As(new(port.OpAMPUsecase))),
			AsRunner(Identity[*opampApplicationService.Service]), // for background processing

			adminApplicationService.New,
			fx.Annotate(Identity[*adminApplicationService.Service], fx.As(new(port.AdminUsecase))),

			commandApplicationService.New,
			fx.Annotate(Identity[*commandApplicationService.Service], fx.As(new(port.CommandLookUpUsecase))),

			agentApplicationService.New,
			fx.Annotate(Identity[*agentApplicationService.Service], fx.As(new(port.AgentManageUsecase))),
		),
	)
}

// NewDomainServiceModule creates a new module for domain services.
//
//nolint:ireturn
func NewDomainServiceModule() fx.Option {
	return fx.Module(
		"domain",
		fx.Provide(
			fx.Annotate(domainservice.NewCommandService, fx.As(new(domainport.CommandUsecase))),
			fx.Annotate(domainservice.NewConnectionService, fx.As(new(domainport.ConnectionUsecase))),
			fx.Annotate(domainservice.NewAgentService, fx.As(new(domainport.AgentUsecase))),
		),
	)
}

// NewOutPortModule creates a new module for output adapters.
//
//nolint:ireturn
func NewOutPortModule() fx.Option {
	return fx.Module(
		"outport",
		fx.Provide(
			NewEtcdClient,
			fx.Annotate(etcd.NewAgentEtcdAdapter, fx.As(new(domainport.AgentPersistencePort))),
			fx.Annotate(etcd.NewCommandEtcdAdapter, fx.As(new(domainport.CommandPersistencePort))),
		),
	)
}

// AsController is a helper function to annotate a function as a controller.
func AsController(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(Controller)),
		fx.ResultTags(`group:"controllers"`),
	)
}

// AsRunner is a helper function to annotate a function as a runner.
func AsRunner(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(helper.Runner)),
		fx.ResultTags(`group:"runners"`),
	)
}

// NoInheritContext provides a non-inherit context.
// It's a marker function for code readers.
// It's from https://github.com/kkHAIKE/contextcheck?tab=readme-ov-file#need-break-ctx-inheritance
func NoInheritContext(_ context.Context) context.Context {
	return context.Background()
}

// Identity is a generic function that returns the input value.
// It is a helper function to generate a function that returns the input value.
// It is used to provide a function as a interface.
func Identity[T any](a T) T {
	return a
}

// PointerFunc is a generic function that returns a function that returns a pointer to the input value.
// It is a helper function to generate a function that returns a pointer to the input value.
// It is used to provide a function as a interface.
func PointerFunc[T any](a T) func() *T {
	return func() *T {
		return &a
	}
}

// ValueFunc is a generic function that returns a function that returns the input value.
// It is a helper function to generate a function that returns the input value.
func ValueFunc[T any](a T) func() T {
	return func() T {
		return a
	}
}
