// Package opamp provides the implementation of the OpAMP use case for managing connections and agents.
package opamp

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server/types"

	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	modelagent "github.com/minuk-dev/opampcommander/internal/domain/agent/model/agent"
	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

const (
	// DefaultOnConnectionCloseTimeout is the default timeout for closing a connection.
	DefaultOnConnectionCloseTimeout = 5 * time.Second
	// DefaultHeartbeatSaveThrottle is the minimum interval between persisting agent state
	// when the incoming message is heartbeat-only (no field reports). Bursts of heartbeats
	// share a single MongoDB write; non-heartbeat messages are always persisted immediately.
	//
	// Kept short (10s) so the API surface still reflects per-agent state — SequenceNum,
	// LastReportedAt — within an order of magnitude of the OpAMP heartbeat interval.
	// Bigger values would amortise more aggressively but the API would lag enough to
	// be observably stale (e2e SequenceNum test relies on increment visibility).
	DefaultHeartbeatSaveThrottle = 10 * time.Second
	// DefaultLastSaveAtGCInterval is how often the lastSaveAt map is swept for stale
	// entries. HTTP-polling agents never trigger the WebSocket-only Delete path on
	// connection close, so their entries would accumulate forever without this sweep.
	DefaultLastSaveAtGCInterval = 5 * time.Minute
	// DefaultLastSaveAtTTL is the age at which a lastSaveAt entry is considered stale
	// and eligible for GC. Set generously above the throttle window so we never evict
	// a live agent's entry mid-throttle.
	DefaultLastSaveAtTTL = 30 * time.Minute
)

// Service is a struct that implements the OpAMPUsecase interface.
type Service struct {
	clock                    clock.Clock
	logger                   *slog.Logger
	agentUsecase             agentport.AgentUsecase
	agentGroupUsecase        agentport.AgentGroupUsecase
	agentPackageUsecase      agentport.AgentPackageUsecase
	agentRemoteConfigUsecase agentport.AgentRemoteConfigUsecase
	serverIdentityProvider   agentport.ServerIdentityProvider

	agentNotificationUsecase agentport.AgentNotificationUsecase

	closedConnectionCh chan types.Connection

	connectionUsecase        agentport.ConnectionUsecase
	onConnectionCloseTimeout time.Duration

	heartbeatSaveThrottle time.Duration
	lastSaveAt            sync.Map // instanceUID(string) -> time.Time
	lastSaveAtGCInterval  time.Duration
	lastSaveAtTTL         time.Duration
}

// New creates a new instance of the OpAMP service.
func New(
	agentUsecase agentport.AgentUsecase,
	connectionUsecase agentport.ConnectionUsecase,
	serverIdentityProvider agentport.ServerIdentityProvider,
	agentGroupUsecase agentport.AgentGroupUsecase,
	agentNotificationUsecase agentport.AgentNotificationUsecase,
	agentPackageUsecase agentport.AgentPackageUsecase,
	agentRemoteConfigUsecase agentport.AgentRemoteConfigUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		clock:                    clock.NewRealClock(),
		logger:                   logger,
		agentUsecase:             agentUsecase,
		connectionUsecase:        connectionUsecase,
		serverIdentityProvider:   serverIdentityProvider,
		agentGroupUsecase:        agentGroupUsecase,
		agentNotificationUsecase: agentNotificationUsecase,
		agentPackageUsecase:      agentPackageUsecase,
		agentRemoteConfigUsecase: agentRemoteConfigUsecase,
		closedConnectionCh:       make(chan types.Connection, 1), // buffered channel

		onConnectionCloseTimeout: DefaultOnConnectionCloseTimeout,
		heartbeatSaveThrottle:    DefaultHeartbeatSaveThrottle,
		lastSaveAt:               sync.Map{},
		lastSaveAtGCInterval:     DefaultLastSaveAtGCInterval,
		lastSaveAtTTL:            DefaultLastSaveAtTTL,
	}
}

// Name returns the name of the service.
func (s *Service) Name() string {
	return "opamp"
}

// Run starts a loop to handle asynchronous operations for the service.
func (s *Service) Run(ctx context.Context) error {
	// Run GC on its own goroutine so a long sweep (lastSaveAt can hold hundreds of
	// thousands of entries at scale) never delays connection-close cleanup on the
	// main loop. Both goroutines exit on ctx cancellation.
	go s.runLastSaveAtGC(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("context done, exiting service loop")

			return fmt.Errorf("service loop exited: %w", ctx.Err())
		case conn := <-s.closedConnectionCh:
			bgCtx, cancel := context.WithTimeout(ctx, s.onConnectionCloseTimeout)
			err := s.cleanUpConnection(bgCtx, conn)

			cancel()

			if err != nil {
				s.logger.Error("failed to clean up connection", slog.String("error", err.Error()))
			}
		}
	}
}

// OnConnected implements port.OpAMPUsecase.
//
// Deprecated: Use OnConnectedWithType instead for proper connection type detection.
func (s *Service) OnConnected(ctx context.Context, conn types.Connection) {
	// Default to unknown type for backward compatibility
	s.OnConnectedWithType(ctx, conn, false)
}

// OnConnectedWithType implements port.OpAMPUsecase.
// This is called for both WebSocket and HTTP connections.
// isWebSocket parameter indicates the connection type.
func (s *Service) OnConnectedWithType(ctx context.Context, conn types.Connection, isWebSocket bool) {
	remoteAddr := conn.Connection().RemoteAddr().String()
	logger := s.logger.With(
		slog.String("method", "OnConnectedWithType"),
		slog.String("remoteAddr", remoteAddr),
		slog.Bool("isWebSocket", isWebSocket),
	)

	logger.Info("start")

	// Create connection with the correct type
	connectionType := agentmodel.ConnectionTypeHTTP
	if isWebSocket {
		connectionType = agentmodel.ConnectionTypeWebSocket
	}

	connection := agentmodel.NewConnection(conn, connectionType)

	err := s.connectionUsecase.SaveConnection(ctx, connection)
	if err != nil {
		logger.Error("failed to save connection",
			slog.String("connectionUID", connection.UID.String()),
			slog.String("error", err.Error()),
		)

		return
	}

	logger.Info("end successfully",
		slog.String("connectionUID", connection.UID.String()),
		slog.String("connectionType", connectionType.String()),
	)
}

// OnMessage implements port.OpAMPUsecase.
// [1] find agentmodel.Connection by types.Connection
// [1-1] if not found, unexpected case because all connections should be created when OnConnected is called.
// so, leave error log and skip connection processing.
// [2] find agentmodel.Agent by instanceUID in message
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

	if response := s.handleInstanceUIDConflict(ctx, logger, conn, instanceUID, message); response != nil {
		return response
	}

	connection, logger := s.prepareConnection(ctx, logger, conn, instanceUID)

	currentServer, err := s.serverIdentityProvider.CurrentServer(ctx)
	if err != nil {
		logger.Warn("failed to get current server", slog.String("error", err.Error()))
	}

	agent, err := s.agentUsecase.GetOrCreateAgent(ctx, instanceUID)
	if err != nil {
		logger.Error("failed to get agent", slog.String("error", err.Error()))

		// whan the agent cannot be retrieved, return a fallback ServerToAgent message
		return s.createFallbackServerToAgent(instanceUID)
	}

	s.syncConnectionNamespace(ctx, logger, connection, agent)

	// Capture a single timestamp so the throttle window is anchored on message
	// arrival, not on SaveAgent return — Mongo latency spikes would otherwise
	// push the next throttle boundary out by the write duration.
	receivedAt := s.clock.Now()

	// Update agent connection status
	agent.UpdateLastCommunicationInfo(receivedAt, connection)

	s.reportAndReconcileGroups(ctx, logger, message, agent, currentServer)

	s.maybePersistAgent(ctx, logger, instanceUID, message, agent, receivedAt)

	// Note: NotifyAgentUpdated is NOT called here to avoid infinite loop.
	// OnMessage already sends a response via fetchServerToAgent.
	// NotifyAgentUpdated should only be called when agent is updated externally (e.g., via API).

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

	select {
	case s.closedConnectionCh <- conn:
	default:
		logger.Warn("closedConnectionCh is full, skipping cleanup for this connection")
	}

	logger.Info("end")
}

// gcLastSaveAt removes lastSaveAt entries older than lastSaveAtTTL. Entries for
// HTTP-polling agents are never cleared by cleanUpConnection (WebSocket-only),
// so this sweep is what bounds the map's footprint when HTTP agents go away
// without explicit teardown.
func (s *Service) gcLastSaveAt() {
	cutoff := s.clock.Now().Add(-s.effectiveLastSaveAtTTL())
	removed := 0

	s.lastSaveAt.Range(func(key, val any) bool {
		lastAt, isTime := val.(time.Time)
		if !isTime || lastAt.Before(cutoff) {
			s.lastSaveAt.Delete(key)

			removed++
		}

		return true
	})

	if removed > 0 {
		s.logger.Debug("garbage-collected stale lastSaveAt entries",
			slog.Int("removed", removed),
			slog.Duration("ttl", s.effectiveLastSaveAtTTL()),
		)
	}
}

// runLastSaveAtGC ticks gcLastSaveAt on its own goroutine until ctx is done.
func (s *Service) runLastSaveAtGC(ctx context.Context) {
	gcTicker := time.NewTicker(s.effectiveLastSaveAtGCInterval())
	defer gcTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-gcTicker.C:
			s.gcLastSaveAt()
		}
	}
}

// effectiveLastSaveAtGCInterval returns the configured GC interval, or the
// default when zero. Reading via a helper keeps Run side-effect-free for tests
// that build *Service via struct literal without invoking New().
func (s *Service) effectiveLastSaveAtGCInterval() time.Duration {
	if s.lastSaveAtGCInterval <= 0 {
		return DefaultLastSaveAtGCInterval
	}

	return s.lastSaveAtGCInterval
}

// effectiveLastSaveAtTTL returns the configured GC TTL, or the default when zero.
func (s *Service) effectiveLastSaveAtTTL() time.Duration {
	if s.lastSaveAtTTL <= 0 {
		return DefaultLastSaveAtTTL
	}

	return s.lastSaveAtTTL
}

func (s *Service) report(
	agent *agentmodel.Agent,
	agentToServer *protobufs.AgentToServer,
	by *agentmodel.Server,
) error {
	// Update communication info
	agent.RecordLastReported(by, s.clock.Now(), agentToServer.GetSequenceNum())

	err := agent.ReportDescription(descToDomain(agentToServer.GetAgentDescription()))
	if err != nil {
		return fmt.Errorf("failed to report description: %w", err)
	}

	err = agent.ReportComponentHealth(healthToDomain(agentToServer.GetHealth()))
	if err != nil {
		return fmt.Errorf("failed to report component health: %w", err)
	}

	capabilities := agentToServer.GetCapabilities()

	err = agent.ReportCapabilities((*modelagent.Capabilities)(&capabilities))
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

	err = agent.ReportConnectionSettingsStatus(
		connectionSettingsStatusToDomain(agentToServer.GetConnectionSettingsStatus()))
	if err != nil {
		return fmt.Errorf("failed to report connection settings status: %w", err)
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
	remoteAddr := conn.Connection().RemoteAddr().String()

	connection, err := s.connectionUsecase.GetConnectionByID(ctx, conn)
	if err != nil {
		s.logger.Error("failed to get connection by ID during cleanup",
			slog.String("method", "cleanUpConnection"),
			slog.String("remoteAddr", remoteAddr),
			slog.String("error", err.Error()),
		)

		return fmt.Errorf("failed to get connection by ID: %w", err)
	}

	logger := s.logger.With(
		slog.String("method", "cleanUpConnection"),
		slog.String("remoteAddr", remoteAddr),
		slog.String("connectionUID", connection.UID.String()),
		slog.String("instanceUID", connection.InstanceUID.String()),
		slog.String("connectionType", connection.Type.String()),
	)
	logger.Info("start cleaning up connection")

	// Update agent connection status to disconnected.
	//
	// Only WebSocket close represents a genuine disconnect. HTTP polling agents fire
	// OnConnectionClose after every request; treating those as disconnects would both
	// (a) flip agent.Status.Connected on every poll, and (b) defeat the heartbeat-save
	// throttle by writing to MongoDB on every request.
	if !connection.IsAnonymous() && connection.Type == agentmodel.ConnectionTypeWebSocket {
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

	// WebSocket close is a genuine disconnect — clear the throttle entry so the first
	// message after reconnect is persisted immediately. HTTP polling agents do not get
	// here because their close is treated as request-end, not disconnect.
	if !connection.IsAnonymous() && connection.Type == agentmodel.ConnectionTypeWebSocket {
		s.lastSaveAt.Delete(connection.InstanceUID.String())
	}

	return nil
}

// prepareConnection resolves the agentmodel.Connection for the incoming network connection,
// injects the instanceUID, and decorates the logger with connection-scoped fields. Errors
// are logged and the caller is expected to continue without the connection if it is nil.
func (s *Service) prepareConnection(
	ctx context.Context,
	logger *slog.Logger,
	conn types.Connection,
	instanceUID uuid.UUID,
) (*agentmodel.Connection, *slog.Logger) {
	connection, err := s.injectInstanceUIDToConnection(ctx, conn, instanceUID)
	if err != nil {
		logger.Error("failed to inject instanceUID to connection", slog.String("error", err.Error()))
	}

	if connection != nil {
		logger = logger.With(
			slog.String("connectionUID", connection.UID.String()),
			slog.String("connectionType", connection.Type.String()),
		)
	}

	return connection, logger
}

func (s *Service) injectInstanceUIDToConnection(
	ctx context.Context,
	conn types.Connection,
	instanceUID uuid.UUID,
) (*agentmodel.Connection, error) {
	connection, err := s.connectionUsecase.GetConnectionByID(ctx, conn)
	// Even if the connection is not found, we should still process the message
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	if connection.InstanceUID == instanceUID {
		// already injected, skip as an optimization
		return connection, nil
	}

	connection.SetInstanceUID(instanceUID)

	err = s.connectionUsecase.SaveConnection(ctx, connection)
	if err != nil {
		return nil, fmt.Errorf("failed to save connection with instanceUID: %w", err)
	}

	return connection, nil
}

// shouldPersistAgent decides whether the agent's in-memory state should be flushed to
// the datastore for this incoming message. Non-heartbeat messages (carrying any
// reported field) are always persisted. For heartbeat-only messages — which dominate
// the volume at scale — persistence is throttled per agent to amortise writes.
func (s *Service) shouldPersistAgent(instanceUID uuid.UUID, message *protobufs.AgentToServer) bool {
	if !isHeartbeatOnly(message) {
		return true
	}

	last, found := s.lastSaveAt.Load(instanceUID.String())
	if !found {
		return true
	}

	lastAt, isTime := last.(time.Time)
	if !isTime {
		return true
	}

	return s.clock.Now().Sub(lastAt) >= s.heartbeatSaveThrottle
}

// isHeartbeatOnly reports whether the AgentToServer message carries no reported field
// updates beyond identification. The fixed Capabilities bitfield is intentionally
// excluded — agents include it on every message even when nothing has changed.
func isHeartbeatOnly(msg *protobufs.AgentToServer) bool {
	if msg == nil {
		return true
	}

	return msg.GetAgentDescription() == nil &&
		msg.GetHealth() == nil &&
		msg.GetEffectiveConfig() == nil &&
		msg.GetRemoteConfigStatus() == nil &&
		msg.GetConnectionSettingsStatus() == nil &&
		msg.GetPackageStatuses() == nil &&
		msg.GetCustomCapabilities() == nil &&
		msg.GetCustomMessage() == nil &&
		msg.GetAvailableComponents() == nil &&
		msg.GetAgentDisconnect() == nil &&
		msg.GetConnectionSettingsRequest() == nil &&
		msg.GetFlags() == 0
}

// identitySnapshot captures the fields of an agent that determine which agent groups
// match it. We compare these before and after applying the incoming AgentToServer
// message to decide whether to re-evaluate matching agent groups.
type identitySnapshot struct {
	namespace                string
	identifyingAttributes    map[string]string
	nonIdentifyingAttributes map[string]string
}

func snapshotIdentity(agent *agentmodel.Agent) identitySnapshot {
	return identitySnapshot{
		namespace:                agent.Metadata.Namespace,
		identifyingAttributes:    maps.Clone(agent.Metadata.Description.IdentifyingAttributes),
		nonIdentifyingAttributes: maps.Clone(agent.Metadata.Description.NonIdentifyingAttributes),
	}
}

func identityChanged(prev identitySnapshot, agent *agentmodel.Agent) bool {
	if prev.namespace != agent.Metadata.Namespace {
		return true
	}

	if !maps.Equal(prev.identifyingAttributes, agent.Metadata.Description.IdentifyingAttributes) {
		return true
	}

	return !maps.Equal(prev.nonIdentifyingAttributes, agent.Metadata.Description.NonIdentifyingAttributes)
}

// reportAndReconcileGroups absorbs incoming AgentToServer reports into the agent and
// re-evaluates which agent groups apply when the description changed. The identity
// snapshot/compare is skipped unless the incoming message carries an AgentDescription —
// that is the only report() input that can affect AgentGroup selectors, and skipping
// the snapshot avoids two map allocations on every heartbeat plus a full ListAgentGroups
// scan when agents put monotonic counters under NonIdentifyingAttributes.
func (s *Service) reportAndReconcileGroups(
	ctx context.Context,
	logger *slog.Logger,
	message *protobufs.AgentToServer,
	agent *agentmodel.Agent,
	currentServer *agentmodel.Server,
) {
	hasDescription := message.GetAgentDescription() != nil

	var prevIdentity identitySnapshot
	if hasDescription {
		prevIdentity = snapshotIdentity(agent)
	}

	err := s.report(agent, message, currentServer)
	if err != nil {
		logger.Error("failed to report agent", slog.String("error", err.Error()))
	}

	if hasDescription {
		s.maybeApplyMatchingAgentGroups(ctx, logger, agent, prevIdentity)
	}
}

// maybePersistAgent writes the agent through the throttle if the message warrants it,
// updating the lastSaveAt anchor on success so the next throttle window is measured
// from this arrival time.
func (s *Service) maybePersistAgent(
	ctx context.Context,
	logger *slog.Logger,
	instanceUID uuid.UUID,
	message *protobufs.AgentToServer,
	agent *agentmodel.Agent,
	receivedAt time.Time,
) {
	if !s.shouldPersistAgent(instanceUID, message) {
		return
	}

	err := s.agentUsecase.SaveAgent(ctx, agent)
	if err != nil {
		logger.Error("failed to save agent", slog.String("error", err.Error()))

		return
	}

	s.lastSaveAt.Store(instanceUID.String(), receivedAt)
}

// maybeApplyMatchingAgentGroups re-evaluates which agent groups apply to the agent when
// its identity changed after processing the incoming AgentToServer message. This is the
// trigger that picks up groups for a newly-described agent without waiting for either a
// group update or the periodic reconciler.
func (s *Service) maybeApplyMatchingAgentGroups(
	ctx context.Context,
	logger *slog.Logger,
	agent *agentmodel.Agent,
	prev identitySnapshot,
) {
	if !identityChanged(prev, agent) {
		return
	}

	err := s.agentGroupUsecase.ApplyMatchingAgentGroupsToAgent(ctx, agent)
	if err != nil {
		logger.Warn("failed to apply matching agent groups", slog.String("error", err.Error()))
	}
}

func (s *Service) syncConnectionNamespace(
	ctx context.Context,
	logger *slog.Logger,
	connection *agentmodel.Connection,
	agent *agentmodel.Agent,
) {
	if connection == nil || agent == nil {
		return
	}

	if connection.Namespace == agent.Metadata.Namespace {
		return
	}

	connection.SetNamespace(agent.Metadata.Namespace)

	err := s.connectionUsecase.SaveConnection(ctx, connection)
	if err != nil {
		logger.Error("failed to sync connection namespace",
			slog.String("error", err.Error()))
	}
}
