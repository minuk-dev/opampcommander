package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

var (
	_ port.ServerUsecase = (*ServerService)(nil)

	// ErrServerIDAlreadyExists is an error that indicates that the server ID already exists.
	ErrServerIDAlreadyExists = errors.New("server ID already exists")
	// ErrServerNotAlive is an error that indicates that the server is not alive.
	ErrServerNotAlive = errors.New("server is not alive")
)

const (
	// DefaultHeartbeatInterval is the default interval for sending heartbeats.
	DefaultHeartbeatInterval = 30 * time.Second
	// DefaultHeartbeatTimeout is the default timeout for considering a server as dead.
	DefaultHeartbeatTimeout = 90 * time.Second
)

// ServerService is a struct that implements the ServerUsecase interface.
type ServerService struct {
	logger *slog.Logger
	id     string

	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration

	serverPersistencePort port.ServerPersistencePort
}

// ServerServiceOption is a function that configures the ServerService.
type ServerServiceOption func(*ServerService)

// WithHeartbeatInterval sets the heartbeat interval.
func WithHeartbeatInterval(interval time.Duration) ServerServiceOption {
	return func(s *ServerService) {
		s.heartbeatInterval = interval
	}
}

// WithHeartbeatTimeout sets the heartbeat timeout.
func WithHeartbeatTimeout(timeout time.Duration) ServerServiceOption {
	return func(s *ServerService) {
		s.heartbeatTimeout = timeout
	}
}

// NewServerService creates a new instance of the ServerService.
func NewServerService(
	logger *slog.Logger,
	serverID config.ServerID,
	serverPersistencePort port.ServerPersistencePort,
	opts ...ServerServiceOption,
) *ServerService {
	service := &ServerService{
		logger:                logger,
		id:                    serverID.String(),
		heartbeatInterval:     DefaultHeartbeatInterval,
		heartbeatTimeout:      DefaultHeartbeatTimeout,
		serverPersistencePort: serverPersistencePort,
	}

	for _, opt := range opts {
		opt(service)
	}

	return service
}

// ID returns the ID of the server.
func (s *ServerService) ID() string {
	return s.id
}

// Name returns the name of the runner.
func (s *ServerService) Name() string {
	return "ServerService"
}

// Run starts the server service and maintains heartbeat.
func (s *ServerService) Run(ctx context.Context) error {
	// Try to register the server
	err := s.registerServer(ctx)
	if err != nil {
		return fmt.Errorf("failed to register server: %w", err)
	}

	s.logger.Info("server registered successfully", slog.String("serverID", s.id))

	ticker := time.NewTicker(s.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		case <-ticker.C:
			err := s.sendHeartbeat(ctx)
			if err != nil {
				s.logger.Error("failed to send heartbeat", slog.String("error", err.Error()))
			}
		}
	}
}

// CurrentServer implements port.ServerUsecase.
func (s *ServerService) CurrentServer(ctx context.Context) (*model.Server, error) {
	server, err := s.serverPersistencePort.GetServer(ctx, s.id)
	if err != nil {
		return nil, fmt.Errorf("failed to get current server: %w", err)
	}

	return server, nil
}

// GetServer implements port.ServerUsecase.
func (s *ServerService) GetServer(ctx context.Context, id string) (*model.Server, error) {
	server, err := s.serverPersistencePort.GetServer(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	return server, nil
}

// ListServers implements port.ServerUsecase.
func (s *ServerService) ListServers(ctx context.Context) ([]*model.Server, error) {
	servers, err := s.serverPersistencePort.ListServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}

	// Filter out dead servers
	now := time.Now()
	aliveServers := make([]*model.Server, 0)

	for _, server := range servers {
		if server.IsAlive(now, s.heartbeatTimeout) {
			aliveServers = append(aliveServers, server)
		}
	}

	return aliveServers, nil
}

// SendMessageToServer implements port.ServerUsecase.
//
//nolint:godox // wip in PR
func (s *ServerService) SendMessageToServer(context.Context, string, port.ServerMessage) error {
	// TODO: Implement the logic to send a message to the specified server.
	panic("not implemented")
}

// registerServer registers the server in the database.
func (s *ServerService) registerServer(ctx context.Context) error {
	now := time.Now()

	// Check if server ID already exists and is alive
	existingServer, err := s.serverPersistencePort.GetServer(ctx, s.id)
	if err != nil && !errors.Is(err, port.ErrResourceNotExist) {
		return fmt.Errorf("failed to check existing server: %w", err)
	}

	if existingServer != nil && existingServer.IsAlive(now, s.heartbeatTimeout) {
		return fmt.Errorf("%w: server ID %s is already in use by an alive server", ErrServerIDAlreadyExists, s.id)
	}

	// Register or update the server
	server := &model.Server{
		ID:              s.id,
		LastHeartbeatAt: now,
		CreatedAt:       now,
	}

	if existingServer != nil {
		server.CreatedAt = existingServer.CreatedAt
	}

	err = s.serverPersistencePort.PutServer(ctx, server)
	if err != nil {
		return fmt.Errorf("failed to put server: %w", err)
	}

	return nil
}

// sendHeartbeat sends a heartbeat to update the server's last heartbeat time.
func (s *ServerService) sendHeartbeat(ctx context.Context) error {
	server, err := s.serverPersistencePort.GetServer(ctx, s.id)
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}

	server.LastHeartbeatAt = time.Now()

	err = s.serverPersistencePort.PutServer(ctx, server)
	if err != nil {
		return fmt.Errorf("failed to update server heartbeat: %w", err)
	}

	s.logger.Debug("heartbeat sent", slog.String("serverID", s.id))

	return nil
}
