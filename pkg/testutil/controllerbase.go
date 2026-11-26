package testutil

import (
	"github.com/gin-gonic/gin"
)

// ControllerBase is a struct that provides a base for controllers.
type ControllerBase struct {
	*Base

	Router *gin.Engine
}

// ForController creates a new instance of ControllerBase with a Base.
func (b *Base) ForController() *ControllerBase {
	return &ControllerBase{
		Base:   b,
		Router: nil,
	}
}

// SetupRouter sets up the router for the controller, applying middleware before routes.
func (b *ControllerBase) SetupRouter(controller Controller, middleware ...gin.HandlerFunc) {
	b.Router = setupRouter(controller, middleware...)
}

// Controller is an interface that defines the methods for a controller.
type Controller interface {
	RoutesInfo() gin.RoutesInfo
}

func setupRouter(controller Controller, middleware ...gin.HandlerFunc) *gin.Engine {
	router := gin.Default()
	router.Use(middleware...)

	for _, route := range controller.RoutesInfo() {
		router.Handle(route.Method, route.Path, route.HandlerFunc)
	}

	return router
}
