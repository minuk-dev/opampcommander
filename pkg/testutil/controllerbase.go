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
