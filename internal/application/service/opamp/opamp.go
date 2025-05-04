// Package opamp provides the implementation of the OpAMP use case for managing connections and agents.
package opamp

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server/types"
	"github.com/samber/lo"

	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

const (
	DefaultOnConnectionCloseTimeout = 5 * time.Second
)

var _ applicationport.OpAMPUsecase = (*Service)(nil)

// Service is a struct that implements the OpAMPUsecase interface.
type Service struct {
	logger            *slog.Logger
	connectionUsecase domainport.ConnectionUsecase
	agentUsecase      domainport.AgentUsecase
	commandUsecase    domainport.CommandUsecase

	OnConnectionCloseTimeout time.Duration
}

// New creates a new instance of the OpAMP service.
func New(
	connectionUsecase domainport.ConnectionUsecase,
	agentUsecase domainport.AgentUsecase,
	commandUsecase domainport.CommandUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		logger:            logger,
		connectionUsecase: connectionUsecase,
		agentUsecase:      agentUsecase,
		commandUsecase:    commandUsecase,

		OnConnectionCloseTimeout: DefaultOnConnectionCloseTimeout,
	}
}

// OnConnected implements port.OpAMPUsecase.
func (s *Service) OnConnected(ctx context.Context, conn types.Connection) {
	remoteAddr := conn.Connection().RemoteAddr().String()
	logger := s.logger.With(slog.String("method", "OnConnected"), slog.String("remoteAddr", remoteAddr))

	logger.Info("start")

	data := createDataFromConnection(conn)
	connection := model.NewAnonymousConnection(data)

	err := s.connectionUsecase.SaveConnection(ctx, connection)
	if err != nil {
		logger.Error("failed to save connection", slog.String("error", err.Error()))

		return
	}

	logger.Info("end")
}

// OnConnectionClose implements port.OpAMPUsecase.
func (s *Service) OnConnectionClose(conn types.Connection) {
	// TODO: run async & managed ctx for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), s.OnConnectionCloseTimeout)
	defer cancel() // ensure the context is cancelled

	remoteAddr := conn.Connection().RemoteAddr().String()
	logger := s.logger.With(slog.String("method", "OnConnectionClose"), slog.String("remoteAddr", remoteAddr))
	logger.Info("start")

	data := createDataFromConnection(conn)

	connections, err := s.connectionUsecase.FindConnectionsByData(ctx, data)
	if err != nil {
		logger.Error("failed to find connections", slog.String("error", err.Error()))

		return
	}

	if len(connections) == 0 {
		logger.Error("failed to find connections because there is connection")

		return
	}

	connection := lo.FindOrElse(connections, connections[0], func(c *model.Connection) bool { return c.IsIdentified() })

	err = s.connectionUsecase.DeleteConnection(ctx, connection)
	if err != nil {
		logger.Error("failed to delete connection", slog.String("error", err.Error()))

		return
	}
}

// OnMessage implements port.OpAMPUsecase.
func (s *Service) OnMessage(ctx context.Context, conn types.Connection, message *protobufs.AgentToServer) *protobufs.ServerToAgent {
	remoteAddr := conn.Connection().RemoteAddr().String()
	instanceUID := uuid.UUID(message.GetInstanceUid())
	logger := s.logger.With(
		slog.String("method", "OnMessage"),
		slog.String("remoteAddr", remoteAddr),
		slog.String("instanceUID", instanceUID.String()),
	)

	logger.Info("start")
	logger.Debug("processing agentToServer", slog.Any("agentToServer", message))

	agent, err := s.agentUsecase.GetOrCreateAgent(ctx, instanceUID)
	if err != nil {
		logger.Error("failed to get agent", slog.String("error", err.Error()))
	}

	logger.Debug("agent", slog.Any("agent", agent))

	err = s.report(agent, message)
	if err != nil {
		logger.Error("failed to report agent", slog.String("error", err.Error()))
	}

	logger.Debug("report agent", slog.Any("agent", agent))

	if !agent.IsManaged() {
		logger.Debug("mark reportFullState as true due to unmanaged agent")
		agent.SetReportFullState(true)
	}

	err = s.agentUsecase.SaveAgent(ctx, agent)
	if err != nil {
		logger.Error("failed to save agent", slog.String("error", err.Error()))
	}

	response, err := s.createServerToAgent(agent)
	if err != nil {
		logger.Error("failed to create serverToAgent", slog.String("error", err.Error()))

		response = s.createFallbackServerToAgent(instanceUID)
	}

	logger.Debug("serverToAgent", slog.Any("serverToAgent", response))
	logger.Info("end")

	return response
}

func createDataFromConnection(conn types.Connection) map[string]string {
	remoteAddr := conn.Connection().RemoteAddr().String()
	data := map[string]string{
		"remoteAddr": remoteAddr,
	}

	return data
}
