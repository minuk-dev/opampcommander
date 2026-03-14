package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/xsync"
)

var _ port.AgentUsecase = (*AgentService)(nil)

const (
	// DefaultAgentCacheTTL is the default time-to-live for agent cache entries.
	// Using a short TTL to ensure agent state stays relatively fresh while still reducing DB load.
	DefaultAgentCacheTTL = 30 * time.Second
)

// AgentService is a struct that implements the AgentUsecase interface.
type AgentService struct {
	agentPersistencePort port.AgentPersistencePort
	logger               *slog.Logger
	agentCache           *xsync.TTLCache[uuid.UUID, *model.Agent]
}

// NewAgentService creates a new instance of AgentService.
func NewAgentService(
	agentPersistencePort port.AgentPersistencePort,
	logger *slog.Logger,
) *AgentService {
	return &AgentService{
		agentPersistencePort: agentPersistencePort,
		logger:               logger,
		agentCache:           xsync.NewTTLCache[uuid.UUID, *model.Agent](DefaultAgentCacheTTL),
	}
}

// Shutdown releases resources held by the service.
// This should be called during graceful shutdown.
func (s *AgentService) Shutdown() {
	s.logger.Info("shutting down agent service, clearing cache")
	s.agentCache.Shutdown()
}

// InvalidateCache removes a specific agent from the cache.
func (s *AgentService) InvalidateCache(instanceUID uuid.UUID) {
	s.agentCache.Delete(instanceUID)
}

// GetAgent retrieves an agent by its instance UID.
func (s *AgentService) GetAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error) {
	// Try cache first
	if agent, ok := s.agentCache.Get(instanceUID); ok {
		return agent, nil
	}

	agent, err := s.agentPersistencePort.GetAgent(ctx, instanceUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent from persistence: %w", err)
	}

	// Cache the result
	s.agentCache.Set(instanceUID, agent)

	return agent, nil
}

// GetOrCreateAgent retrieves an agent by its instance UID.
// If the agent doesn't exist, it creates a new one with default values.
func (s *AgentService) GetOrCreateAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error) {
	agent, err := s.GetAgent(ctx, instanceUID)
	if err != nil {
		if errors.Is(err, port.ErrResourceNotExist) {
			agent = model.NewAgent(instanceUID)
		} else {
			return nil, fmt.Errorf("failed to get agent: %w", err)
		}
	}

	return agent, nil
}

// SaveAgent saves the agent to the persistence layer.
func (s *AgentService) SaveAgent(ctx context.Context, agent *model.Agent) error {
	err := s.agentPersistencePort.PutAgent(ctx, agent)
	if err != nil {
		return fmt.Errorf("failed to save agent to persistence: %w", err)
	}

	// Update cache with the saved agent
	s.agentCache.Set(agent.Metadata.InstanceUID, agent)

	return nil
}

// ListAgents retrieves all agents from the persistence layer.
func (s *AgentService) ListAgents(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	res, err := s.agentPersistencePort.ListAgents(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	return res, nil
}

// ListAgentsBySelector implements port.AgentUsecase.
func (s *AgentService) ListAgentsBySelector(
	ctx context.Context,
	selector model.AgentSelector,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	resp, err := s.agentPersistencePort.ListAgentsBySelector(ctx, selector, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents by selector: %w", err)
	}

	return resp, nil
}

// SearchAgents implements port.AgentUsecase.
func (s *AgentService) SearchAgents(
	ctx context.Context,
	query string,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	resp, err := s.agentPersistencePort.SearchAgents(ctx, query, options)
	if err != nil {
		return nil, fmt.Errorf("failed to search agents: %w", err)
	}

	return resp, nil
}
