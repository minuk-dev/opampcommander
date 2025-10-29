package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/serverevent"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
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

	clock clock.Clock

	serverPersistencePort   port.ServerPersistencePort
	serverEventSenderPort   port.ServerEventSenderPort
	serverEventReceiverPort port.ServerEventReceiverPort
}

// NewServerService creates a new instance of the ServerService.
func NewServerService(
	logger *slog.Logger,
	serverID config.ServerID,
	serverPersistencePort port.ServerPersistencePort,
	serverEventSenderPort port.ServerEventSenderPort,
) *ServerService {
	service := &ServerService{
		logger:                logger,
		id:                    serverID.String(),
		clock:                 clock.NewRealClock(),
		heartbeatInterval:     DefaultHeartbeatInterval,
		heartbeatTimeout:      DefaultHeartbeatTimeout,
		serverPersistencePort: serverPersistencePort,
		serverEventSenderPort: serverEventSenderPort,
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

	var wg sync.WaitGroup
	wg.Go(func() {
		err := s.loopForHeartbeat(ctx)
		if err != nil {
			s.logger.Error("heartbeat loop exited with error", slog.String("error", err.Error()))
		}
	})
	wg.Go(func() {
		err := s.loopForReceivingMessages(ctx)
		if err != nil {
			s.logger.Error("message receiving loop exited with error", slog.String("error", err.Error()))
		}
	})

	wg.Wait()
	return nil
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
	now := s.clock.Now()
	aliveServers := make([]*model.Server, 0)

	for _, server := range servers {
		if server.IsAlive(now, s.heartbeatTimeout) {
			aliveServers = append(aliveServers, server)
		}
	}

	return aliveServers, nil
}

// registerServer registers the server in the database.
func (s *ServerService) registerServer(ctx context.Context) error {
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

// SendMessageToServer implements port.ServerUsecase.
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

func (s *ServerService) SendMessageToServer(
	ctx context.Context,
	server *model.Server,
	message serverevent.Message,
) error {
	if !server.IsAlive(s.clock.Now(), s.heartbeatTimeout) {
		return fmt.Errorf("%w: server ID %s is not alive", ErrServerNotAlive, server.ID)
	}

	err := s.serverEventSenderPort.SendMessageToServer(ctx, server.ID, message)
	if err != nil {
		return fmt.Errorf("failed to send message to server %s: %w", server.ID, err)
	}

	return nil
}

func (s *ServerService) loopForHeartbeat(ctx context.Context) error {
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

func (s *ServerService) loopForReceivingMessages(ctx context.Context) error {
	for {
		serverEvent, err := s.serverEventReceiverPort.ReceiveMessageFromServer(ctx)
		if errors.Is(err, context.Canceled) {
			return err
		} else if err != nil {
			s.logger.Error("failed to receive message from server", slog.String("error", err.Error()))
		}

		// TODO: Handle the received server event
		_ = serverEvent
	}
}
