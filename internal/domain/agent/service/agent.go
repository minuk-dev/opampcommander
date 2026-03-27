package agentservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jellydator/ttlcache/v3"

	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ agentport.AgentUsecase = (*AgentService)(nil)

const (
	// DefaultAgentCacheTTL is the default time-to-live for agent cache entries.
	DefaultAgentCacheTTL = 30 * time.Second
	// DefaultAgentCacheCapacity is the default maximum number of agent cache entries.
	DefaultAgentCacheCapacity int64 = 1000
)

// AgentCacheConfig holds the configuration for agent caching.
type AgentCacheConfig struct {
	Enabled     bool
	TTL         time.Duration
	MaxCapacity int64
}

// AgentService is a struct that implements the AgentUsecase interface.
type AgentService struct {
	agentPersistencePort agentport.AgentPersistencePort
	logger               *slog.Logger
	agentCache           *ttlcache.Cache[uuid.UUID, *agentmodel.Agent]
	cacheEnabled         bool
}

// NewAgentService creates a new instance of AgentService with default cache configuration.
func NewAgentService(
	agentPersistencePort agentport.AgentPersistencePort,
	logger *slog.Logger,
) *AgentService {
	return NewAgentServiceWithConfig(agentPersistencePort, logger, AgentCacheConfig{
		Enabled:     true,
		TTL:         DefaultAgentCacheTTL,
		MaxCapacity: DefaultAgentCacheCapacity,
	})
}

// NewAgentServiceWithConfig creates a new instance of AgentService with custom cache configuration.
func NewAgentServiceWithConfig(
	agentPersistencePort agentport.AgentPersistencePort,
	logger *slog.Logger,
	cacheConfig AgentCacheConfig,
) *AgentService {
	if !cacheConfig.Enabled {
		logger.Info("agent cache disabled")

		return &AgentService{
			agentPersistencePort: agentPersistencePort,
			logger:               logger,
			agentCache:           nil,
			cacheEnabled:         false,
		}
	}

	ttl := cacheConfig.TTL
	if ttl <= 0 {
		ttl = DefaultAgentCacheTTL
	}

	capacity := cacheConfig.MaxCapacity
	if capacity <= 0 {
		capacity = DefaultAgentCacheCapacity
	}

	agentCache := ttlcache.New[uuid.UUID, *agentmodel.Agent](
		ttlcache.WithTTL[uuid.UUID, *agentmodel.Agent](ttl),
		ttlcache.WithCapacity[uuid.UUID, *agentmodel.Agent](uint64(max(capacity, 0))),
	)

	logger.Info("agent cache initialized",
		slog.Duration("ttl", ttl),
		slog.Int64("maxCapacity", capacity),
	)

	return &AgentService{
		agentPersistencePort: agentPersistencePort,
		logger:               logger,
		agentCache:           agentCache,
		cacheEnabled:         true,
	}
}

// Shutdown releases resources held by the service.
// This should be called during graceful shutdown.
func (s *AgentService) Shutdown() {
	if !s.cacheEnabled {
		return
	}

	s.logger.Info("shutting down agent service, clearing cache")
	s.agentCache.DeleteAll()
	s.agentCache.Stop()
}

// InvalidateCache removes a specific agent from the cache.
func (s *AgentService) InvalidateCache(instanceUID uuid.UUID) {
	if !s.cacheEnabled {
		return
	}

	s.agentCache.Delete(instanceUID)
}

// GetAgent retrieves an agent by its instance UID.
func (s *AgentService) GetAgent(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.Agent, error) {
	// Try cache first
	if s.cacheEnabled {
		item := s.agentCache.Get(instanceUID)
		if item != nil {
			// Return a clone to prevent callers from mutating cached data
			return item.Value().Clone(), nil
		}
	}

	agent, err := s.agentPersistencePort.GetAgent(ctx, instanceUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent from persistence: %w", err)
	}

	// Cache a clone to prevent external mutations from affecting cache
	if s.cacheEnabled {
		s.agentCache.Set(instanceUID, agent.Clone(), ttlcache.DefaultTTL)
	}

	return agent, nil
}

// GetOrCreateAgent retrieves an agent by its instance UID.
// If the agent doesn't exist, it creates a new one with default values.
func (s *AgentService) GetOrCreateAgent(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.Agent, error) {
	agent, err := s.GetAgent(ctx, instanceUID)
	if err != nil {
		if errors.Is(err, port.ErrResourceNotExist) {
			agent = agentmodel.NewAgent(instanceUID)
		} else {
			return nil, fmt.Errorf("failed to get agent: %w", err)
		}
	}

	return agent, nil
}

// SaveAgent saves the agent to the persistence layer.
func (s *AgentService) SaveAgent(ctx context.Context, agent *agentmodel.Agent) error {
	err := s.agentPersistencePort.PutAgent(ctx, agent)
	if err != nil {
		return fmt.Errorf("failed to save agent to persistence: %w", err)
	}

	// Cache a clone to prevent external mutations from affecting cache
	if s.cacheEnabled {
		s.agentCache.Set(agent.Metadata.InstanceUID, agent.Clone(), ttlcache.DefaultTTL)
	}

	return nil
}

// ListAgents retrieves all agents from the persistence layer.
func (s *AgentService) ListAgents(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	res, err := s.agentPersistencePort.ListAgents(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	return res, nil
}

// ListAgentsBySelector implements agentport.AgentUsecase.
func (s *AgentService) ListAgentsBySelector(
	ctx context.Context,
	selector agentmodel.AgentSelector,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	resp, err := s.agentPersistencePort.ListAgentsBySelector(ctx, selector, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents by selector: %w", err)
	}

	return resp, nil
}

// SearchAgents implements agentport.AgentUsecase.
func (s *AgentService) SearchAgents(
	ctx context.Context,
	query string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	resp, err := s.agentPersistencePort.SearchAgents(ctx, query, options)
	if err != nil {
		return nil, fmt.Errorf("failed to search agents: %w", err)
	}

	return resp, nil
}
