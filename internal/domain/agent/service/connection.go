package agentservice

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server/types"
	"github.com/samber/lo"

	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/xsync"
)

var _ agentport.ConnectionUsecase = (*Service)(nil)

const (
	idxInstanceUID = "instanceUID"
)

// Service is a struct that implements the ConnectionUsecase interface.
type Service struct {
	agentUsecase  agentport.AgentUsecase
	logger        *slog.Logger
	connectionMap *xsync.MultiMap[*agentmodel.Connection]
}

// NewConnectionService creates a new instance of the Service struct.
func NewConnectionService(
	agentUsecase agentport.AgentUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		agentUsecase:  agentUsecase,
		logger:        logger,
		connectionMap: xsync.NewMultiMap[*agentmodel.Connection](),
	}
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
