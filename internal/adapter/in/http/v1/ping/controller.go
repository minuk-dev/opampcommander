package ping

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	logger *slog.Logger
}

type Option func(*Controller)

func NewController(options ...Option) *Controller {
	controller := &Controller{
		logger: slog.Default(),
	}

	for _, option := range options {
		option(controller)
	}

	return controller
}

func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      "GET",
			Path:        "/v1/ping",
			Handler:     "http.v1.ping.Handle",
			HandlerFunc: c.Handle,
		},
	}
}

func (c *Controller) Handle(ctx *gin.Context) {
	c.logger.Info("handling request")
	ctx.JSON(http.StatusOK, gin.H{"message": "pong"})
}
