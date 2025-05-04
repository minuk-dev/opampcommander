// Package testutil provides utility functions and types for testing.
package testutil

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

// ControllerBase is a struct that provides a base for controllers.
type ControllerBase struct {
	Router *gin.Engine
	Logger *slog.Logger
}

// NewControllerBase creates a new instance of ControllerBase.
func NewControllerBase() *ControllerBase {
	return &ControllerBase{
		Router: nil,
		Logger: slog.Default(),
	}
}

// SetupRouter sets up the router for the controller.
func (b *ControllerBase) SetupRouter(controller Controller) {
	b.Router = setupRouter(controller)
}

// Controller is an interface that defines the methods for a controller.
type Controller interface {
	RoutesInfo() gin.RoutesInfo
}

func setupRouter(controller Controller) *gin.Engine {
	router := gin.Default()

	for _, route := range controller.RoutesInfo() {
		router.Handle(route.Method, route.Path, route.HandlerFunc)
	}

	return router
}
