package connection

import (
	"log/slog"

	"github.com/gin-gonic/gin"

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

func (c *Controller) List(*gin.Context) {
}

func (c *Controller) Get(*gin.Context) {
}
