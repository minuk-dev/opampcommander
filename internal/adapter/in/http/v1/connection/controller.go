// Package connection provides the HTTP controller for managing connections.
package connection

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/samber/lo"
	k8sclock "k8s.io/utils/clock"

	connectionv1 "github.com/minuk-dev/opampcommander/api/v1/connection"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

// Controller is a struct that handles HTTP requests related to connections.
type Controller struct {
	logger *slog.Logger
	clock  clock.Clock

	// usecases
	connectionUsecase port.ConnectionUsecase
}

// NewController creates a new instance of the Controller struct.
func NewController(connectionUsecase port.ConnectionUsecase) *Controller {
	controller := &Controller{
		logger: slog.Default(),
		clock:  k8sclock.RealClock{},

		connectionUsecase: connectionUsecase,
	}

	return controller
}

// RoutesInfo returns the routes information for the controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      "GET",
			Path:        "/v1/connections",
			Handler:     "http.v1.connection.List",
			HandlerFunc: c.List,
		},
		{
			Method:      "GET",
			Path:        "/v1/connections/:id",
			Handler:     "http.v1.connection.Get",
			HandlerFunc: c.Get,
		},
	}
}

// List handles the request to list all connections.
func (c *Controller) List(ctx *gin.Context) {
	now := c.clock.Now()
	connections := c.connectionUsecase.ListConnections()
	connectionResponse := lo.Map(connections, func(connection *model.Connection, _ int) *connectionv1.Connection {
		return &connectionv1.Connection{
			ID:                 connection.ID,
			InstanceUID:        connection.ID,
			Alive:              connection.IsAlive(now),
			LastCommunicatedAt: connection.LastCommunicatedAt(),
		}
	})

	ctx.JSON(http.StatusOK, connectionResponse)
}

// Get handles the request to get a connection by ID.
func (c *Controller) Get(ctx *gin.Context) {
	connectionID := ctx.GetString("id")

	connectionUUID, err := uuid.Parse(connectionID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, err)

		return
	}

	connection, err := c.connectionUsecase.GetConnection(connectionUUID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, err)

		return
	}

	ctx.JSON(http.StatusOK, connection)
}
