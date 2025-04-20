package app

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/agent"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/connection"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/opamp"
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/ping"
	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/etcd"
	"github.com/minuk-dev/opampcommander/internal/application/port"
	opampApplicationService "github.com/minuk-dev/opampcommander/internal/application/service/opamp"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	domainservice "github.com/minuk-dev/opampcommander/internal/domain/service"
)

// ServerSettings is a struct that holds the server settings.
type ServerSettings struct {
	Addr      string
	EtcdHosts []string
}

// Server is a struct that represents the server application.
// It embeds the fx.App struct from the Uber Fx framework.
type Server struct {
	*fx.App

	settings ServerSettings
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
			AsController(ping.NewController),
			AsController(opamp.NewController),
			AsController(connection.NewController),
			AsController(agent.NewController),
		),
		// application
		fx.Provide(
			fx.Annotate(opampApplicationService.New, fx.As(new(port.OpAMPUsecase))),
		),
		// domain
		fx.Provide(
			fx.Annotate(domainservice.NewCommandService, fx.As(new(domainport.CommandUsecase))),
			fx.Annotate(domainservice.NewConnectionManager, fx.As(new(domainport.ConnectionUsecase))),
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

// Controller is an interface that defines the methods for handling HTTP requests.
type Controller interface {
	RoutesInfo() gin.RoutesInfo
}
