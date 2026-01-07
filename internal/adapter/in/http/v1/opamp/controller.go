// Package opamp provides the implementation of the OPAMP protocol.
package opamp

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	opampServer "github.com/open-telemetry/opamp-go/server"
	"github.com/open-telemetry/opamp-go/server/types"

	"github.com/minuk-dev/opampcommander/internal/application/port"
)

// Controller is a struct that implements OPAMP protocol.
// It handles the connection and message processing for the OPAMP protocol.
type Controller struct {
	logger *slog.Logger

	handler     opampServer.HTTPHandlerFunc
	ConnContext opampServer.ConnContext

	opampServer       opampServer.OpAMPServer
	enableCompression bool

	// usecases
	opampUsecase port.OpAMPUsecase
}

// Option is a function that takes a Controller and modifies it.
type Option func(*Controller)

// NewController creates a new instance of Controller.
func NewController(
	opampUsecase port.OpAMPUsecase,
	logger *slog.Logger,
) *Controller {
	ops := opampServer.New(&Logger{
		logger: logger,
	})

	controller := &Controller{
		logger:       logger,
		opampUsecase: opampUsecase,

		enableCompression: false,

		handler:     nil, // fill below
		ConnContext: nil, // fill below
		opampServer: ops,
	}

	var err error

	controller.handler, controller.ConnContext, err = ops.Attach(opampServer.Settings{
		EnableCompression: controller.enableCompression,
		Callbacks: types.Callbacks{
			OnConnecting: controller.OnConnecting,
		},
		CustomCapabilities: nil,
	})
	if err != nil {
		controller.logger.Error("failed to attach opamp server", "error", err.Error())

		return nil
	}

	return controller
}

// OnConnecting is a method that handles the connection request.
// It is an adapter for the opampServer's OnConnecting callback.
func (c *Controller) OnConnecting(req *http.Request) types.ConnectionResponse {
	c.logger.Debug("OnConnecting", slog.Any("req", req))

	// Detect connection type based on HTTP request
	// WebSocket connections have "Upgrade: websocket" header
	// HTTP connections use POST method without upgrade
	isWebSocket := req.Header.Get("Upgrade") == "websocket"

	return types.ConnectionResponse{
		Accept:             true,
		HTTPStatusCode:     http.StatusOK,
		HTTPResponseHeader: map[string]string{},
		ConnectionCallbacks: types.ConnectionCallbacks{
			OnConnected: func(ctx context.Context, conn types.Connection) {
				c.opampUsecase.OnConnectedWithType(ctx, conn, isWebSocket)
			},
			OnMessage:              c.opampUsecase.OnMessage,
			OnConnectionClose:      c.opampUsecase.OnConnectionClose,
			OnReadMessageError:     c.opampUsecase.OnReadMessageError,
			OnMessageResponseError: c.opampUsecase.OnMessageResponseError,
		},
	}
}

// RoutesInfo returns the routes information for the controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      http.MethodGet,
			Path:        "/api/v1/opamp",
			Handler:     "opamp.v1.opamp.Handle",
			HandlerFunc: c.Handle,
		},
		{
			Method:      http.MethodPost,
			Path:        "/api/v1/opamp",
			Handler:     "opamp.v1.opamp.Handle",
			HandlerFunc: c.Handle,
		},
	}
}

// Handle is a method that handles the HTTP request.
func (c *Controller) Handle(ctx *gin.Context) {
	c.logger.Info("Handle", "message", "start")
	c.handler(ctx.Writer, ctx.Request)
}
