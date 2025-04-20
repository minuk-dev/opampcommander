// Package opamp provides the implementation of the OPAMP protocol.
package opamp

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
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

	connections       map[types.Connection]struct{}
	opampServer       opampServer.OpAMPServer
	enableCompression bool

	// usecases
	opampUsecase port.OpAMPUsecase
}

// Option is a function that takes a Controller and modifies it.
type Option func(*Controller)

// Logger is a struct which wraps the slog.Logger for supporting OpAMP logger interface.
type Logger struct {
	logger *slog.Logger
}

// Debugf is a method that logs a debug message.
func (l *Logger) Debugf(_ context.Context, format string, v ...any) {
	l.logger.Debug(format, v...)
}

// Errorf is a method that logs an error message.
func (l *Logger) Errorf(_ context.Context, format string, v ...any) {
	l.logger.Error(format, v...)
}

// NewController creates a new instance of Controller.
func NewController(
	opampUsecase port.OpAMPUsecase,
	logger *slog.Logger,
) *Controller {
	controller := &Controller{
		logger:       logger,
		connections:  make(map[types.Connection]struct{}),
		opampUsecase: opampUsecase,

		enableCompression: false,

		handler:     nil, // fill below
		ConnContext: nil, // fill below
		opampServer: nil, // fill below
	}

	controller.opampServer = opampServer.New(&Logger{
		logger: controller.logger,
	})

	var err error

	controller.handler, controller.ConnContext, err = controller.opampServer.Attach(opampServer.Settings{
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

	if err != nil {
		controller.logger.Error("controller validation failed", "error", err.Error())

		return nil
	}

	return controller
}

// OnConnecting is a method that handles the connection request.
// It is an adapter for the opampServer's OnConnecting callback.
func (c *Controller) OnConnecting(*http.Request) types.ConnectionResponse {
	return types.ConnectionResponse{
		Accept:             true,
		HTTPStatusCode:     http.StatusOK,
		HTTPResponseHeader: map[string]string{},
		ConnectionCallbacks: types.ConnectionCallbacks{
			OnConnected:       c.OnConnected,
			OnMessage:         c.OnMessage,
			OnConnectionClose: c.OnConnectionClose,
		},
	}
}

// OnConnected is a method that handles the connection established event.
// It is an adapter for the opampServer's OnConnected callback.
func (c *Controller) OnConnected(_ context.Context, conn types.Connection) {
	c.connections[conn] = struct{}{}
}

// OnMessage is a method that handles the incoming message.
// It is an adapter for the opampServer's OnMessage callback.
func (c *Controller) OnMessage(
	ctx context.Context,
	_ types.Connection,
	message *protobufs.AgentToServer,
) *protobufs.ServerToAgent {
	instanceUID := message.GetInstanceUid()

	err := c.opampUsecase.HandleAgentToServer(ctx, message)
	if err != nil {
		c.logger.Error("failed to handle agent to server message", "error", err.Error())
	}

	serverToAgent, err := c.opampUsecase.FetchServerToAgent(ctx, uuid.UUID(instanceUID))
	if err != nil {
		c.logger.Error("failed to fetch server to agent message", "error", err.Error())
	}

	return serverToAgent
}

// OnConnectionClose is a method that handles the connection close event.
// It is an adapter for the opampServer's OnConnectionClose callback.
func (c *Controller) OnConnectionClose(conn types.Connection) {
	delete(c.connections, conn)
}

// RoutesInfo returns the routes information for the controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      "GET",
			Path:        "/v1/opamp",
			Handler:     "opamp.v1.opamp.Handle",
			HandlerFunc: c.Handle,
		},
		{
			Method:      "POST",
			Path:        "/v1/opamp",
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
