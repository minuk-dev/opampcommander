package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	clientv3 "go.etcd.io/etcd/client/v3"
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

const (
	DefaultHTTPReadTimeout = 30 * time.Second
)

var (
	ErrInfrastructureInitFailed = errors.New("infrastructure init failed")
	ErrInAdapterInitFailed      = errors.New("in adapter init failed")
	ErrOutAdapterInitFailed     = errors.New("out adapter init failed")
	ErrDomainInitFailed         = errors.New("domain init failed")
	ErrApplicationInitFailed    = errors.New("application init failed")
)

type ServerSettings struct {
	EtcdHosts []string
}

type Server struct {
	settings   ServerSettings
	Engine     *gin.Engine
	httpServer *http.Server
}

func NewHTTPServer(lc fx.Lifecycle, engine *gin.Engine) *http.Server {
	srv := &http.Server{
		ReadTimeout: DefaultHTTPReadTimeout,
		Addr:        ":8080",
		Handler:     engine,
	}

	//srv.ConnContext = opampController.ConnContext
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			fmt.Println("Starting HTTP server at", srv.Addr)
			go srv.Serve(ln)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}

func NewEngine(controllers []Controller) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())
	//engine.Use(sloggin.New(logger))

	for _, controller := range controllers {
		routeInfo := controller.RoutesInfo()
		for _, route := range routeInfo {
			engine.Handle(route.Method, route.Path, route.HandlerFunc)
		}
	}

	return engine
}

func NewLogger() *slog.Logger {
	logger := slog.Default()
	return logger
}

func NewEtcdClient(settings *ServerSettings) (*clientv3.Client, error) {
	etcdConfig := clientv3.Config{
		Endpoints: settings.EtcdHosts,
	}

	etcdClient, err := clientv3.New(etcdConfig)
	if err != nil {
		return nil, fmt.Errorf("etcd client init failed: %w", err)
	}

	return etcdClient, nil
}

func AsController(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(Controller)),
		fx.ResultTags(`group:"controllers"`),
	)
}

func NewServer(settings ServerSettings) *Server {
	app := fx.New(
		fx.Provide(
			NewHTTPServer,
			fx.Annotate(
				NewEngine,
				fx.ParamTags(`group:"controllers"`),
			),
		),
		fx.Provide(NewLogger),
		fx.Provide(NewEtcdClient),

		// controllers
		fx.Provide(
			AsController(ping.NewController),
			AsController(opamp.NewController),
			AsController(connection.NewController),
			AsController(agent.NewController),
		),
		fx.Provide(fx.Annotate(etcd.NewAgentEtcdAdapter, fx.As(new(domainport.AgentPersistencePort)))),
		fx.Provide(fx.Annotate(etcd.NewCommandEtcdAdapter, fx.As(new(domainport.CommandPersistencePort)))),
		fx.Provide(fx.Annotate(opampApplicationService.New, fx.As(new(port.OpAMPUsecase)))),
		fx.Provide(fx.Annotate(domainservice.NewCommandService, fx.As(new(domainport.CommandUsecase)))),
		fx.Provide(fx.Annotate(domainservice.NewConnectionManager, fx.As(new(domainport.ConnectionUsecase)))),
		fx.Provide(fx.Annotate(domainservice.NewAgentService, fx.As(new(domainport.AgentUsecase)))),
		fx.Provide(func() *ServerSettings {
			return &settings
		}),
		fx.Invoke(func(*http.Server) {}),
	)

	app.Run()
	server := &Server{
		settings: settings,
	}

	return server
}

func (s *Server) Run() error {
	err := s.httpServer.ListenAndServe()
	if err != nil {
		return fmt.Errorf("server run failed: %w", err)
	}

	return nil
}

type Controller interface {
	RoutesInfo() gin.RoutesInfo
}
