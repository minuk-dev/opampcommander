package opamp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/pkg/wsprotobufutil"
)

const (
	headerContentType   = "Content-Type"
	contentTypeProtobuf = "application/x-protobuf"
)

type Controller struct {
	logger     *slog.Logger
	wsUpgrader websocket.Upgrader

	// usecases
	opampUsecase port.OpAMPUsecase
}

type Option func(*Controller)

func NewController(opampUsecase port.OpAMPUsecase, options ...Option) *Controller {
	controller := &Controller{
		logger: slog.Default(),
		wsUpgrader: websocket.Upgrader{
			HandshakeTimeout:  0,
			ReadBufferSize:    0,
			WriteBufferSize:   0,
			WriteBufferPool:   nil,
			Subprotocols:      nil,
			Error:             nil,
			CheckOrigin:       nil,
			EnableCompression: false,
		},

		opampUsecase: opampUsecase,
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
	switch {
	case isHTTPRequest(ctx.Request):
		c.handleHTTPRequest(ctx)
	case isWSRequest(ctx.Request):
		c.handleWSRequest(ctx)
	default:
		c.logger.Warn("cannot handle type")
		ctx.Writer.WriteHeader(http.StatusBadRequest)
	}
}

func (c *Controller) handleHTTPRequest(_ *gin.Context) {
}

func (c *Controller) handleWSRequest(ctx *gin.Context) {
	w, req := ctx.Writer, ctx.Request

	conn, err := c.wsUpgrader.Upgrade(w, req, nil)
	if err != nil {
		c.logger.Warn("Cannot upgrade HTTP connection to WebSocket", "error", err.Error())

		return
	}

	c.handleWSConnection(req.Context(), conn)
}

func (c *Controller) handleWSConnection(ctx context.Context, conn *websocket.Conn) {
	defer conn.Close()

	agentToServer, err := wsprotobufutil.Receive(ctx, conn)
	if err != nil {
		c.logger.Error("handleWSConnection", slog.Any("error", err))

		return
	}

	instanceUID := uuid.UUID(agentToServer.GetInstanceUid())

	err = c.opampUsecase.HandleAgentToServer(ctx, agentToServer)
	if err != nil {
		c.logger.Error("handleWSConnection", slog.Any("error", err))

		return // first HandleAgentToServer should be success
	}

	var wgConn sync.WaitGroup
	// reader loop
	wgConn.Add(1)

	go func() {
		defer wgConn.Done()
		c.readerLoop(ctx, conn)
	}()

	// writer loop
	wgConn.Add(1)

	go func() {
		defer wgConn.Done()
		c.writeLoop(ctx, conn, instanceUID)
	}()
	wgConn.Wait()
}

func (c *Controller) readerLoop(ctx context.Context, conn *websocket.Conn) {
	for {
		agentToServer, err := wsprotobufutil.Receive(ctx, conn)
		if err != nil {
			c.logger.Error("handleWSConnection", slog.Any("error", err))

			return
		}

		err = c.opampUsecase.HandleAgentToServer(ctx, agentToServer)
		if err != nil {
			c.logger.Error("handleWSConnection", slog.Any("error", err))
			// even if opampUsecase is failed, we should continue to read from the connection
			continue
		}
	}
}

func (c *Controller) writeLoop(ctx context.Context, conn *websocket.Conn, instanceUID uuid.UUID) {
	for {
		serverToAgent, err := c.opampUsecase.FetchServerToAgent(ctx, instanceUID)
		if err != nil {
			c.logger.Error("handleWSConnection", slog.Any("error", err))

			continue
		}

		err = wsprotobufutil.Send(ctx, conn, serverToAgent)
		if err != nil && errors.Is(err, websocket.ErrCloseSent) {
			c.logger.Error("handleWSConnection", slog.Any("error", err))

			return
		} else if err != nil {
			c.logger.Error("handleWSConnection", slog.Any("error", err))

			continue
		}
	}
}

func isHTTPRequest(req *http.Request) bool {
	contentType := req.Header.Get(headerContentType)
	contentType = strings.ToLower(contentType)

	return contentType == strings.ToLower(contentTypeProtobuf)
}

func isWSRequest(_ *http.Request) bool {
	return true
}
