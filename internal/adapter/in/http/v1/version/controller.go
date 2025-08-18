// Package version provides server version information.
package version

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/minuk-dev/opampcommander/pkg/version"
)

// Controller is a struct that implements the version controller.
type Controller struct {
	logger *slog.Logger
}

// NewController creates a new instance of the Controller struct with the provided settings.
func NewController(logger *slog.Logger) *Controller {
	return &Controller{
		logger: logger,
	}
}

// RoutesInfo returns the routes information for the version controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/version",
			Handler:     "version.v1.GetVersion",
			HandlerFunc: c.GetVersion,
		},
	}
}

// GetVersion handles the request to get the server version.
//
// @Summary Get Server Version
// @Tags version
// @Description  Retrieve the server version information.
// @Accept  json
// @Success 200 {object} VersionInfo
// @Router /api/v1/version [get].
func (c *Controller) GetVersion(ctx *gin.Context) {
	c.logger.Debug("GetVersion called")

	versionInfo := version.Get()
	ctx.JSON(http.StatusOK, versionInfo)
}
