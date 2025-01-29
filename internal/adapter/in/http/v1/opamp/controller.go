package opamp

import (
	"context"
	"net/http"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	headerContentType = "Content-Type"
	contentTypeProtobuf = "application/x-protobuf"
)

type Controller struct {
	logger     *slog.Logger
	wsUpgrader websocket.Upgrader
}

type Option func(*Controller)

func NewController(options ...Option) *Controller {
	c := &Controller{
		logger: slog.Default(),
		wsUpgrader: websocket.Upgrader{
			EnableCompression: false,
		},
	}

	for _, option := range options {
		option(c)
	}
	return c
}

func (c *Controller) Path() string {
	return "/v1/opamp"
}

func (c *Controller) Handle(ctx *gin.Context) {
	if isHTTPRequest(ctx.Request) {
		c.handleHTTPRequest(ctx)
	} else if isWSRequest(ctx.Request) {
		c.handleWSRequest(ctx)
	} else {
		c.logger.Warn("cannot handle type")
		ctx.Writer.WriteHeader(http.StatusBadRequest)
	}
}

func (c *Controller) handleHTTPRequest(ctx *gin.Context) {
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
	wsConn := newWSConnection(conn)
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
