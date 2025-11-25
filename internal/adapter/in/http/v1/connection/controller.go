// Package connection provides the HTTP controller for managing connections.
package connection

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	k8sclock "k8s.io/utils/clock"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	connectionv1 "github.com/minuk-dev/opampcommander/api/v1/connection"
	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
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
func NewController(adminUsecase applicationport.AdminUsecase) *Controller {
	controller := &Controller{
		logger:       slog.Default(),
		clock:        k8sclock.RealClock{},
		adminUsecase: adminUsecase,
	}

	return controller
}

// RoutesInfo returns the routes information for the controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      "GET",
			Path:        "/api/v1/connections",
			Handler:     "http.v1.connection.List",
			HandlerFunc: c.List,
		},
	}
}

// List handles the request to list all connections.
//
// @Summary List Connections
// @Tags connection
// @Description  Retrieve a list of all connections.
// @Accept  json
// @Produce json
// @Success 200 {array} Connection
// @Failure 500 {object} map[string]any
// @Router /api/v1/connections [get].
func (c *Controller) List(ctx *gin.Context) {
	now := c.clock.Now()

	limit, err := ginutil.ParseInt64(ctx, "limit", 0)
	if err != nil {
		ginutil.HandleValidationError(ctx, "limit", ctx.Query("limit"), err, false)
		return
	}

	continueToken := ctx.Query("continue")

	response, err := c.adminUsecase.ListConnections(ctx.Request.Context(), &model.ListOptions{
		Limit:    limit,
		Continue: continueToken,
	})
	if err != nil {
		c.logger.Error("failed to list connections", "error", err.Error())
		ginutil.InternalServerError(ctx, err, "An error occurred while listing connections.")
		return
	}

	connectionResponse := connectionv1.NewListResponse(
		lo.Map(response.Items, func(connection *model.Connection, _ int) connectionv1.Connection {
			return connectionv1.Connection{
				ID:                 connection.UID,
				InstanceUID:        connection.InstanceUID,
				Alive:              connection.IsAlive(now),
				LastCommunicatedAt: connection.LastCommunicatedAt,
			}
		}),
		v1.ListMeta{
			RemainingItemCount: response.RemainingItemCount,
			Continue:           response.Continue,
		},
	)

	ctx.JSON(http.StatusOK, connectionResponse)
}
