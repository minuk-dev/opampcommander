package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

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
)

const (
	// DefaultServerStartTimeout = 30 * time.Second.
	DefaultServerStartTimeout = 30 * time.Second

	// DefaultServerStopTimeout is the default timeout for stopping the server.
	DefaultServerStopTimeout = 30 * time.Second
)

// ServerSettings is a struct that holds the server settings.
type ServerSettings struct {
	Addr      string
	EtcdHosts []string
	LogLevel  slog.Level
	LogFormat LogFormat
}

// Server is a struct that represents the server application.
// It embeds the fx.App struct from the Uber Fx framework.
type Server struct {
	*fx.App

	settings ServerSettings
}

// Run starts the server and blocks until the context is done.
func (s *Server) Run(ctx context.Context) error {
	startCtx, startCancel := context.WithTimeout(ctx, DefaultServerStartTimeout)
	defer startCancel()

	err := s.Start(startCtx)
	if err != nil {
		return fmt.Errorf("failed to start the server: %w", err)
	}

	<-ctx.Done()

	// To gracefully shutdown, it needs stopCtx.
	stopCtx, stopCancel := context.WithTimeout(NoInheritContext(ctx), DefaultServerStopTimeout)
	defer stopCancel()

	err = s.Stop(stopCtx)
	if err != nil {
		return fmt.Errorf("failed to stop the server: %w", err)
	}

	return nil
}

// NoInheritContext provides a non-inherit context.
// It's a marker function for code readers.
// It's from https://github.com/kkHAIKE/contextcheck?tab=readme-ov-file#need-break-ctx-inheritance
func NoInheritContext(_ context.Context) context.Context {
	return context.Background()
}

// NewServer creates a new instance of the Server struct.
func NewServer(settings ServerSettings) *Server {
	app := fx.New(
		// base
		fx.Provide(
			NewHTTPServer,
			fx.Annotate(NewEngine, fx.ParamTags(`group:"controllers"`)),
			NewLogger,
		),
		// controllers
		fx.Provide(
			ping.NewController, AsController(Identity[*ping.Controller]),
			opamp.NewController, AsController(Identity[*opamp.Controller]),
			connection.NewController, AsController(Identity[*connection.Controller]),
			agent.NewController, AsController(Identity[*agent.Controller]),
			command.NewController, AsController(Identity[*command.Controller]),
		),
		// opamp spec
		fx.Provide(
			func(opampController *opamp.Controller) func(context.Context, net.Conn) context.Context {
				return opampController.ConnContext
			},
		),
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
		fx.Provide(
			fx.Annotate(NewExecutor, fx.ParamTags("", `group:"runners"`)),
		),
		// domain
		fx.Provide(
			fx.Annotate(domainservice.NewCommandService, fx.As(new(domainport.CommandUsecase))),
			fx.Annotate(domainservice.NewConnectionService, fx.As(new(domainport.ConnectionUsecase))),
			fx.Annotate(domainservice.NewAgentService, fx.As(new(domainport.AgentUsecase))),
		),
		// database
		fx.Provide(
			NewEtcdClient,
			fx.Annotate(etcd.NewAgentEtcdAdapter, fx.As(new(domainport.AgentPersistencePort))),
			fx.Annotate(etcd.NewCommandEtcdAdapter, fx.As(new(domainport.CommandPersistencePort))),
		),
		// config
		fx.Provide(func() *ServerSettings {
			return &settings
		}),
		// init
		fx.Invoke(func(*http.Server) {}),
		fx.Invoke(func(*Executor) {}),
	)

	server := &Server{
		App:      app,
		settings: settings,
	}

	return server
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

// Controller is an interface that defines the methods for handling HTTP requests.
type Controller interface {
	RoutesInfo() gin.RoutesInfo
}

// Identity is a generic function that returns the input value.
// It is a helper function to generate a function that returns the input value.
// It is used to provide a function as a interface.
func Identity[T any](a T) T {
	return a
}
