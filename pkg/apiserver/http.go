package apiserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/internal/security"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/docs"
)

const (
	// DefaultHTTPReadTimeout is the default timeout for reading HTTP requests.
	// It should be set to a reasonable value to avoid security issues.
	DefaultHTTPReadTimeout = 30 * time.Second
)

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
		Addr:        settings.Address,
		Handler:     engine,
		ConnContext: connContext,
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			listener, err := net.Listen("tcp", srv.Addr)
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
	logger *slog.Logger,
) *gin.Engine {
	engine := gin.New()
	engine.Use(sloggin.New(logger))
	engine.Use(gin.Recovery())
	engine.Use(security.NewAuthJWTMiddleware(securityService))
	engine.Use(otelgin.Middleware("opampcommander"))
	// swagger
	engine.GET("/swagger/*any", ginswagger.WrapHandler(swaggerfiles.Handler))

	docs.SwaggerInfo.BasePath = "/"

	for _, controller := range controllers {
		routeInfo := controller.RoutesInfo()
		for _, route := range routeInfo {
			engine.Handle(route.Method, route.Path, route.HandlerFunc)
		}
	}

	return engine
}
