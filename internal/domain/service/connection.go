package service

import (
	"context"
	"log/slog"
	"sort"

	"github.com/google/uuid"
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
