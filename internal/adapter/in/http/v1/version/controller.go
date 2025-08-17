// Package version provides server version information.
package version

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/minuk-dev/opampcommander/pkg/version"
)

type Controller struct {
	logger *slog.Logger
}

func NewController(logger *slog.Logger) *Controller {
	return &Controller{
		logger: logger,
	}
}

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
func (c *Controller) GetVersion(ctx *gin.Context) {
	c.logger.Debug("GetVersion called")
	versionInfo := version.Get()
	ctx.JSON(http.StatusOK, versionInfo)
}
