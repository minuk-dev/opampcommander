package opamp

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	headerContentType   = "Content-Type"
	contentTypeProtobuf = "application/x-protobuf"
)

type Controller struct {
	logger     *slog.Logger
	wsUpgrader websocket.Upgrader
}

type Option func(*Controller)

func NewController(options ...Option) *Controller {
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
	}

	for _, option := range options {
		option(controller)
	}

	return controller
}

func (c *Controller) Path() string {
	return "/v1/opamp"
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

	go c.handleWSConnection(req.Context(), conn)
}

func (c *Controller) handleWSConnection(ctx context.Context, conn *websocket.Conn) {
	wsConn := newWSConnection(conn, c.logger)
	defer wsConn.Close()

	err := wsConn.Run(ctx)
	if err != nil {
		c.logger.Warn("Cannot run WebSocket connection", "error", err.Error())
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
