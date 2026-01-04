package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ port.ServerIdentityProvider = (*ServerIdentityService)(nil)

var (
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

// ServerIdentityService provides server identity and heartbeat management.
// It implements port.ServerIdentityProvider interface.
type ServerIdentityService struct {
	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration
	id                string

	clock                 clock.Clock
	logger                *slog.Logger
	serverPersistencePort port.ServerPersistencePort
}

// NewServerIdentityService creates a new ServerIdentityService instance.
func NewServerIdentityService(
	serverPersistencePort port.ServerPersistencePort,
	serverID config.ServerID,
	logger *slog.Logger,
) *ServerIdentityService {
	return &ServerIdentityService{
		clock:                 clock.NewRealClock(),
		id:                    serverID.String(),
		heartbeatInterval:     DefaultHeartbeatInterval,
		heartbeatTimeout:      DefaultHeartbeatTimeout,
		serverPersistencePort: serverPersistencePort,
		logger:                logger,
	}
}

// CurrentServer implements port.ServerUsecase.
func (s *ServerIdentityService) CurrentServer(ctx context.Context) (*model.Server, error) {
	server, err := s.serverPersistencePort.GetServer(ctx, s.id)
	if err != nil {
		return nil, fmt.Errorf("failed to get current server: %w", err)
	}

	return server, nil
}

// Run starts the server service and maintains heartbeat.
func (s *ServerIdentityService) Run(ctx context.Context) error {
	// Try to register the server
	err := s.registerServer(ctx)
	if err != nil {
		return fmt.Errorf("failed to register server: %w", err)
	}

	s.logger.Info("server registered successfully", slog.String("serverID", s.id))

	var wg sync.WaitGroup
	wg.Go(func() {
		err := s.loopForHeartbeat(ctx)
		if err != nil {
			s.logger.Error("heartbeat loop exited with error", slog.String("error", err.Error()))
		}
	})
	wg.Wait()

	return nil
}

func (s *ServerIdentityService) loopForHeartbeat(ctx context.Context) error {
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

// registerServer registers the server in the database.
func (s *ServerIdentityService) registerServer(ctx context.Context) error {
	now := s.clock.Now()

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
		Conditions:      []model.ServerCondition{},
	}

	if existingServer != nil {
		// Copy existing conditions
		server.Conditions = existingServer.Conditions
	} else {
		// Mark as registered for new servers
		server.MarkRegistered("system")
	}

	// Mark as alive
	server.MarkAlive("heartbeat")

	err = s.serverPersistencePort.PutServer(ctx, server)
	if err != nil {
		return fmt.Errorf("failed to put server: %w", err)
	}

	return nil
}

// sendHeartbeat sends a heartbeat to update the server's last heartbeat time.
func (s *ServerIdentityService) sendHeartbeat(ctx context.Context) error {
	server, err := s.serverPersistencePort.GetServer(ctx, s.id)
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}

	server.LastHeartbeatAt = s.clock.Now()

	err = s.serverPersistencePort.PutServer(ctx, server)
	if err != nil {
		return fmt.Errorf("failed to update server heartbeat: %w", err)
	}

	s.logger.Debug("heartbeat sent", slog.String("serverID", s.id))

	return nil
}
