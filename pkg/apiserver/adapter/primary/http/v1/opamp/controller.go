// Package opamp provides the implementation of the OPAMP protocol.
package opamp

import (
	"context"
	"crypto/x509"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	opampServer "github.com/open-telemetry/opamp-go/server"
	"github.com/open-telemetry/opamp-go/server/types"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/common/opampconnection"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/usecase"
)

// Controller is a struct that implements OPAMP protocol.
// It handles the connection and message processing for the OPAMP protocol.
type Controller struct {
	logger *slog.Logger

	transport *opampconnection.Transport

	handler     opampServer.HTTPHandlerFunc
	ConnContext opampServer.ConnContext

	opampServer              opampServer.OpAMPServer
	enableCompression        bool
	RequireClientCertificate bool

	// usecases
	opampUsecase usecase.OpAMPUsecase
}

// Option is a function that takes a Controller and modifies it.
type Option func(*Controller)

// NewController creates a new instance of Controller.
func NewController(
	opampUsecase usecase.OpAMPUsecase,
	logger *slog.Logger,
	transport *opampconnection.Transport,
) *Controller {
	ops := opampServer.New(&Logger{
		logger: logger,
	})

	controller := &Controller{
		logger:       logger,
		transport:    transport,
		opampUsecase: opampUsecase,

		enableCompression: false,

		handler:                  nil, // fill below
		ConnContext:              nil, // fill below
		opampServer:              ops,
		RequireClientCertificate: false,
	}

	var err error

	controller.handler, controller.ConnContext, err = ops.Attach(opampServer.Settings{
		EnableCompression: controller.enableCompression,
		MaxMessageSize:    0, // opamp-go default (64 MiB)
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

	if c.RequireClientCertificate && (req.TLS == nil || len(req.TLS.VerifiedChains) == 0 ||
		len(req.TLS.PeerCertificates) == 0) {
		//exhaustruct:ignore
		return types.ConnectionResponse{Accept: false, HTTPStatusCode: http.StatusUnauthorized}
	}

	// Detect connection type based on HTTP request
	// WebSocket connections have "Upgrade: websocket" header
	// HTTP connections use POST method without upgrade
	isWebSocket := req.Header.Get("Upgrade") == "websocket"

	onMessage := c.opampUsecase.OnMessage
	if c.RequireClientCertificate {
		cert := req.TLS.PeerCertificates[0]
		onMessage = func(
			ctx context.Context, conn types.Connection, message *protobufs.AgentToServer,
		) *protobufs.ServerToAgent {
			return c.onClientCertificateMessage(ctx, conn, message, cert)
		}
	}

	return types.ConnectionResponse{
		Accept:             true,
		HTTPStatusCode:     http.StatusOK,
		HTTPResponseHeader: map[string]string{},
		ConnectionCallbacks: types.ConnectionCallbacks{
			OnConnected: func(ctx context.Context, conn types.Connection) {
				c.opampUsecase.OnConnectedWithType(ctx, conn, isWebSocket)

				if isWebSocket {
					c.transport.Connected(conn.Connection())
				}
			},
			OnMessage:              onMessage,
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

	if !c.transport.Accepting() {
		ctx.Status(http.StatusServiceUnavailable)

		return
	}

	c.handler(c.transport.WrapWriter(ctx.Writer), ctx.Request)
}

func (c *Controller) onClientCertificateMessage(
	ctx context.Context, conn types.Connection, message *protobufs.AgentToServer, cert *x509.Certificate,
) *protobufs.ServerToAgent {
	if !c.authorizeAgentMessage(ctx, cert, message) {
		return &protobufs.ServerToAgent{
			InstanceUid: message.GetInstanceUid(),
			ErrorResponse: &protobufs.ServerErrorResponse{
				Type:         protobufs.ServerErrorResponseType_ServerErrorResponseType_BadRequest,
				ErrorMessage: "client certificate is not authorized for agent instance UID",
			},
		}
	}

	return c.opampUsecase.OnMessage(ctx, conn, message)
}

func (c *Controller) authorizeAgentMessage(
	ctx context.Context, cert *x509.Certificate, message *protobufs.AgentToServer,
) bool {
	uid, err := uuid.FromBytes(message.GetInstanceUid())
	if err != nil || uid.String() != cert.Subject.CommonName {
		return false
	}

	return c.opampUsecase.AuthorizeClientCertificate(ctx, uid, cert.Raw)
}
