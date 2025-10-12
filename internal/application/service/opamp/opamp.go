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
)

const (
	// DefaultOnConnectionCloseTimeout is the default timeout for closing a connection.
	DefaultOnConnectionCloseTimeout = 5 * time.Second
)

// Service is a struct that implements the OpAMPUsecase interface.
type Service struct {
	logger         *slog.Logger
	agentUsecase   domainport.AgentUsecase
	commandUsecase domainport.CommandUsecase

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
	logger *slog.Logger,
) *Service {
	return &Service{
		logger:            logger,
		agentUsecase:      agentUsecase,
		commandUsecase:    commandUsecase,
		connectionUsecase: connectionUsecase,
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

	connection := model.NewConnection(conn, model.TypeUnknown)

	err := s.connectionUsecase.SaveConnection(ctx, connection)
	if err != nil {
		logger.Error("failed to save connection", slog.String("error", err.Error()))
	}

	logger.Info("end")
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

		err = s.connectionUsecase.DeleteConnection(ctx, connection)
		if err != nil {
			logger.Error("failed to delete connection", slog.String("error", err.Error()))

			return
		}
	}
	s.backgroundLoopCh <- backgroundFn

	logger.Info("end")
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

		connection = model.NewConnection(conn, model.TypeUnknown)
	}

	connection.InstanceUID = instanceUID

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

	err = s.report(agent, message)
	if err != nil {
		logger.Error("failed to report agent", slog.String("error", err.Error()))
	}

	err = s.agentUsecase.SaveAgent(ctx, agent)
	if err != nil {
		logger.Error("failed to save agent", slog.String("error", err.Error()))
	}

	response := s.fetchServerToAgent(agent)

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
