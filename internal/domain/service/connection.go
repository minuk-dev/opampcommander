package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/samber/lo"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/xsync"
)

var _ port.ConnectionUsecase = (*Service)(nil)

const (
	idxInstanceUID = "instanceUID"
)

// Service is a struct that implements the ConnectionUsecase interface.
type Service struct {
	agentUsecase  port.AgentUsecase
	logger        *slog.Logger
	wsRegistry    port.WebSocketRegistry
	connectionMap *xsync.MultiMap[*model.Connection]
}

// NewConnectionService creates a new instance of the Service struct.
func NewConnectionService(
	agentUsecase port.AgentUsecase,
	wsRegistry port.WebSocketRegistry,
	logger *slog.Logger,
) *Service {
	return &Service{
		agentUsecase:  agentUsecase,
		wsRegistry:    wsRegistry,
		logger:        logger,
		connectionMap: xsync.NewMultiMap[*model.Connection](),
	}
}

// DeleteConnection implements port.ConnectionUsecase.
func (s *Service) DeleteConnection(_ context.Context, connection *model.Connection) error {
	connID := connection.IDString()

	// Remove from WebSocket registry if it's a WebSocket connection
	if connection.Type == model.ConnectionTypeWebSocket {
		s.wsRegistry.Remove(connID)
	}

	s.connectionMap.Delete(connID)

	return nil
}

// GetConnectionByID implements port.ConnectionUsecase.
func (s *Service) GetConnectionByID(_ context.Context, id any) (*model.Connection, error) {
	connID := model.ConvertConnIDToString(id)

	conn, ok := s.connectionMap.Load(connID)
	if !ok {
		return nil, port.ErrConnectionNotFound
	}

	return conn, nil
}

// GetConnectionByInstanceUID implements port.ConnectionUsecase.
func (s *Service) GetConnectionByInstanceUID(_ context.Context, instanceUID uuid.UUID) (*model.Connection, error) {
	conn, ok := s.connectionMap.LoadByIndex(idxInstanceUID, instanceUID.String())
	if !ok {
		return nil, port.ErrConnectionNotFound
	}

	return conn, nil
}

// ListConnections implements port.ConnectionUsecase.
func (s *Service) ListConnections(
	_ context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.Connection], error) {
	if options == nil {
		options = &model.ListOptions{
			Limit:    0,  // 0 means no limit
			Continue: "", // empty continue token means start from the beginning
		}
	}

	keyValues := s.connectionMap.KeyValues()
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
	if options.Limit > 0 && len(keys) > int(options.Limit) {
		nextContinue = keys[options.Limit]
		keys = keys[:options.Limit]
	} else {
		nextContinue = "\xff" // Use a sentinel value to indicate no more items
	}

	return &model.ListResponse[*model.Connection]{
		Items: lo.Map(keys, func(key string, _ int) *model.Connection {
			return keyValues[key]
		}),
		Continue:           nextContinue,
		RemainingItemCount: int64(totalMatchedItemsCount - len(keys)),
	}, nil
}

// SaveConnection implements port.ConnectionUsecase.
func (s *Service) SaveConnection(_ context.Context, connection *model.Connection) error {
	connID := connection.IDString()

	var additionalIndexesOpts []xsync.StoreOption
	if connection.InstanceUID != uuid.Nil {
		additionalIndexesOpts = append(additionalIndexesOpts,
			xsync.WithIndex(idxInstanceUID, connection.InstanceUID.String()))
	}

	s.connectionMap.Store(connID, connection, additionalIndexesOpts...)

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

	if connection.Type != model.ConnectionTypeWebSocket {
		return &NotSupportedConnectionTypeError{ConnectionType: connection.Type}
	}

	// Get the actual WebSocket connection from the registry
	wsConn, ok := s.wsRegistry.GetByInstanceUID(instanceUID.String())
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

// NotSupportedConnectionTypeError is returned when an operation is attempted.
type NotSupportedConnectionTypeError struct {
	ConnectionType model.ConnectionType
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
