package agentservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jellydator/ttlcache/v3"
	"k8s.io/utils/clock"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/port"
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
	// defaultNamespace is the namespace assigned to a newly-seen agent that has
	// not reported a service.namespace. Sourced from configuration.
	defaultNamespace string
	// clock is consulted only for the delete connection-guard (staleness evaluation).
	clock clock.PassiveClock
}

// NewAgentService creates a new instance of AgentService with default cache configuration.
// The agent default namespace falls back to agentmodel.DefaultNamespaceName.
func NewAgentService(
	agentPersistencePort agentport.AgentPersistencePort,
	logger *slog.Logger,
) *AgentService {
	return NewAgentServiceWithConfig(agentPersistencePort, logger, AgentCacheConfig{
		Enabled:     true,
		TTL:         DefaultAgentCacheTTL,
		MaxCapacity: DefaultAgentCacheCapacity,
	}, agentmodel.DefaultNamespaceName)
}

// NewAgentServiceWithConfig creates a new instance of AgentService with custom cache configuration.
// defaultNamespace is the namespace assigned to newly-seen agents without a service.namespace;
// an empty value falls back to agentmodel.DefaultNamespaceName.
func NewAgentServiceWithConfig(
	agentPersistencePort agentport.AgentPersistencePort,
	logger *slog.Logger,
	cacheConfig AgentCacheConfig,
	defaultNamespace string,
) *AgentService {
	if defaultNamespace == "" {
		defaultNamespace = agentmodel.DefaultNamespaceName
	}

	if !cacheConfig.Enabled {
		logger.Info("agent cache disabled")

		return &AgentService{
			agentPersistencePort: agentPersistencePort,
			logger:               logger,
			agentCache:           nil,
			cacheEnabled:         false,
			defaultNamespace:     defaultNamespace,
			clock:                clock.RealClock{},
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
		ttlcache.WithCapacity[uuid.UUID, *agentmodel.Agent](uint64(capacity)),
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
		defaultNamespace:     defaultNamespace,
		clock:                clock.RealClock{},
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
			agent = agentmodel.NewAgent(instanceUID, agentmodel.WithNamespace(s.defaultNamespace))
		} else {
			return nil, fmt.Errorf("failed to get agent: %w", err)
		}
	}

	return agent, nil
}

// SaveAgent saves the agent to the persistence layer with optimistic concurrency:
// the write is rejected with [port.ErrConflict] when another writer (another HA
// node, the reconcile loop, a racing API call) modified the agent since it was
// loaded, rather than silently clobbering that change.
//
// On conflict the cached copy is invalidated so the next read goes to persistence
// and observes the winning writer's version. Without this the owning server would
// keep re-reading its own stale cached version and conflict on every retry until
// the cache entry expired. The caller is expected to re-read and retry (or, for the
// heartbeat path, simply let the next message re-report the state).
func (s *AgentService) SaveAgent(ctx context.Context, agent *agentmodel.Agent) error {
	err := s.agentPersistencePort.PutAgent(ctx, agent)
	if err != nil {
		if errors.Is(err, port.ErrConflict) {
			s.InvalidateCache(agent.Metadata.InstanceUID)
		}

		return fmt.Errorf("failed to save agent to persistence: %w", err)
	}

	// Cache a clone to prevent external mutations from affecting cache. PutAgent has
	// bumped agent.Metadata.ResourceVersion on success, so the cached clone carries
	// the new version and the next SaveAgent from this process uses the right token.
	if s.cacheEnabled {
		s.agentCache.Set(agent.Metadata.InstanceUID, agent.Clone(), ttlcache.DefaultTTL)
	}

	return nil
}

// DeleteAgent permanently (hard) removes a disconnected agent by its instance UID
// and invalidates the cache.
//
// The "only disconnected agents may be deleted" policy is enforced here so it
// cannot be bypassed by callers that hold an AgentUsecase directly. The agent is
// read fresh from persistence (not the cache) so the decision reflects the agent's
// real current state — important right after a disconnect, where another server's
// write may not be visible in this process's cache yet. A still-connected agent is
// rejected with [agentport.ErrAgentConnected].
func (s *AgentService) DeleteAgent(ctx context.Context, instanceUID uuid.UUID) error {
	agent, err := s.agentPersistencePort.GetAgent(ctx, instanceUID)
	if err != nil {
		return fmt.Errorf("failed to get agent for deletion: %w", err)
	}

	if agent.IsConnectedAt(s.clock.Now(), agentmodel.DefaultConnectionStaleness) {
		return fmt.Errorf("failed to delete agent: %w", agentport.ErrAgentConnected)
	}

	err = s.agentPersistencePort.DeleteAgent(ctx, instanceUID)
	if err != nil {
		return fmt.Errorf("failed to delete agent from persistence: %w", err)
	}

	s.InvalidateCache(instanceUID)

	return nil
}

// ListAgents retrieves agents filtered by namespace from the persistence layer.
func (s *AgentService) ListAgents(
	ctx context.Context,
	namespace string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	res, err := s.agentPersistencePort.ListAgents(ctx, namespace, options)
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
	namespace string,
	query string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	resp, err := s.agentPersistencePort.SearchAgents(ctx, namespace, query, options)
	if err != nil {
		return nil, fmt.Errorf("failed to search agents: %w", err)
	}

	return resp, nil
}
