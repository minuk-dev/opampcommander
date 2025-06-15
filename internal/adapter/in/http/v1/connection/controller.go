// Package connection provides the HTTP controller for managing connections.
package connection

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	k8sclock "k8s.io/utils/clock"

	connectionv1 "github.com/minuk-dev/opampcommander/api/v1/connection"
	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
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
		logger: slog.Default(),
		clock:  k8sclock.RealClock{},

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
// @Success 200 {array} connectionv1.Connection
// @Failure 500 {object} gin.H
// @Router /api/v1/connections [get].
func (c *Controller) List(ctx *gin.Context) {
	now := c.clock.Now()

	connections, err := c.adminUsecase.ListConnections(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, err)

		return
	}

	connectionResponse := lo.Map(connections, func(connection *model.Connection, _ int) *connectionv1.Connection {
		return &connectionv1.Connection{
			ID:                 connection.UID,
			InstanceUID:        connection.InstanceUID,
			Alive:              connection.IsAlive(now),
			LastCommunicatedAt: connection.LastCommunicatedAt,
		}
	})

	ctx.JSON(http.StatusOK, connectionResponse)
}
