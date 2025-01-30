package connection

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/minuk-dev/minuk-apiserver/internal/domain/port"
)

type Controller struct {
	logger *slog.Logger

	// usecases
	connectionUsecase Usecase
}

type Usecase interface {
	port.GetConnectionUsecase
	port.ListConnectionIDsUsecase
}

func NewController(options ...Option) *Controller {
	controller := &Controller{
		logger: slog.Default(),

		connectionUsecase: nil,
	}

	for _, option := range options {
		option(controller)
	}

	err := controller.Validate()
	if err != nil {
		controller.logger.Error("controller validation failed", "error", err.Error())

		return nil
	}

	return controller
}

func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      "GET",
			Path:        "/connections",
			Handler:     "http.v1.connection.List",
			HandlerFunc: c.List,
		},
		{
			Method:      "GET",
			Path:        "/connections/:id",
			Handler:     "http.v1.connection.Get",
			HandlerFunc: c.Get,
		},
	}
}

func (c *Controller) Validate() error {
	if c.connectionUsecase == nil {
		return &UsecaseNotProvidedError{Usecase: "connectionUsecase"}
	}

	return nil
}

func (c *Controller) List(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, c.connectionUsecase.ListConnectionIDs())
}

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
