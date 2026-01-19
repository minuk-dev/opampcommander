// Package ping provides the ping controller for the HTTP API.
// It handles the ping request and returns a JSON response with a "pong" message.
// It is helpful for testing the server's availability and responsiveness.
package ping

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

// Controller is a struct that implements the ping controller.
type Controller struct {
	logger *slog.Logger
}

// NewController creates a new instance of the ping controller.
func NewController(logger *slog.Logger) *Controller {
	controller := &Controller{
		logger: logger,
	}

	return controller
}

// RoutesInfo returns the routes information for the ping controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      "GET",
			Path:        "/api/v1/ping",
			Handler:     "http.v1.ping.Handle",
			HandlerFunc: c.Handle,
		},
	}
}

// Handle handles the ping request.
//
// @Summary  Ping
// @Tags ping
// @Description Ping the server to check if it is alive.
// @Accept json
// @Produce json
// @Success 200 {object} PingResponse
// @Failure 500 {object} map[string]any
// @Router /api/v1/ping [get].
func (c *Controller) Handle(ctx *gin.Context) {
	c.logger.Info("handling request")
	ctx.JSON(http.StatusOK, v1.PingResponse{
		Message: "pong",
	})
}
