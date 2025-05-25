package app

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
)

// NewEngine creates a new Gin engine and registers the provided controllers' routes.
func NewEngine(
	controllers []Controller,
	logger *slog.Logger,
) *gin.Engine {
	engine := gin.New()
	engine.Use(sloggin.New(logger))
	engine.Use(gin.Recovery())

	for _, controller := range controllers {
		routeInfo := controller.RoutesInfo()
		for _, route := range routeInfo {
			engine.Handle(route.Method, route.Path, route.HandlerFunc)
		}
	}

	return engine
}
