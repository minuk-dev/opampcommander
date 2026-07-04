package agentservice

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server/types"
	"github.com/samber/lo"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
	"github.com/minuk-dev/opampcommander/pkg/xsync"
)

var (
	_ agentport.ConnectionUsecase        = (*Service)(nil)
	_ agentport.ClusterConnectionUsecase = (*Service)(nil)
)

const (
	idxInstanceUID = "instanceUID"

	connectionServiceName = "ConnectionService"

	// DefaultConnectionSnapshotInterval is how often this server persists a snapshot of its
	// local connections so other servers can see them in the cluster-wide view.
	DefaultConnectionSnapshotInterval = 30 * time.Second
	// DefaultConnectionSnapshotStaleness is how long a server's snapshot is trusted after its
	// last refresh. Records older than this (a crashed/stopped server) are excluded from
	// cluster reads. Sized at 3x the snapshot interval so a single missed snapshot does not
	// drop a live server's connections.
	DefaultConnectionSnapshotStaleness = 3 * DefaultConnectionSnapshotInterval
)

// Service is a struct that implements the ConnectionUsecase interface.
//
// connectionMap holds only the live connections owned by THIS server instance: an
// OpAMP WebSocket is a stateful socket that lives on exactly one node, so it cannot be
// shared or reconstructed elsewhere. Consequently every read here (GetConnectionByID,
// ListConnections, ...) is node-scoped by design.
//
// For a cluster-wide view, each server periodically snapshots its connectionMap into the
// shared ServerConnectionPersistencePort (see Run/ListClusterConnections); reads there span
// every alive server. The agent record (Status.Connected / ConnectionType / LastReportedTo)
// remains the authoritative, always-current source of agent connectivity.
type Service struct {
	agentUsecase                    agentport.AgentUsecase
	logger                          *slog.Logger
	connectionMap                   *xsync.MultiMap[*agentmodel.Connection]
	serverIdentityProvider          agentport.ServerIdentityProvider
	serverConnectionPersistencePort agentport.ServerConnectionPersistencePort
	clock                           clock.Clock

	snapshotInterval  time.Duration
	snapshotStaleness time.Duration
}

// NewConnectionService creates a new instance of the Service struct.
func NewConnectionService(
	agentUsecase agentport.AgentUsecase,
	serverIdentityProvider agentport.ServerIdentityProvider,
	serverConnectionPersistencePort agentport.ServerConnectionPersistencePort,
	logger *slog.Logger,
) *Service {
	return &Service{
		agentUsecase:                    agentUsecase,
		logger:                          logger,
		connectionMap:                   xsync.NewMultiMap[*agentmodel.Connection](),
		serverIdentityProvider:          serverIdentityProvider,
		serverConnectionPersistencePort: serverConnectionPersistencePort,
		clock:                           clock.NewRealClock(),
		snapshotInterval:                DefaultConnectionSnapshotInterval,
		snapshotStaleness:               DefaultConnectionSnapshotStaleness,
	}
}

// Name implements scheduler.Scheduler.
func (s *Service) Name() string {
	return connectionServiceName
}

// Run implements scheduler.Scheduler. It periodically snapshots this server's local
// connections into the shared store so they appear in the cluster-wide view, and clears
// the server's records on graceful shutdown so a stopped node drops out immediately rather
// than lingering until its snapshot goes stale.
func (s *Service) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.effectiveSnapshotInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.clearSnapshotOnShutdown(ctx)

			return nil
		case <-ticker.C:
			s.snapshotConnections(ctx)
		}
	}
}

// ListClusterConnections implements agentport.ClusterConnectionUsecase. It returns the
// connections of every alive server from the shared snapshot store, excluding records whose
// last snapshot is older than the staleness window (so a crashed server's connections do
// not linger in the view).
func (s *Service) ListClusterConnections(
	ctx context.Context,
	namespace string,
	serverID string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.ServerConnection], error) {
	notBefore := s.clock.Now().Add(-s.effectiveSnapshotStaleness())

	resp, err := s.serverConnectionPersistencePort.ListServerConnections(ctx, namespace, serverID, notBefore, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster connections: %w", err)
	}

	return resp, nil
}

// DeleteConnection implements agentport.ConnectionUsecase.
func (s *Service) DeleteConnection(_ context.Context, connection *agentmodel.Connection) error {
	connID := connection.IDString()
	s.connectionMap.Delete(connID)

	return nil
}

// GetConnectionByID implements agentport.ConnectionUsecase.
func (s *Service) GetConnectionByID(_ context.Context, id any) (*agentmodel.Connection, error) {
	connID := agentmodel.ConvertConnIDToString(id)

	conn, ok := s.connectionMap.Load(connID)
	if !ok {
		s.logger.Debug("connection not found by ID",
			slog.String("connIDHash", connID),
			slog.String("rawID", fmt.Sprintf("%v", id)),
		)

		return nil, agentport.ErrConnectionNotFound
	}

	return conn, nil
}

// GetOrCreateConnectionByID implements agentport.ConnectionUsecase.
func (s *Service) GetOrCreateConnectionByID(_ context.Context, id any) (*agentmodel.Connection, error) {
	connID := agentmodel.ConvertConnIDToString(id)

	conn, ok := s.connectionMap.Load(connID)
	if ok {
		return conn, nil
	}

	connectionType := s.detectConnectionType(id)
	// Create a new connection
	newConn := agentmodel.NewConnection(id, connectionType)

	return newConn, nil
}

// GetConnectionByInstanceUID implements agentport.ConnectionUsecase.
func (s *Service) GetConnectionByInstanceUID(_ context.Context, instanceUID uuid.UUID) (*agentmodel.Connection, error) {
	conn, ok := s.connectionMap.LoadByIndex(idxInstanceUID, instanceUID.String())
	if !ok {
		return nil, agentport.ErrConnectionNotFound
	}

	return conn, nil
}

// ListConnections implements agentport.ConnectionUsecase.
//
// It returns only the connections held by this server instance (see the Service doc):
// in HA the result is the local node's live connections, not a cluster-wide list. Use
// the agents API for a global view of agent connectivity.
func (s *Service) ListConnections(
	_ context.Context,
	namespace string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Connection], error) {
	if options == nil {
		options = &model.ListOptions{
			Limit:          0,  // 0 means no limit
			Continue:       "", // empty continue token means start from the beginning
			IncludeDeleted: false,
		}
	}

	keyValues := s.connectionMap.KeyValues()

	// Filter by namespace
	for key, conn := range keyValues {
		if conn.Namespace != namespace {
			delete(keyValues, key)
		}
	}

	keys := lo.Keys(keyValues)
	sort.Strings(keys)

	if options.Continue != "" {
		// Find the index of the continue token
		index := sort.SearchStrings(keys, options.Continue)
		if index < len(keys) {
			keys = keys[index:]
		} else {
			keys = nil // If continue token not found, return empty list
		}
	}

	totalMatchedItemsCount := len(keys)

	var nextContinue string
	if options.Limit > 0 && int64(len(keys)) > options.Limit {
		nextContinue = keys[options.Limit]
		keys = keys[:options.Limit]
	} else {
		nextContinue = "\xff" // Use a sentinel value to indicate no more items
	}

	return &model.ListResponse[*agentmodel.Connection]{
		Items: lo.Map(keys, func(key string, _ int) *agentmodel.Connection {
			return keyValues[key]
		}),
		Continue:           nextContinue,
		RemainingItemCount: int64(totalMatchedItemsCount - len(keys)),
	}, nil
}

// SaveConnection implements agentport.ConnectionUsecase.
func (s *Service) SaveConnection(_ context.Context, connection *agentmodel.Connection) error {
	connID := connection.IDString()

	var additionalIndexesOpts []xsync.StoreOption
	if connection.InstanceUID != uuid.Nil {
		additionalIndexesOpts = append(additionalIndexesOpts,
			xsync.WithIndex(idxInstanceUID, connection.InstanceUID.String()))
	}

	s.connectionMap.Store(connID, connection, additionalIndexesOpts...)

	s.logger.Debug("connection saved",
		slog.String("connectionUID", connection.UID.String()),
		slog.String("instanceUID", connection.InstanceUID.String()),
		slog.String("connectionType", connection.Type.String()),
		slog.String("namespace", connection.Namespace),
	)

	return nil
}

// SendServerToAgent sends a ServerToAgent message to the agent via WebSocket connection.
func (s *Service) SendServerToAgent(
	ctx context.Context,
	instanceUID uuid.UUID,
	message *protobufs.ServerToAgent,
) error {
	// Get connection metadata
	connection, err := s.GetConnectionByInstanceUID(ctx, instanceUID)
	if err != nil {
		return fmt.Errorf("failed to get connection for agent %s: %w", instanceUID, err)
	}

	wsConn, ok := connection.ID.(types.Connection)
	if !ok {
		return &ConnectionNotFoundError{InstanceUID: instanceUID}
	}

	err = wsConn.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to send ServerToAgent message to agent %s: %w", instanceUID, err)
	}

	s.logger.Info("sent ServerToAgent message to agent via WebSocket",
		slog.String("instanceUID", instanceUID.String()))

	return nil
}

// detectConnectionType detects whether the connection is WebSocket or HTTP.
// According to OpAMP spec:
// - WebSocket: Bidirectional, persistent connection. OnConnected is called first.
// - HTTP: Request-response only, stateless. Only OnMessage is called per request.
//
// The key difference: WebSocket connections maintain a persistent net.Conn,
// while HTTP connections are ephemeral (created per request).
// We check if the connection object supports the Send method and has a persistent connection.
func (s *Service) detectConnectionType(id any) agentmodel.ConnectionType {
	conn, ok := id.(types.Connection)
	if !ok {
		// If it's not a types.Connection, we can't determine the type
		return agentmodel.ConnectionTypeUnknown
	}

	// Check if we have an underlying network connection
	netConn := conn.Connection()
	if netConn == nil {
		// No underlying connection means it's likely HTTP (request-response only)
		return agentmodel.ConnectionTypeHTTP
	}

	// If we have a persistent connection, it's WebSocket
	// maintains the connection after OnConnected
	// HTTP only has connection during request processing
	return agentmodel.ConnectionTypeWebSocket
}

// snapshotConnections persists the current set of this server's local connections,
// replacing any previously stored set for this server.
func (s *Service) snapshotConnections(ctx context.Context) {
	serverID := s.serverIdentityProvider.CurrentServerID()
	if serverID == "" {
		s.logger.Warn("skipping connection snapshot: current server has no identity")

		return
	}

	now := s.clock.Now()
	conns := s.connectionMap.Values()

	records := make([]*agentmodel.ServerConnection, 0, len(conns))
	for _, conn := range conns {
		records = append(records, &agentmodel.ServerConnection{
			ServerID:           serverID,
			UID:                conn.UID,
			InstanceUID:        conn.InstanceUID,
			Type:               conn.Type,
			Namespace:          conn.Namespace,
			LastCommunicatedAt: conn.LastCommunicatedAt,
			SnapshotAt:         now,
		})
	}

	err := s.serverConnectionPersistencePort.ReplaceServerConnections(ctx, serverID, records)
	if err != nil {
		s.logger.Error("failed to snapshot connections", slog.String("error", err.Error()))

		return
	}

	s.logger.Debug("snapshotted connections", slog.Int("count", len(records)))
}

// clearSnapshotOnShutdown removes this server's snapshot records so it drops out of the
// cluster view promptly on graceful shutdown. The passed ctx is already cancelled (Run only
// calls this after ctx.Done), so it derives a fresh, short-lived context via WithoutCancel —
// keeping any request values while shedding the cancellation. Best-effort: failures are
// logged, not retried.
func (s *Service) clearSnapshotOnShutdown(ctx context.Context) {
	serverID := s.serverIdentityProvider.CurrentServerID()
	if serverID == "" {
		return
	}

	const clearTimeout = 5 * time.Second

	clearCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), clearTimeout)
	defer cancel()

	err := s.serverConnectionPersistencePort.ReplaceServerConnections(clearCtx, serverID, nil)
	if err != nil {
		s.logger.Warn("failed to clear connection snapshot on shutdown", slog.String("error", err.Error()))
	}
}

func (s *Service) effectiveSnapshotInterval() time.Duration {
	if s.snapshotInterval <= 0 {
		return DefaultConnectionSnapshotInterval
	}

	return s.snapshotInterval
}

func (s *Service) effectiveSnapshotStaleness() time.Duration {
	if s.snapshotStaleness <= 0 {
		return DefaultConnectionSnapshotStaleness
	}

	return s.snapshotStaleness
}

// NotSupportedConnectionTypeError is returned when an operation is attempted.
type NotSupportedConnectionTypeError struct {
	ConnectionType agentmodel.ConnectionType
}

// Error implements the error interface.
func (e *NotSupportedConnectionTypeError) Error() string {
	return fmt.Sprintf("connection type %s is not supported", e.ConnectionType.String())
}

// ConnectionNotFoundError is returned when a connection is not found for a given instance UID.
type ConnectionNotFoundError struct {
	InstanceUID uuid.UUID
}

// Error implements the error interface.
func (e *ConnectionNotFoundError) Error() string {
	return "connection not found for instance UID " + e.InstanceUID.String()
}
