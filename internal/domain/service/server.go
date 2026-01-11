package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/serverevent"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var (
	_ port.ServerUsecase = (*ServerService)(nil)
)

// ServerService is a struct that implements the ServerUsecase interface.
type ServerService struct {
	logger           *slog.Logger
	clock            clock.Clock
	heartbeatTimeout time.Duration

	cachedServers sync.Map // map[string]*model.Server

	serverPersistencePort   port.ServerPersistencePort
	serverEventSenderPort   port.ServerEventSenderPort
	serverEventReceiverPort port.ServerEventReceiverPort
	connectionUsecase       port.ConnectionUsecase
	agentUsecase            port.AgentUsecase
}

// NewServerService creates a new instance of the ServerService.
func NewServerService(
	logger *slog.Logger,
	serverPersistencePort port.ServerPersistencePort,
	serverEventSenderPort port.ServerEventSenderPort,
	serverEventReceiverPort port.ServerEventReceiverPort,
	connectionUsecase port.ConnectionUsecase,
	agentUsecase port.AgentUsecase,
) *ServerService {
	service := &ServerService{
		logger:                  logger,
		clock:                   clock.NewRealClock(),
		cachedServers:           sync.Map{},
		serverPersistencePort:   serverPersistencePort,
		serverEventSenderPort:   serverEventSenderPort,
		serverEventReceiverPort: serverEventReceiverPort,
		heartbeatTimeout:        DefaultHeartbeatTimeout,
		connectionUsecase:       connectionUsecase,
		agentUsecase:            agentUsecase,
	}

	return service
}

// Name returns the name of the runner.
func (s *ServerService) Name() string {
	return "ServerService"
}

// SetClock sets the clock for testing purposes.
func (s *ServerService) SetClock(c clock.Clock) {
	s.clock = c
}

// Run starts the server service.
func (s *ServerService) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	wg.Go(func() {
		err := s.loopForReceivingMessages(ctx)
		if err != nil {
			s.logger.Error("message receiving loop exited with error", slog.String("error", err.Error()))
		}
	})

	wg.Wait()

	return nil
}

// GetServer implements port.ServerUsecase.
func (s *ServerService) GetServer(ctx context.Context, id string) (*model.Server, error) {
	if cachedServer, ok := s.cachedServers.Load(id); ok {
		server, ok := cachedServer.(*model.Server)
		if ok && server.IsAlive(s.clock.Now(), s.heartbeatTimeout) {
			return server, nil
		}
	}

	server, err := s.serverPersistencePort.GetServer(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	s.cachedServers.Store(id, server)

	return server, nil
}

// ListServers implements port.ServerUsecase.
func (s *ServerService) ListServers(ctx context.Context) ([]*model.Server, error) {
	servers, err := s.serverPersistencePort.ListServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}

	// Filter out dead servers
	now := s.clock.Now()
	aliveServers := make([]*model.Server, 0)

	for _, server := range servers {
		if server.IsAlive(now, s.heartbeatTimeout) {
			aliveServers = append(aliveServers, server)
		}
	}

	return aliveServers, nil
}

// SendMessageToServerByServerID implements port.ServerUsecase.
func (s *ServerService) SendMessageToServerByServerID(
	ctx context.Context,
	serverID string,
	message serverevent.Message,
) error {
	server, err := s.serverPersistencePort.GetServer(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}

	err = s.SendMessageToServer(ctx, server, message)
	if err != nil {
		return fmt.Errorf("failed to send message to server %s: %w", serverID, err)
	}

	return nil
}

// SendMessageToServer sends a message to the specified server.
func (s *ServerService) SendMessageToServer(
	ctx context.Context,
	server *model.Server,
	message serverevent.Message,
) error {
	if !server.IsAlive(s.clock.Now(), s.heartbeatTimeout) {
		return fmt.Errorf("%w: server ID %s is not alive", ErrServerNotAlive, server.ID)
	}

	s.logger.Info("sending message to server",
		slog.String("serverID", server.ID),
		slog.String("messageType", message.Type.String()),
	)

	err := s.serverEventSenderPort.SendMessageToServer(ctx, server.ID, message)
	if err != nil {
		return fmt.Errorf("failed to send message to server %s: %w", server.ID, err)
	}

	return nil
}

func (s *ServerService) loopForReceivingMessages(ctx context.Context) error {
	// StartReceiver is a blocking call.
	// So, we don't need a loop here.
	err := s.serverEventReceiverPort.StartReceiver(ctx, s.handleServerEvent)
	if err != nil {
		return fmt.Errorf("failed to start server event receiver: %w", err)
	}

	return nil
}

// handleServerEvent processes a received server event and takes appropriate action.
func (s *ServerService) handleServerEvent(ctx context.Context, event *serverevent.Message) error {
	switch event.Type {
	case serverevent.MessageTypeSendServerToAgent:
		return s.handleSendServerToAgentEvent(ctx, event)
	default:
		s.logger.Warn("unknown server event type", slog.String("eventType", event.Type.String()))

		return nil
	}
}

var (
	// ErrEventPayloadNil is returned when the event payload is nil.
	ErrEventPayloadNil = errors.New("event payload is nil")
)

// handleSendServerToAgentEvent handles SendServerToAgent events by fetching agent details
// and sending serverToAgent messages via WebSocket connections.
func (s *ServerService) handleSendServerToAgentEvent(ctx context.Context, event *serverevent.Message) error {
	if event.Payload.MessageForServerToAgent == nil {
		return ErrEventPayloadNil
	}

	targetAgentUIDs := event.Payload.TargetAgentInstanceUIDs
	if len(targetAgentUIDs) == 0 {
		s.logger.Warn("no target agents specified in SendServerToAgent event")

		return nil
	}

	s.logger.Info("handling SendServerToAgent event",
		slog.Int("targetAgentCount", len(targetAgentUIDs)))

	for _, instanceUID := range targetAgentUIDs {
		err := s.sendServerToAgentForInstance(ctx, instanceUID)
		if err != nil {
			s.logger.Error("failed to send ServerToAgent message",
				slog.String("instanceUID", instanceUID.String()),
				slog.String("error", err.Error()))
			// Continue processing other agents even if one fails
			continue
		}

		s.logger.Info("successfully sent ServerToAgent message",
			slog.String("instanceUID", instanceUID.String()))
	}

	return nil
}

// sendServerToAgentForInstance sends a serverToAgent message to a specific agent instance.
func (s *ServerService) sendServerToAgentForInstance(ctx context.Context, instanceUID uuid.UUID) error {
	// Get the agent to fetch current state and build the ServerToAgent message
	agent, err := s.agentUsecase.GetAgent(ctx, instanceUID)
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// Build the ServerToAgent message based on agent's current state
	// This should include any pending commands, config updates, etc.
	serverToAgentMessage := s.buildServerToAgentMessage(agent)

	// Send the message via the connection service
	err = s.connectionUsecase.SendServerToAgent(ctx, instanceUID, serverToAgentMessage)
	if err != nil {
		return fmt.Errorf("failed to send message via connection: %w", err)
	}

	return nil
}

// buildServerToAgentMessage builds a ServerToAgent message for the given agent.
// This is a simplified version - you may want to use the opamp service's fetchServerToAgent instead.
func (s *ServerService) buildServerToAgentMessage(agent *model.Agent) *protobufs.ServerToAgent {
	var flags uint64

	// Request ReportFullState if needed
	if agent.NeedFullStateCommand() {
		flags |= uint64(protobufs.ServerToAgentFlags_ServerToAgentFlags_ReportFullState)
	}

	instanceUID := agent.Metadata.InstanceUID

	//exhaustruct:ignore
	return &protobufs.ServerToAgent{
		InstanceUid: instanceUID[:],
		Flags:       flags,
		// Note: RemoteConfig building is omitted here for simplicity
		// In production, you should fetch agent groups and build config properly
	}
}
