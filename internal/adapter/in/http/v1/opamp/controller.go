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

const (
	headerContentType   = "Content-Type"
	contentTypeProtobuf = "application/x-protobuf"
)

type Controller struct {
	logger *slog.Logger

	handler     opampServer.HTTPHandlerFunc
	ConnContext opampServer.ConnContext

	connections map[types.Connection]struct{}

	opampServer opampServer.OpAMPServer
	// usecases
	opampUsecase port.OpAMPUsecase
}

type Option func(*Controller)

type Logger struct {
	logger *slog.Logger
}

func (l *Logger) Debugf(ctx context.Context, format string, v ...interface{}) {
	l.logger.Debug(format, v...)
}

func (l *Logger) Errorf(ctx context.Context, format string, v ...interface{}) {
	l.logger.Error(format, v...)
}

func NewController(opampUsecase port.OpAMPUsecase, options ...Option) *Controller {
	controller := &Controller{
		logger:       slog.Default(),
		connections:  make(map[types.Connection]struct{}),
		opampUsecase: opampUsecase,
	}

	for _, option := range options {
		option(controller)
	}

	controller.opampServer = opampServer.New(&Logger{
		logger: controller.logger,
	})
	var err error
	controller.handler, controller.ConnContext, err = controller.opampServer.Attach(opampServer.Settings{
		Callbacks: types.Callbacks{
			OnConnecting: controller.OnConnecting,
		},
	})
	if err != nil {
		controller.logger.Error("failed to attach opamp server", "error", err.Error())
	}

	err = controller.Validate()
	if err != nil {
		controller.logger.Error("controller validation failed", "error", err.Error())

		return nil
	}

	return controller
}

func (c *Controller) OnConnecting(request *http.Request) types.ConnectionResponse {
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

func (c *Controller) OnConnected(ctx context.Context, conn types.Connection) {
	c.connections[conn] = struct{}{}
}

func (c *Controller) OnMessage(ctx context.Context, conn types.Connection, message *protobufs.AgentToServer) *protobufs.ServerToAgent {
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

func (c *Controller) OnConnectionClose(conn types.Connection) {
	delete(c.connections, conn)
}

func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      "GET",
			Path:        "/v1/opamp",
			Handler:     "opamp.v1.opamp.Handle",
			HandlerFunc: c.Handle,
		},
	}
}

func (c *Controller) Validate() error {
	if c.opampUsecase == nil {
		return &UsecaseNotProvidedError{
			Usecase: "opamp",
		}
	}

	return nil
}

func (c *Controller) Handle(ctx *gin.Context) {
	c.handler(ctx.Writer, ctx.Request)
}
