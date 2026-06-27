// Package connection provides the HTTP controller for managing connections.
package connection

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/ginutil"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

// Controller is a struct that handles HTTP requests related to connections.
type Controller struct {
	logger *slog.Logger
	clock  clock.Clock

	// usecases
	adminUsecase applicationport.AdminUsecase
}

// NewController creates a new instance of the Controller struct.
func NewController(
	logger *slog.Logger,
	adminUsecase applicationport.AdminUsecase,
) *Controller {
	controller := &Controller{
		logger:       logger,
		clock:        clock.NewRealClock(),
		adminUsecase: adminUsecase,
	}

	return controller
}

// RoutesInfo returns the routes information for the controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      "GET",
			Path:        "/api/v1/namespaces/:namespace/connections",
			Handler:     "http.v1.connection.List",
			HandlerFunc: c.List,
		},
	}
}

// List handles the request to list all connections in a namespace.
//
// @Summary List Connections
// @Tags connection
// @Description Retrieve the live connections in a namespace. NOTE: connections are
// @Description WebSockets bound to a single server, so this returns only the connections
// @Description held by the server instance handling the request — in a multi-server (HA)
// @Description deployment it is a node-local view, not a cluster-wide list. For a global
// @Description view of which agents are connected (and to which server), use the agents
// @Description API (each agent reports its Connected status and last-reported server).
// @Accept  json
// @Produce json
// @Param namespace path string true "Namespace"
// @Param limit query int false "Maximum number of connections to return"
// @Param continue query string false "Token to continue listing connections"
// @Success 200 {object} v1.ListResponse[v1.Connection]
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/connections [get].
func (c *Controller) List(ctx *gin.Context) {
	namespace := ctx.Param("namespace")

	limit, err := ginutil.ParseInt64(ctx, "limit", 0)
	if err != nil {
		ginutil.HandleValidationError(ctx, "limit", ctx.Query("limit"), err, false)

		return
	}

	continueToken := ctx.Query("continue")

	var connectionResponse *v1.ListResponse[v1.Connection]

	connectionResponse, err = c.adminUsecase.ListConnections(
		ctx.Request.Context(), namespace, &applicationport.ListOptions{
			Limit:          limit,
			Continue:       continueToken,
			IncludeDeleted: false,
		})
	if err != nil {
		c.logger.Error("failed to list connections", "error", err.Error())
		ginutil.InternalServerError(ctx, err, "An error occurred while listing connections.")

		return
	}

	ctx.JSON(http.StatusOK, connectionResponse)
}
