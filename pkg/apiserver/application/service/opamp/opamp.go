// Package opamp provides the implementation of the OpAMP use case for managing connections and agents.
package opamp

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server/types"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/usecase"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	modelagent "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ usecase.OpAMPUsecase = (*Service)(nil)

const (
	// DefaultOnConnectionCloseTimeout is the default timeout for closing a connection.
	DefaultOnConnectionCloseTimeout = 5 * time.Second
)

// Service is a struct that implements the OpAMPUsecase interface.
type Service struct {
	clock                    clock.Clock
	logger                   *slog.Logger
	agentUsecase             agentport.AgentUsecase
	agentGroupUsecase        agentport.AgentGroupUsecase
	agentRemoteConfigUsecase agentport.AgentRemoteConfigUsecase
	hostUsecase              agentport.HostUsecase
	containerUsecase         agentport.ContainerUsecase
	serverIdentityProvider   agentport.ServerIdentityProvider
	serverToAgentBuilder     *agentservice.ServerToAgentBuilder

	agentNotificationUsecase agentport.AgentNotificationUsecase

	customMessageRegistry *CustomMessageRegistry

	closedConnectionCh chan types.Connection

	connectionUsecase        agentport.ConnectionUsecase
	onConnectionCloseTimeout time.Duration
}

// New creates a new instance of the OpAMP service.
func New(
	agentUsecase agentport.AgentUsecase,
	connectionUsecase agentport.ConnectionUsecase,
	serverIdentityProvider agentport.ServerIdentityProvider,
	agentGroupUsecase agentport.AgentGroupUsecase,
	agentNotificationUsecase agentport.AgentNotificationUsecase,
	serverToAgentBuilder *agentservice.ServerToAgentBuilder,
	agentRemoteConfigUsecase agentport.AgentRemoteConfigUsecase,
	hostUsecase agentport.HostUsecase,
	containerUsecase agentport.ContainerUsecase,
	customMessageRegistry *CustomMessageRegistry,
	logger *slog.Logger,
) *Service {
	return &Service{
		clock:                    clock.NewRealClock(),
		logger:                   logger,
		agentUsecase:             agentUsecase,
		connectionUsecase:        connectionUsecase,
		serverIdentityProvider:   serverIdentityProvider,
		serverToAgentBuilder:     serverToAgentBuilder,
		agentGroupUsecase:        agentGroupUsecase,
		agentNotificationUsecase: agentNotificationUsecase,
		agentRemoteConfigUsecase: agentRemoteConfigUsecase,
		hostUsecase:              hostUsecase,
		containerUsecase:         containerUsecase,
		customMessageRegistry:    customMessageRegistry,
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
			err := s.cleanUpConnection(bgCtx, conn)

			cancel()

			if err != nil {
				s.logger.Error("failed to clean up connection", slog.String("error", err.Error()))
			}
		}
	}
}

// OnConnected implements usecase.OpAMPUsecase.
//
// Deprecated: Use OnConnectedWithType instead for proper connection type detection.
func (s *Service) OnConnected(ctx context.Context, conn types.Connection) {
	// Default to unknown type for backward compatibility
	s.OnConnectedWithType(ctx, conn, false)
}

// OnConnectedWithType implements usecase.OpAMPUsecase.
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

// OnMessage implements usecase.OpAMPUsecase.
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

		// The server could not load the agent's state, so it cannot process this message.
		// Signal it as Unavailable (a transient server-side failure) so the agent retries.
		return s.createErrorServerToAgent(instanceUID,
			protobufs.ServerErrorResponseType_ServerErrorResponseType_Unavailable,
			"failed to load agent state")
	}

	s.syncConnectionNamespace(ctx, logger, connection, agent)

	// Capture a single timestamp so the throttle window is anchored on message
	// arrival, not on SaveAgent return — Mongo latency spikes would otherwise
	// push the next throttle boundary out by the write duration.
	receivedAt := s.clock.Now()

	// Update agent connection status
	agent.UpdateLastCommunicationInfo(receivedAt, connection)

	reportErr := s.reportAndReconcileGroups(ctx, logger, message, agent, currentServer)
	if reportErr != nil {
		// The agent's report could not be absorbed into its state. Return an error-only
		// response (BadRequest) rather than a desired-state message the agent would ignore,
		// and skip persistence so partially-applied state is not written.
		return s.createErrorServerToAgent(instanceUID,
			protobufs.ServerErrorResponseType_ServerErrorResponseType_BadRequest,
			reportErr.Error())
	}

	s.maybePersistAgent(ctx, logger, message, agent, receivedAt)

	// Note: NotifyAgentUpdated is NOT called here to avoid infinite loop.
	// OnMessage already sends a response via fetchServerToAgent.
	// NotifyAgentUpdated should only be called when agent is updated externally (e.g., via API).

	response := s.fetchServerToAgent(ctx, agent)

	if reply := s.handleInboundCustomMessage(ctx, logger, agent, message.GetCustomMessage()); reply != nil {
		response.CustomMessage = reply
	}

	logger.Info("end successfully")

	return response
}

// OnReadMessageError implements usecase.OpAMPUsecase.
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

// OnMessageResponseError implements usecase.OpAMPUsecase.
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

// OnConnectionClose implements usecase.OpAMPUsecase.
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

// handleInboundCustomMessage routes an inbound custom_message to the handler registered for its
// custom capability and returns the handler's optional reply as a wire message to include in the
// same ServerToAgent response, or nil for no reply.
//
// It returns nil (dropping the message) when: there is no custom_message, no registry, no handler
// for the capability, the handler errored, or the agent has not advertised the reply's capability
// (the OpAMP spec forbids sending a custom_message for a capability the agent did not advertise).
func (s *Service) handleInboundCustomMessage(
	ctx context.Context,
	logger *slog.Logger,
	agent *agentmodel.Agent,
	msg *protobufs.CustomMessage,
) *protobufs.CustomMessage {
	if msg == nil || s.customMessageRegistry == nil {
		return nil
	}

	capability := msg.GetCapability()

	handler, ok := s.customMessageRegistry.Handler(capability)
	if !ok {
		logger.Debug("no handler for inbound custom message capability, dropping",
			slog.String("capability", capability))

		return nil
	}

	reply, err := handler.HandleCustomMessage(ctx, agent, customMessageToDomain(msg))
	if err != nil {
		logger.Error("custom message handler failed",
			slog.String("capability", capability),
			slog.String("error", err.Error()))

		return nil
	}

	if reply == nil {
		return nil
	}

	// Capability gate: never send a custom_message for a capability the agent has not advertised.
	if !agent.HasCustomCapability(reply.Capability) {
		logger.Warn("dropping outbound custom message: agent has not advertised its capability",
			slog.String("capability", reply.Capability))

		return nil
	}

	return customMessageToProtobuf(reply)
}

func (s *Service) report(
	agent *agentmodel.Agent,
	agentToServer *protobufs.AgentToServer,
	by *agentmodel.Server,
) error {
	now := s.clock.Now()

	// Update communication info
	agent.RecordLastReported(by, now, agentToServer.GetSequenceNum())

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

	err = agent.ReportRemoteConfigStatus(remoteConfigStatusToDomain(agentToServer.GetRemoteConfigStatus(), now))
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

	// agentToServer.CustomMessage is not consumed here: report() folds the agent's reported
	// state into the model, whereas a custom_message is routed to its registered handler on the
	// hot path (see handleInboundCustomMessage in OnMessage), not persisted as agent state.

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

	// WebSocket close is a genuine disconnect — drop the liveness record so the first
	// message after reconnect is written through immediately instead of waiting out the
	// throttle window left by the previous session. HTTP polling agents do not get here
	// because their close is treated as request-end, not disconnect.
	if !connection.IsAnonymous() && connection.Type == agentmodel.ConnectionTypeWebSocket {
		forgetErr := s.agentUsecase.ForgetAgentLiveness(ctx, connection.InstanceUID)
		if forgetErr != nil {
			logger.Warn("failed to forget agent liveness", slog.String("error", forgetErr.Error()))
		}
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

// recordLiveness absorbs the agent's liveness into the fast tier and decides whether
// this message also warrants a write to the durable store.
//
// Non-heartbeat messages (carrying any reported field) changed durable state and are
// always persisted. Heartbeat-only messages — which dominate the volume at scale —
// changed nothing but the liveness fields the fast tier now holds, so they are written
// through only once per throttle window.
func (s *Service) recordLiveness(
	ctx context.Context,
	agent *agentmodel.Agent,
	message *protobufs.AgentToServer,
	receivedAt time.Time,
) bool {
	duePersist := s.agentUsecase.TouchAgentLiveness(ctx, agent, receivedAt)

	return !isHeartbeatOnly(message) || duePersist
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
//
// It returns the report error (if any) so the caller can surface it to the agent as an
// error_response; on error the group reconcile is skipped because the agent's state may be
// inconsistent.
func (s *Service) reportAndReconcileGroups(
	ctx context.Context,
	logger *slog.Logger,
	message *protobufs.AgentToServer,
	agent *agentmodel.Agent,
	currentServer *agentmodel.Server,
) error {
	hasDescription := message.GetAgentDescription() != nil

	var prevIdentity identitySnapshot
	if hasDescription {
		prevIdentity = snapshotIdentity(agent)
	}

	err := s.report(agent, message, currentServer)
	if err != nil {
		logger.Error("failed to report agent", slog.String("error", err.Error()))

		return err
	}

	if hasDescription {
		s.maybeApplyMatchingAgentGroups(ctx, logger, agent, prevIdentity)
	}

	return nil
}

// maybePersistAgent records the agent's liveness and writes it to the durable store
// when the message warrants it. SaveAgent re-anchors the throttle window on success,
// so the next heartbeat-only message is measured from this write.
func (s *Service) maybePersistAgent(
	ctx context.Context,
	logger *slog.Logger,
	message *protobufs.AgentToServer,
	agent *agentmodel.Agent,
	receivedAt time.Time,
) {
	if !s.recordLiveness(ctx, agent, message, receivedAt) {
		return
	}

	err := s.agentUsecase.SaveAgent(ctx, agent)
	if err != nil {
		logger.Error("failed to save agent", slog.String("error", err.Error()))

		return
	}

	s.observeEnvironment(ctx, logger, agent)
}

// observeEnvironment discovers and upserts the host/container the agent runs in
// from its reported description. It rides the same throttle as agent persistence
// so the discovery inventory advances at the agent-save cadence rather than on
// every heartbeat. Discovery failures are non-fatal: they are logged and do not
// affect the agent's own processing.
func (s *Service) observeEnvironment(
	ctx context.Context,
	logger *slog.Logger,
	agent *agentmodel.Agent,
) {
	hostErr := s.hostUsecase.ObserveAgent(ctx, agent)
	if hostErr != nil {
		logger.Error("failed to observe host for agent", slog.String("error", hostErr.Error()))
	}

	containerErr := s.containerUsecase.ObserveAgent(ctx, agent)
	if containerErr != nil {
		logger.Error("failed to observe container for agent", slog.String("error", containerErr.Error()))
	}
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
