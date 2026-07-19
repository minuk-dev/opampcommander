package agentservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jellydator/ttlcache/v3"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var (
	_ agentport.ServerUsecase                   = (*ServerService)(nil)
	_ agentport.LeaderElector                   = (*ServerService)(nil)
	_ agentport.AgentCacheInvalidationPublisher = (*ServerService)(nil)

	// ErrNoCurrentServerID is returned by IsLeader when the current server has no
	// identity, so leadership cannot be determined.
	ErrNoCurrentServerID = errors.New("current server has no identity")
)

const (
	// DefaultServerCacheTTL is the default time-to-live for server cache entries.
	DefaultServerCacheTTL = 30 * time.Second
	// DefaultServerCacheCapacity is the default maximum number of server cache entries.
	DefaultServerCacheCapacity = 100
)

// ServerService is a struct that implements the ServerUsecase interface.
type ServerService struct {
	logger           *slog.Logger
	clock            clock.Clock
	heartbeatTimeout time.Duration

	serverCache *ttlcache.Cache[string, *agentmodel.Server]

	serverPersistencePort   agentport.ServerPersistencePort
	serverEventSenderPort   agentport.ServerEventSenderPort
	serverEventReceiverPort agentport.ServerEventReceiverPort
	serverIdentityProvider  agentport.ServerIdentityProvider
	connectionUsecase       agentport.ConnectionUsecase
	agentUsecase            agentport.AgentUsecase
	agentCacheInvalidator   agentport.AgentCacheInvalidator
	serverToAgentBuilder    *ServerToAgentBuilder
}

// NewServerService creates a new instance of the ServerService.
func NewServerService(
	logger *slog.Logger,
	serverPersistencePort agentport.ServerPersistencePort,
	serverEventSenderPort agentport.ServerEventSenderPort,
	serverEventReceiverPort agentport.ServerEventReceiverPort,
	serverIdentityProvider agentport.ServerIdentityProvider,
	connectionUsecase agentport.ConnectionUsecase,
	agentUsecase agentport.AgentUsecase,
	agentCacheInvalidator agentport.AgentCacheInvalidator,
	serverToAgentBuilder *ServerToAgentBuilder,
) *ServerService {
	serverCache := ttlcache.New[string, *agentmodel.Server](
		ttlcache.WithTTL[string, *agentmodel.Server](DefaultServerCacheTTL),
		ttlcache.WithCapacity[string, *agentmodel.Server](DefaultServerCacheCapacity),
	)

	logger.Info("server cache initialized",
		slog.Duration("ttl", DefaultServerCacheTTL),
		slog.Int64("maxCapacity", DefaultServerCacheCapacity),
	)

	return &ServerService{
		logger:                  logger,
		clock:                   clock.NewRealClock(),
		serverCache:             serverCache,
		serverPersistencePort:   serverPersistencePort,
		serverEventSenderPort:   serverEventSenderPort,
		serverEventReceiverPort: serverEventReceiverPort,
		serverIdentityProvider:  serverIdentityProvider,
		heartbeatTimeout:        DefaultHeartbeatTimeout,
		connectionUsecase:       connectionUsecase,
		agentUsecase:            agentUsecase,
		agentCacheInvalidator:   agentCacheInvalidator,
		serverToAgentBuilder:    serverToAgentBuilder,
	}
}

// Name returns the name of the runner.
func (s *ServerService) Name() string {
	return "ServerService"
}

// SetClock sets the clock for testing purposes.
func (s *ServerService) SetClock(c clock.Clock) {
	s.clock = c
}

// Shutdown releases resources held by the service.
// This should be called during graceful shutdown.
func (s *ServerService) Shutdown() {
	s.logger.Info("shutting down server service, clearing cache")
	s.serverCache.DeleteAll()
	s.serverCache.Stop()
}

// Run starts the server service.
//
// The message-receiving loop runs directly: Run is already invoked on its own goroutine
// by the executor, so wrapping the single blocking loop in a WaitGroup only added another
// goroutine that parked on Wait. The loop returns on ctx cancellation.
func (s *ServerService) Run(ctx context.Context) error {
	err := s.loopForReceivingMessages(ctx)
	if err != nil {
		s.logger.Error("message receiving loop exited with error", slog.String("error", err.Error()))
	}

	return nil
}

// GetServer implements agentport.ServerUsecase.
func (s *ServerService) GetServer(ctx context.Context, id string) (*agentmodel.Server, error) {
	cachedServer, ok := s.getCachedServer(id)
	if ok {
		return cachedServer, nil
	}

	server, err := s.serverPersistencePort.GetServer(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	s.updateCachedServer(server)

	return server, nil
}

// ListServers implements agentport.ServerUsecase.
func (s *ServerService) ListServers(ctx context.Context) ([]*agentmodel.Server, error) {
	servers, err := s.serverPersistencePort.ListServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}

	// Filter out dead servers
	now := s.clock.Now()
	aliveServers := make([]*agentmodel.Server, 0)

	for _, server := range servers {
		if server.IsAlive(now, s.heartbeatTimeout) {
			aliveServers = append(aliveServers, server)
		}
	}

	return aliveServers, nil
}

// IsLeader implements agentport.LeaderElector.
//
// Leadership is deterministic and requires no extra writes or lease document: the
// leader is the alive server with the smallest ID. Each node already publishes a
// heartbeat (ServerIdentityService), so ListServers' alive set is the single source
// of truth. When no server is registered yet (cold start) the current node leads so
// singleton work is not stranded. A brief dual-leader window during a heartbeat-timeout
// transition is acceptable: the only leader-gated job (agent-group reconcile) is
// idempotent.
func (s *ServerService) IsLeader(ctx context.Context) (bool, error) {
	currentID := s.serverIdentityProvider.CurrentServerID()
	if currentID == "" {
		return false, ErrNoCurrentServerID
	}

	servers, err := s.ListServers(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to list servers for leader election: %w", err)
	}

	if len(servers) == 0 {
		return true, nil
	}

	for _, server := range servers {
		if server.ID < currentID {
			return false, nil
		}
	}

	return true, nil
}

// SendMessageToServerByServerID implements agentport.ServerUsecase.
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
//
// If the target server is the current server, the message is dispatched in-process
// instead of going through the messaging backend (Kafka), avoiding a needless round-trip.
func (s *ServerService) SendMessageToServer(
	ctx context.Context,
	server *agentmodel.Server,
	message serverevent.Message,
) error {
	if !server.IsAlive(s.clock.Now(), s.heartbeatTimeout) {
		return fmt.Errorf("%w: server ID %s is not alive", ErrServerNotAlive, server.ID)
	}

	if s.serverIdentityProvider != nil && server.ID == s.serverIdentityProvider.CurrentServerID() {
		s.logger.Debug("dispatching message locally (target is current server)",
			slog.String("serverID", server.ID),
			slog.String("messageType", message.Type.String()),
		)

		err := s.handleServerEvent(ctx, &message)
		if err != nil {
			return fmt.Errorf("failed to dispatch local message: %w", err)
		}

		return nil
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

// BroadcastAgentCacheInvalidation implements agentport.AgentCacheInvalidationPublisher.
//
// It asks every other alive server to drop the listed agents from its cache. The current
// server is skipped (its cache was already refreshed by the write that triggered this).
// Delivery is best-effort and per-peer: a failure to reach one peer is logged and does not
// stop the others or fail the call, because the stale entry expires within the cache TTL
// regardless.
func (s *ServerService) BroadcastAgentCacheInvalidation(
	ctx context.Context,
	instanceUIDs ...uuid.UUID,
) error {
	if len(instanceUIDs) == 0 {
		return nil
	}

	servers, err := s.ListServers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list servers for cache invalidation: %w", err)
	}

	currentID := ""
	if s.serverIdentityProvider != nil {
		currentID = s.serverIdentityProvider.CurrentServerID()
	}

	for _, server := range servers {
		if server.ID == currentID {
			continue
		}

		message := serverevent.Message{
			Source: currentID,
			Target: server.ID,
			Type:   serverevent.MessageTypeInvalidateAgentCache,
			Payload: serverevent.MessagePayload{
				MessageForServerToAgent: nil,
				MessageForInvalidateAgentCache: &serverevent.MessageForInvalidateAgentCache{
					AgentInstanceUIDs: instanceUIDs,
				},
			},
		}

		sendErr := s.SendMessageToServer(ctx, server, message)
		if sendErr != nil {
			s.logger.Warn("failed to broadcast cache invalidation to peer",
				slog.String("peerServerID", server.ID),
				slog.String("error", sendErr.Error()))
		}
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
	case serverevent.MessageTypeInvalidateAgentCache:
		return s.handleInvalidateAgentCacheEvent(event)
	default:
		s.logger.Warn("unknown server event type", slog.String("eventType", event.Type.String()))

		return nil
	}
}

// handleInvalidateAgentCacheEvent drops the listed agents from this server's local cache
// so a write made on another node is not served stale from here. It never errors: a
// missing/empty payload is logged and ignored, since a stale entry expires on its own.
func (s *ServerService) handleInvalidateAgentCacheEvent(event *serverevent.Message) error {
	if event.Payload.MessageForInvalidateAgentCache == nil {
		s.logger.Warn("invalidate-agent-cache event has no payload")

		return nil
	}

	uids := event.Payload.AgentInstanceUIDs
	if len(uids) == 0 {
		return nil
	}

	for _, instanceUID := range uids {
		s.agentCacheInvalidator.InvalidateCache(instanceUID)
	}

	s.logger.Debug("invalidated agents from local cache on peer request",
		slog.Int("agentCount", len(uids)))

	return nil
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

	// Build the full ServerToAgent message from the agent's current state (remote config,
	// packages, connection settings, pending commands) via the shared builder — the same one
	// the OpAMP hot path uses — so a cross-server push delivers the same message as a direct
	// response rather than a degraded stub.
	serverToAgentMessage := s.serverToAgentBuilder.Build(ctx, agent)

	// Send the message via the connection service
	err = s.connectionUsecase.SendServerToAgent(ctx, instanceUID, serverToAgentMessage)
	if err != nil {
		return fmt.Errorf("failed to send message via connection: %w", err)
	}

	return nil
}

func (s *ServerService) getCachedServer(id string) (*agentmodel.Server, bool) {
	item := s.serverCache.Get(id)
	if item == nil {
		return nil, false
	}

	server := item.Value()
	if !server.IsAlive(s.clock.Now(), s.heartbeatTimeout) {
		s.invalidateCachedServer(id)

		return nil, false
	}

	return server.Clone(), true
}

func (s *ServerService) invalidateCachedServer(id string) {
	s.serverCache.Delete(id)
}

func (s *ServerService) updateCachedServer(server *agentmodel.Server) {
	s.serverCache.Set(server.ID, server, ttlcache.DefaultTTL)
}
