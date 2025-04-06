package app

import "github.com/gin-gonic/gin"

func NewEngine(controllers []Controller) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())
	// engine.Use(sloggin.New(logger))

	for _, controller := range controllers {
		routeInfo := controller.RoutesInfo()
		for _, route := range routeInfo {
			engine.Handle(route.Method, route.Path, route.HandlerFunc)
		}
	}

	return engine
}
