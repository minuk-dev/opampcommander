package primary

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/auth/basic"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/auth/github"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/agentgroup"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/agentpackage"
	agentremoteconfigcontroller "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/agentremoteconfig"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/certificate"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/connection"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/container"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/endpoint"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/endpointmetrics"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/host"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/namespace"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/opamp"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/ping"
	reconcilecontroller "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/reconcile"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/role"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/rolebinding"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/server"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/user"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/v1/version"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/docs"
	userport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/management/observability"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/security"
)

const (
	// DefaultHTTPReadTimeout is the default timeout for reading HTTP requests.
	// It should be set to a reasonable value to avoid security issues.
	DefaultHTTPReadTimeout = 30 * time.Second

	// DefaultHTTPIdleTimeout is how long a keep-alive connection stays open
	// between requests. It must be longer than the OpAMP HTTP client's poll
	// interval (default 30s) — otherwise the server can close an idle
	// keep-alive connection exactly as the client tries to reuse it, which
	// surfaces as `Post ... : EOF` on the agent. When unset, net/http falls
	// back to ReadTimeout, which produces that exact race.
	DefaultHTTPIdleTimeout = 120 * time.Second
)

var (
	//nolint:gochecknoglobals // Swagger global variable is initialized once to prevent race conditions
	swaggerOnce sync.Once
)

// NewHTTP provides the HTTP server, the Gin engine, and every controller registered into it.
func NewHTTP() fx.Option {
	return fx.Options(
		fx.Provide(
			// HTTP Server & Engine
			NewHTTPServer,
			fx.Annotate(NewEngine, fx.ParamTags(`group:"controllers"`)),

			// Controllers — added to the "controllers" group consumed by NewEngine.
			AsController(ping.NewController),
			AsController(version.NewController),
			AsController(connection.NewController),
			AsController(agent.NewController),
			AsController(agentgroup.NewController),
			AsController(agentpackage.NewController),
			AsController(agentremoteconfigcontroller.NewController),
			AsController(reconcilecontroller.NewController),
			AsController(endpoint.NewController),
			AsController(endpointmetrics.NewController),
			AsController(namespace.NewController),
			AsController(certificate.NewController),
			AsController(host.NewController),
			AsController(container.NewController),
			AsController(server.NewController),
			AsController(user.NewController),
			AsController(role.NewController),
			AsController(rolebinding.NewController),
			AsController(github.NewController),
			AsController(basic.NewController),

			// The OpAMP controller is also needed as its concrete type for the
			// connection context, so it is provided plainly and then added to the
			// group via a pass-through (fx.Self() can't be used here: ResultTags
			// would also tag the concrete output, hiding it from connContext).
			opamp.NewController,
			fx.Annotate(
				func(c *opamp.Controller) Controller { return c },
				fx.ResultTags(`group:"controllers"`),
			),

			// OpAMP specific connection context
			func(opampController *opamp.Controller) func(context.Context, net.Conn) context.Context {
				return opampController.ConnContext
			},
		),
	)
}

// NewHTTPServer creates a new HTTP server instance.
func NewHTTPServer(
	lifecycle fx.Lifecycle,
	engine *gin.Engine,
	settings *config.ServerSettings,
	logger *slog.Logger,
	connContext func(context.Context, net.Conn) context.Context,
) *http.Server {
	//exhaustruct:ignore
	srv := &http.Server{
		ReadTimeout: DefaultHTTPReadTimeout,
		IdleTimeout: DefaultHTTPIdleTimeout,
		Addr:        settings.Address,
		Handler:     engine,
		ConnContext: connContext,
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			//exhaustruct:ignore
			listenConfig := &net.ListenConfig{}

			listener, err := listenConfig.Listen(ctx, "tcp", srv.Addr)
			if err != nil {
				return fmt.Errorf("failed to listen: %w", err)
			}

			logger.Info("HTTP server listening",
				slog.String("addr", settings.Address),
			)

			go func() {
				err := srv.Serve(listener)
				if err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Error("HTTP server error",
						slog.String("error", err.Error()),
					)
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})

	return srv
}

// NewEngine creates a new Gin engine and registers the provided controllers' routes.
func NewEngine(
	controllers []Controller,
	securityService *security.Service,
	rbacUsecase userport.RBACUsecase,
	userUsecase userport.UserUsecase,
	settings *config.ServerSettings,
	observabilityService *observability.Service,
	logger *slog.Logger,
) *gin.Engine {
	engine := gin.New()
	engine.Use(sloggin.New(logger))
	engine.Use(gin.Recovery())
	engine.Use(security.NewAuthJWTMiddleware(securityService))
	engine.Use(security.NewAuthorizationMiddleware(
		rbacUsecase,
		userUsecase,
		settings.Security.AdminSettings.Email,
		logger,
	))
	engine.Use(observabilityService.Middleware())
	// swagger
	engine.GET("/swagger/*any", ginswagger.WrapHandler(swaggerfiles.Handler))
	engine.GET("/docs", func(ctx *gin.Context) {
		ctx.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})

	// Initialize swagger info only once to avoid race conditions in tests
	swaggerOnce.Do(func() {
		docs.SwaggerInfo.BasePath = "/"
	})

	for _, controller := range controllers {
		routeInfo := controller.RoutesInfo()
		for _, route := range routeInfo {
			engine.Handle(route.Method, route.Path, route.HandlerFunc)
		}
	}

	return engine
}

// Controller is an interface that defines the methods for handling HTTP requests.
type Controller interface {
	RoutesInfo() gin.RoutesInfo
}

// AsController annotates a controller constructor so its result is provided as the
// Controller interface into the "controllers" group consumed by NewEngine.
func AsController(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(Controller)),
		fx.ResultTags(`group:"controllers"`),
	)
}
