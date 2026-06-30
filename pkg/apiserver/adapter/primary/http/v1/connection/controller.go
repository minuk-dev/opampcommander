// Package connection provides the HTTP controller for managing connections.
package connection

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/usecase"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/ginutil"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

// scopeCluster is the value of the "scope" query parameter that requests a cluster-wide
// connection listing instead of the default node-local one.
const scopeCluster = "cluster"

// Controller is a struct that handles HTTP requests related to connections.
type Controller struct {
	logger *slog.Logger
	clock  clock.Clock

	// usecases
	adminUsecase usecase.AdminUsecase
}

// NewController creates a new instance of the Controller struct.
func NewController(
	logger *slog.Logger,
	adminUsecase usecase.AdminUsecase,
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
// @Description Retrieve connections in a namespace. By default (scope=local) this returns
// @Description only the connections held by the server instance handling the request —
// @Description connections are WebSockets bound to a single node, so in a multi-server (HA)
// @Description deployment the default is a node-local view. Pass scope=cluster to get a
// @Description cluster-wide view aggregated from each server's periodic snapshot; those
// @Description items include the owning serverId. For an always-current view of agent
// @Description connectivity, the agents API remains authoritative.
// @Accept  json
// @Produce json
// @Param namespace path string true "Namespace"
// @Param scope query string false "Scope of the listing: 'local' (default) or 'cluster'"
// @Param serverId query string false "Restrict to one server's connections (implies cluster scope)"
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

	options := &applicationport.ListOptions{
		Limit:          limit,
		Continue:       ctx.Query("continue"),
		IncludeDeleted: false,
	}

	serverID := ctx.Query("serverId")

	var connectionResponse *v1.ListResponse[v1.Connection]

	// A specific serverId is inherently a cross-node lookup, so it implies cluster scope.
	if ctx.Query("scope") == scopeCluster || serverID != "" {
		connectionResponse, err = c.adminUsecase.ListClusterConnections(
			ctx.Request.Context(), namespace, serverID, options)
	} else {
		connectionResponse, err = c.adminUsecase.ListConnections(ctx.Request.Context(), namespace, options)
	}

	if err != nil {
		c.logger.Error("failed to list connections", "error", err.Error())
		ginutil.InternalServerError(ctx, err, "An error occurred while listing connections.")

		return
	}

	ctx.JSON(http.StatusOK, connectionResponse)
}
