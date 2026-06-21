// Package server provides the HTTP controller for managing servers.
package server

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/ginutil"
)

// Controller is a struct that handles HTTP requests related to servers.
type Controller struct {
	logger *slog.Logger

	// usecases
	serverUsecase applicationport.ServerManageUsecase
}

// NewController creates a new instance of the Controller struct.
func NewController(
	logger *slog.Logger,
	serverUsecase applicationport.ServerManageUsecase,
) *Controller {
	return &Controller{
		logger:        logger,
		serverUsecase: serverUsecase,
	}
}

// RoutesInfo returns the routes information for the controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      "GET",
			Path:        "/api/v1/servers",
			Handler:     "http.v1.server.List",
			HandlerFunc: c.List,
		},
	}
}

// List handles the request to list all alive servers.
//
// @Summary List Servers
// @Tags server
// @Description  Retrieve a list of all alive servers.
// @Accept  json
// @Produce json
// @Success 200 {object} v1.ListResponse[v1.Server]
// @Failure 500 {object} map[string]any
// @Router /api/v1/servers [get].
func (c *Controller) List(ctx *gin.Context) {
	var serverResponse *v1.ListResponse[v1.Server]

	serverResponse, err := c.serverUsecase.ListServers(ctx.Request.Context())
	if err != nil {
		c.logger.Error("failed to list servers", "error", err.Error())
		ginutil.InternalServerError(ctx, err, "An error occurred while listing servers.")

		return
	}

	ctx.JSON(http.StatusOK, serverResponse)
}
