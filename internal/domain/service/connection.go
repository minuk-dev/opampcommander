package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

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
	agentUsecase port.AgentUsecase
	logger       *slog.Logger

	connectionMap *xsync.MultiMap[*model.Connection]
}

// NewConnectionService creates a new instance of the Service struct.
func NewConnectionService(
	agentUsecase port.AgentUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		agentUsecase:  agentUsecase,
		logger:        logger,
		connectionMap: xsync.NewMultiMap[*model.Connection](),
	}
}

// DeleteConnection implements port.ConnectionUsecase.
func (s *Service) DeleteConnection(_ context.Context, connection *model.Connection) error {
	connID := connection.IDString()
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
func (s *Service) ListConnections(_ context.Context) ([]*model.Connection, error) {
	return s.connectionMap.Values(), nil
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
