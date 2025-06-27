package apiserver

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/minuk-dev/opampcommander/internal/security"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/docs"
)

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
