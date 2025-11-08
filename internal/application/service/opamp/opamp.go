// Package opamp provides the implementation of the OpAMP use case for managing connections and agents.
package opamp

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server/types"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

const (
	// DefaultOnConnectionCloseTimeout is the default timeout for closing a connection.
	DefaultOnConnectionCloseTimeout = 5 * time.Second
)

// Service is a struct that implements the OpAMPUsecase interface.
type Service struct {
	clock             clock.Clock
	logger            *slog.Logger
	agentUsecase      domainport.AgentUsecase
	commandUsecase    domainport.CommandUsecase
	serverUsecase     domainport.ServerUsecase
	agentGroupUsecase domainport.AgentGroupUsecase
	wsRegistry        domainport.WebSocketRegistry

	backgroundLoopCh chan backgroundCallbackFn

	connectionUsecase        domainport.ConnectionUsecase
	OnConnectionCloseTimeout time.Duration
}

type backgroundCallbackFn func(context.Context)

// New creates a new instance of the OpAMP service.
func New(
	agentUsecase domainport.AgentUsecase,
	commandUsecase domainport.CommandUsecase,
	connectionUsecase domainport.ConnectionUsecase,
	serverUsecase domainport.ServerUsecase,
	agentGroupUsecase domainport.AgentGroupUsecase,
	wsRegistry domainport.WebSocketRegistry,
	logger *slog.Logger,
) *Service {
	return &Service{
		clock:             clock.NewRealClock(),
		logger:            logger,
		agentUsecase:      agentUsecase,
		commandUsecase:    commandUsecase,
		connectionUsecase: connectionUsecase,
		serverUsecase:     serverUsecase,
		agentGroupUsecase: agentGroupUsecase,
		wsRegistry:        wsRegistry,
		backgroundLoopCh:  make(chan backgroundCallbackFn),

		OnConnectionCloseTimeout: DefaultOnConnectionCloseTimeout,
	}
}

// Name returns the name of the service.
func (s *Service) Name() string {
	return "opamp"
}

// Run starts a loop to handle asynchronous operations for the service.
func (s *Service) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("context done, exiting service loop")

			return fmt.Errorf("service loop exited: %w", ctx.Err())
		case action := <-s.backgroundLoopCh:
			action(ctx)
		}
	}
}

// OnConnected implements port.OpAMPUsecase.
func (s *Service) OnConnected(ctx context.Context, conn types.Connection) {
	remoteAddr := conn.Connection().RemoteAddr().String()
	logger := s.logger.With(slog.String("method", "OnConnected"), slog.String("remoteAddr", remoteAddr))

	// Detect connection type by checking if it's a WebSocket connection
	// WebSocket connections support the Send method, while HTTP connections don't
	connType := s.detectConnectionType(conn)

	connection := model.NewConnection(conn, connType)
	connID := connection.IDString()

	// For WebSocket connections, register in the WebSocket registry
	if connType == model.ConnectionTypeWebSocket {
		wsConn := s.wrapOpAMPConnection(conn)
		s.wsRegistry.Register(connID, wsConn)
	}

	err := s.connectionUsecase.SaveConnection(ctx, connection)
	if err != nil {
		logger.Error("failed to save connection", slog.String("error", err.Error()))
	}

	logger.Info("end", slog.String("connectionType", s.connectionTypeString(connType)))
}

// opampConnectionAdapter adapts OpAMP's Connection to our WebSocketConnection interface.
type opampConnectionAdapter struct {
	conn types.Connection
}

func (a *opampConnectionAdapter) Send(ctx context.Context, message *protobufs.ServerToAgent) error {
	return a.conn.Send(ctx, message) //nolint:wrapcheck // adapter pattern
}

func (a *opampConnectionAdapter) Close() error {
	return a.conn.Disconnect() //nolint:wrapcheck // adapter pattern
}

// OnMessage implements port.OpAMPUsecase.
func (s *Service) OnMessage(
	ctx context.Context,
	conn types.Connection,
	message *protobufs.AgentToServer,
) *protobufs.ServerToAgent {
	remoteAddr := conn.Connection().RemoteAddr().String()
	instanceUID := uuid.UUID(message.GetInstanceUid())

	logger := s.logger.With(
		slog.String("method", "OnMessage"),
		slog.String("remoteAddr", remoteAddr),
		slog.String("instanceUID", instanceUID.String()),
	)
	connection, err := s.connectionUsecase.GetConnectionByID(ctx, conn)

	// Even if the connection is not found, we should still process the message
	if err != nil {
		logger.Error("failed to get connection", slog.String("error", err.Error()))

		connType := s.detectConnectionType(conn)

		connection = model.NewConnection(conn, connType)
		if connType == model.ConnectionTypeWebSocket {
			connID := connection.IDString()
			wsConn := s.wrapOpAMPConnection(conn)
			s.wsRegistry.Register(connID, wsConn)
		}
	}

	connection.InstanceUID = instanceUID
	connID := connection.IDString()

	// Update connection type to WebSocket if we're processing a message
	// (HTTP connections would not repeatedly call OnMessage)
	if connection.Type == model.ConnectionTypeUnknown {
		connection.Type = model.ConnectionTypeWebSocket
		wsConn := s.wrapOpAMPConnection(conn)
		s.wsRegistry.Register(connID, wsConn)
	}

	// Update the instance UID mapping in the registry for WebSocket connections
	if connection.Type == model.ConnectionTypeWebSocket {
		s.wsRegistry.UpdateInstanceUID(connID, instanceUID.String())
	}

	err = s.connectionUsecase.SaveConnection(ctx, connection)
	if err != nil {
		logger.Error("failed to save connection", slog.String("error", err.Error()))
	}

	logger.Info("start")

	agent, err := s.agentUsecase.GetOrCreateAgent(ctx, instanceUID)
	if err != nil {
		logger.Error("failed to get agent", slog.String("error", err.Error()))

		return s.createFallbackServerToAgent(instanceUID)
	}

	// Update agent connection status
	agent.Status.LastConnectionType = connection.Type
	agent.Status.Connected = true

	currentServer, err := s.serverUsecase.CurrentServer(ctx)
	if err != nil {
		logger.Warn("failed to get current server", slog.String("error", err.Error()))
	}

	err = s.report(agent, message, currentServer)
	if err != nil {
		logger.Error("failed to report agent", slog.String("error", err.Error()))
	}

	err = s.agentUsecase.SaveAgent(ctx, agent)
	if err != nil {
		logger.Error("failed to save agent", slog.String("error", err.Error()))
	}

	response := s.fetchServerToAgent(ctx, agent)

	logger.Info("end successfully")

	return response
}

// OnReadMessageError implements port.OpAMPUsecase.
func (s *Service) OnReadMessageError(
	conn types.Connection,
	messageType int,
	msgByte []byte,
	err error,
) {
	remoteAddr := conn.Connection().RemoteAddr().String()
	logger := s.logger.With(
		slog.String("method", "OnReadMessageError"),
		slog.String("remoteAddr", remoteAddr),
		slog.Int("messageType", messageType),
		slog.String("message", string(msgByte)),
		slog.String("error", err.Error()),
	)

	logger.Error("read message error")
}

// OnMessageResponseError implements port.OpAMPUsecase.
func (s *Service) OnMessageResponseError(conn types.Connection, message *protobufs.ServerToAgent, err error) {
	remoteAddr := conn.Connection().RemoteAddr().String()
	logger := s.logger.With(
		slog.String("method", "OnMessageResponseError"),
		slog.String("remoteAddr", remoteAddr),
		slog.String("message", fmt.Sprintf("%+v", message)),
		slog.String("error", err.Error()),
	)

	logger.Error("send message error")
}

// OnConnectionClose implements port.OpAMPUsecase.
func (s *Service) OnConnectionClose(conn types.Connection) {
	remoteAddr := conn.Connection().RemoteAddr().String()
	logger := s.logger.With(slog.String("method", "OnConnectionClose"), slog.String("remoteAddr", remoteAddr))
	logger.Info("start")

	backgroundFn := func(ctx context.Context) {
		logger.Info("start")

		connection, err := s.connectionUsecase.GetConnectionByID(ctx, conn)
		if err != nil {
			logger.Error("failed to get connection", slog.String("error", err.Error()))

			return
		}

		// Update agent connection status to disconnected
		if !connection.IsAnonymous() {
			agent, err := s.agentUsecase.GetAgent(ctx, connection.InstanceUID)
			if err != nil {
				logger.Error("failed to get agent for connection close", slog.String("error", err.Error()))
			} else {
				agent.Status.Connected = false

				err = s.agentUsecase.SaveAgent(ctx, agent)
				if err != nil {
					logger.Error("failed to save agent connection status", slog.String("error", err.Error()))
				}
			}
		}

		err = s.connectionUsecase.DeleteConnection(ctx, connection)
		if err != nil {
			logger.Error("failed to delete connection", slog.String("error", err.Error()))

			return
		}
	}
	s.backgroundLoopCh <- backgroundFn

	logger.Info("end")
}

// detectConnectionType detects whether the connection is WebSocket or HTTP.
// According to OpAMP spec, WebSocket connections support bidirectional communication
// and can use the Send method, while HTTP connections are request-response only.
func (s *Service) detectConnectionType(conn types.Connection) model.ConnectionType {
	// Try to send a nil message to detect if it's a WebSocket connection
	// WebSocket will accept this (though we won't actually send anything),
	// while HTTP will return an error
	// Note: We don't actually want to send anything here, we just want to check the type
	// A better approach is to check if the connection implements a WebSocket-specific interface
	// For now, we'll assume it's WebSocket and let it be corrected later if needed

	// Check the underlying connection type
	// If the connection's remote address is set and it's persistent, it's likely WebSocket
	netConn := conn.Connection()
	if netConn != nil {
		// WebSocket connections typically keep the connection open
		// For now, we'll default to WebSocket since most OpAMP implementations use WebSocket
		// This will be updated in OnMessage when we receive the first message
		return model.ConnectionTypeWebSocket
	}

	return model.ConnectionTypeHTTP
}

// connectionTypeString returns a string representation of the connection type.
func (s *Service) connectionTypeString(connType model.ConnectionType) string {
	switch connType {
	case model.ConnectionTypeWebSocket:
		return "WebSocket"
	case model.ConnectionTypeHTTP:
		return "HTTP"
	case model.ConnectionTypeUnknown:
		return "Unknown"
	default:
		return "Undefined"
	}
}

// wrapOpAMPConnection wraps an OpAMP connection into our WebSocketConnection interface.
func (s *Service) wrapOpAMPConnection(conn types.Connection) domainport.WebSocketConnection {
	return &opampConnectionAdapter{conn: conn}
}
