// Package connection provides the HTTP controller for managing connections.
package connection

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/ginutil"
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
// @Description  Retrieve a list of connections in a namespace.
// @Accept  json
// @Produce json
// @Param namespace path string true "Namespace"
// @Param limit query int false "Maximum number of connections to return"
// @Param continue query string false "Token to continue listing connections"
// @Success 200 {object} v1.ListResponse[v1.Connection]
// @Failure 500 {object} map[string]any
// @Router /api/v1/namespaces/{namespace}/connections [get].
func (c *Controller) List(ctx *gin.Context) {
	now := c.clock.Now()
	namespace := ctx.Param("namespace")

	limit, err := ginutil.ParseInt64(ctx, "limit", 0)
	if err != nil {
		ginutil.HandleValidationError(ctx, "limit", ctx.Query("limit"), err, false)

		return
	}

	continueToken := ctx.Query("continue")

	response, err := c.adminUsecase.ListConnections(ctx.Request.Context(), namespace, &model.ListOptions{
		Limit:          limit,
		Continue:       continueToken,
		IncludeDeleted: false,
	})
	if err != nil {
		c.logger.Error("failed to list connections", "error", err.Error())
		ginutil.InternalServerError(ctx, err, "An error occurred while listing connections.")

		return
	}

	connectionResponse := v1.NewConnectionListResponse(
		lo.Map(response.Items, func(connection *agentmodel.Connection, _ int) v1.Connection {
			return v1.Connection{
				ID:                 connection.UID,
				InstanceUID:        connection.InstanceUID,
				Namespace:          connection.Namespace,
				Type:               connection.Type.String(),
				Alive:              connection.IsAlive(now),
				LastCommunicatedAt: v1.NewTime(connection.LastCommunicatedAt),
			}
		}),
		v1.ListMeta{
			RemainingItemCount: response.RemainingItemCount,
			Continue:           response.Continue,
		},
	)

	ctx.JSON(http.StatusOK, connectionResponse)
}
