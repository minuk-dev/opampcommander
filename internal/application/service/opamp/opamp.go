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
	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/model/agent"
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
	serverUsecase     domainport.ServerUsecase
	agentGroupUsecase domainport.AgentGroupUsecase

	agentNotificationUsecase domainport.AgentNotificationUsecase

	closedConnectionCh chan types.Connection

	connectionUsecase        domainport.ConnectionUsecase
	onConnectionCloseTimeout time.Duration
}

// New creates a new instance of the OpAMP service.
func New(
	agentUsecase domainport.AgentUsecase,
	connectionUsecase domainport.ConnectionUsecase,
	serverUsecase domainport.ServerUsecase,
	agentGroupUsecase domainport.AgentGroupUsecase,
	agentNotificationUsecase domainport.AgentNotificationUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		clock:                    clock.NewRealClock(),
		logger:                   logger,
		agentUsecase:             agentUsecase,
		connectionUsecase:        connectionUsecase,
		serverUsecase:            serverUsecase,
		agentGroupUsecase:        agentGroupUsecase,
		agentNotificationUsecase: agentNotificationUsecase,
		closedConnectionCh:       make(chan types.Connection, 1), // buffered channel

		onConnectionCloseTimeout: DefaultOnConnectionCloseTimeout,
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
		case conn := <-s.closedConnectionCh:
			bgCtx, cancel := context.WithTimeout(ctx, s.onConnectionCloseTimeout)
			defer cancel()

			err := s.cleanUpConnection(bgCtx, conn)
			if err != nil {
				s.logger.Error("failed to clean up connection", slog.String("error", err.Error()))
			}
		}
	}
}

// OnConnected implements port.OpAMPUsecase.
func (s *Service) OnConnected(ctx context.Context, conn types.Connection) {
	remoteAddr := conn.Connection().RemoteAddr().String()
	logger := s.logger.With(slog.String("method", "OnConnected"), slog.String("remoteAddr", remoteAddr))

	logger.Info("start")

	connection, err := s.connectionUsecase.GetOrCreateConnectionByID(ctx, conn)
	if err != nil {
		logger.Error("failed to get or create connection", slog.String("error", err.Error()))

		return
	}

	err = s.connectionUsecase.SaveConnection(ctx, connection)
	if err != nil {
		logger.Error("failed to save connection", slog.String("error", err.Error()))

		return
	}

	logger.Info("end successfully")
}

// OnMessage implements port.OpAMPUsecase.
// [1] find domainmodel.Connection by types.Connection
// [1-1] if not found, unexpected case because all connections should be created when OnConnected is called.
// so, leave error log and skip connection processing.
// [2] find domainmodel.Agent by instanceUID in message
// [2-1] if not found, this is the first time the agent connects, so create a new agent with default values.
// [3] process the message and update agent state accordingly
// [4] save the updated agent
// [5] fetch ServerToAgent message to send back to the agent
// [6] return the ServerToAgent message.
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
	logger.Info("start")

	err := s.injectInstanceUIDToConnection(ctx, conn, instanceUID)
	if err != nil {
		logger.Error("failed to inject instanceUID to connection", slog.String("error", err.Error()))
		// even if injecting instanceUID fails, proceed to process the message
	}

	currentServer, err := s.serverUsecase.CurrentServer(ctx)
	if err != nil {
		logger.Warn("failed to get current server", slog.String("error", err.Error()))
	}

	agent, err := s.agentUsecase.GetOrCreateAgent(ctx, instanceUID)
	if err != nil {
		logger.Error("failed to get agent", slog.String("error", err.Error()))

		// whan the agent cannot be retrieved, return a fallback ServerToAgent message
		return s.createFallbackServerToAgent(instanceUID)
	}

	// Update agent connection status
	agent.UpdateLastCommunicationInfo(s.clock.Now())

	err = s.report(agent, message, currentServer)
	if err != nil {
		logger.Error("failed to report agent", slog.String("error", err.Error()))
	}

	err = s.agentUsecase.SaveAgent(ctx, agent)
	if err != nil {
		logger.Error("failed to save agent", slog.String("error", err.Error()))
	}

	err = s.agentNotificationUsecase.NotifyAgentUpdated(ctx, agent)
	if err != nil {
		logger.Error("failed to notify agent update", slog.String("error", err.Error()))
	}

	response, err := s.fetchServerToAgent(ctx, agent)
	if err != nil {
		logger.Error("failed to fetch ServerToAgent message", slog.String("error", err.Error()))

		// whan the ServerToAgent message cannot be created, return a fallback ServerToAgent message
		return s.createFallbackServerToAgent(instanceUID)
	}

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

	s.closedConnectionCh <- conn

	logger.Info("end")
}

func (s *Service) report(
	agent *model.Agent,
	agentToServer *protobufs.AgentToServer,
	by *model.Server,
) error {
	// Update communication info
	agent.RecordLastReported(by, s.clock.Now())

	err := agent.ReportDescription(descToDomain(agentToServer.GetAgentDescription()))
	if err != nil {
		return fmt.Errorf("failed to report description: %w", err)
	}

	err = agent.ReportComponentHealth(healthToDomain(agentToServer.GetHealth()))
	if err != nil {
		return fmt.Errorf("failed to report component health: %w", err)
	}

	capabilities := agentToServer.GetCapabilities()

	err = agent.ReportCapabilities((*agentmodel.Capabilities)(&capabilities))
	if err != nil {
		return fmt.Errorf("failed to report capabilities: %w", err)
	}

	err = agent.ReportEffectiveConfig(effectiveConfigToDomain(agentToServer.GetEffectiveConfig()))
	if err != nil {
		return fmt.Errorf("failed to report effective config: %w", err)
	}

	err = agent.ReportRemoteConfigStatus(remoteConfigStatusToDomain(agentToServer.GetRemoteConfigStatus()))
	if err != nil {
		return fmt.Errorf("failed to report remote config status: %w", err)
	}

	err = agent.ReportPackageStatuses(packageStatusToDomain(agentToServer.GetPackageStatuses()))
	if err != nil {
		return fmt.Errorf("failed to report package statuses: %w", err)
	}

	err = agent.ReportCustomCapabilities(customCapabilitiesToDomain(agentToServer.GetCustomCapabilities()))
	if err != nil {
		return fmt.Errorf("failed to report custom capabilities: %w", err)
	}

	err = agent.ReportAvailableComponents(availableComponentsToDomain(agentToServer.GetAvailableComponents()))
	if err != nil {
		return fmt.Errorf("failed to report available components: %w", err)
	}

	return nil
}

func (s *Service) cleanUpConnection(ctx context.Context, conn types.Connection) error {
	connection, err := s.connectionUsecase.GetConnectionByID(ctx, conn)
	if err != nil {
		return fmt.Errorf("failed to get connection by ID: %w", err)
	}

	logger := s.logger.With(
		slog.String("method", "cleanUpConnection"),
		slog.String("connectionID", connection.IDString()),
	)
	logger.Info("start cleaning up connection")

	// Update agent connection status to disconnected
	if !connection.IsAnonymous() {
		agent, err := s.agentUsecase.GetAgent(ctx, connection.InstanceUID)
		if err != nil {
			logger.Error("failed to get agent for connection close", slog.String("error", err.Error()))
			// even if getting agent fails, proceed to delete the connection
		} else {
			agent.Status.Connected = false

			err = s.agentUsecase.SaveAgent(ctx, agent)
			if err != nil {
				logger.Error("failed to save agent connection status", slog.String("error", err.Error()))
				// even if saving fails, proceed to delete the connection
			}
		}
	}

	err = s.connectionUsecase.DeleteConnection(ctx, connection)
	if err != nil {
		return fmt.Errorf("failed to delete connection: %w", err)
	}

	return nil
}

func (s *Service) injectInstanceUIDToConnection(
	ctx context.Context,
	conn types.Connection,
	instanceUID uuid.UUID,
) error {
	connection, err := s.connectionUsecase.GetConnectionByID(ctx, conn)
	// Even if the connection is not found, we should still process the message
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}

	if connection.InstanceUID == instanceUID {
		// already injected, skip as an optimization
		return nil
	}

	connection.SetInstanceUID(instanceUID)

	err = s.connectionUsecase.SaveConnection(ctx, connection)
	if err != nil {
		return fmt.Errorf("failed to save connection with instanceUID: %w", err)
	}

	return nil
}
