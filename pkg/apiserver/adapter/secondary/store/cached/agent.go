// Package cached implements stores with node-local read caches over persistence.
package cached

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jellydator/ttlcache/v3"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var _ agentport.AgentStore = (*AgentStore)(nil)

const (
	defaultAgentCacheTTL      = 30 * time.Second
	defaultAgentCacheCapacity = 1000
)

// AgentCacheConfig configures the optional agent read cache.
type AgentCacheConfig struct {
	Enabled     bool
	TTL         time.Duration
	MaxCapacity int64
}

// DefaultAgentCacheConfig returns the existing bounded cache defaults.
func DefaultAgentCacheConfig() AgentCacheConfig {
	return AgentCacheConfig{Enabled: true, TTL: defaultAgentCacheTTL, MaxCapacity: defaultAgentCacheCapacity}
}

// AgentStore caches document reads; lists and liveness writes use persistence
// directly. Liveness is overlaid by the service rather than frozen in this cache.
type AgentStore struct {
	persistence agentport.AgentPersistencePort
	cache       *ttlcache.Cache[uuid.UUID, *agentmodel.Agent]
}

// NewAgentStore wraps persistence with an optional cache.
func NewAgentStore(persistence agentport.AgentPersistencePort, config AgentCacheConfig) *AgentStore {
	store := &AgentStore{persistence: persistence, cache: nil}
	if !config.Enabled {
		return store
	}

	defaults := DefaultAgentCacheConfig()
	if config.TTL <= 0 {
		config.TTL = defaults.TTL
	}

	if config.MaxCapacity <= 0 {
		config.MaxCapacity = defaults.MaxCapacity
	}

	store.cache = ttlcache.New[uuid.UUID, *agentmodel.Agent](
		ttlcache.WithTTL[uuid.UUID, *agentmodel.Agent](config.TTL),
		ttlcache.WithCapacity[uuid.UUID, *agentmodel.Agent](uint64(config.MaxCapacity)),
		// Heartbeats must eventually reload desired state even if a cache-invalidation event is lost.
		ttlcache.WithDisableTouchOnHit[uuid.UUID, *agentmodel.Agent](),
	)

	return store
}

// Get returns an isolated copy, reading through the cache when enabled.
func (s *AgentStore) Get(ctx context.Context, uid uuid.UUID) (*agentmodel.Agent, error) {
	if s.cache != nil {
		if item := s.cache.Get(uid); item != nil {
			return item.Value().Clone(), nil
		}
	}

	agent, err := s.GetFresh(ctx, uid)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		s.cache.Set(uid, agent.Clone(), ttlcache.DefaultTTL)
	}

	return agent, nil
}

// GetFresh bypasses the local cache.
func (s *AgentStore) GetFresh(ctx context.Context, uid uuid.UUID) (*agentmodel.Agent, error) {
	agent, err := s.persistence.GetAgent(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("read agent: %w", err)
	}

	return agent, nil
}

// Put refreshes the cache only after persistence accepts the write. A
// conflict drops the losing version so the next read can observe the winner.
func (s *AgentStore) Put(ctx context.Context, agent *agentmodel.Agent) error {
	err := s.persistence.PutAgent(ctx, agent)
	if err != nil {
		if errors.Is(err, model.ErrConflict) {
			s.InvalidateCache(agent.Metadata.InstanceUID)
		}

		return fmt.Errorf("persist agent: %w", err)
	}

	if s.cache != nil {
		s.cache.Set(agent.Metadata.InstanceUID, agent.Clone(), ttlcache.DefaultTTL)
	}

	return nil
}

// Delete invalidates cached data after a successful durable deletion.
func (s *AgentStore) Delete(ctx context.Context, uid uuid.UUID) error {
	err := s.persistence.DeleteAgent(ctx, uid)
	if err != nil {
		return fmt.Errorf("delete agent: %w", err)
	}

	s.InvalidateCache(uid)

	return nil
}

// UpdateAgentLiveness writes narrow liveness fields directly to persistence.
func (s *AgentStore) UpdateAgentLiveness(ctx context.Context, liveness *agentmodel.AgentLiveness) error {
	err := s.persistence.UpdateAgentLiveness(ctx, liveness)
	if err != nil {
		return fmt.Errorf("update agent liveness: %w", err)
	}

	return nil
}

// ListAgents reads a namespace's agents directly from persistence.
func (s *AgentStore) ListAgents(ctx context.Context, namespace string,
	options *model.ListOptions) (*model.ListResponse[*agentmodel.Agent], error) {
	agents, err := s.persistence.ListAgents(ctx, namespace, options)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}

	return agents, nil
}

// ListAgentsBySelector reads matching agents directly from persistence.
func (s *AgentStore) ListAgentsBySelector(ctx context.Context, selector agentmodel.AgentSelector,
	options *model.ListOptions) (*model.ListResponse[*agentmodel.Agent], error) {
	agents, err := s.persistence.ListAgentsBySelector(ctx, selector, options)
	if err != nil {
		return nil, fmt.Errorf("list agents by selector: %w", err)
	}

	return agents, nil
}

// SearchAgents searches persisted agents without freezing liveness in the cache.
func (s *AgentStore) SearchAgents(ctx context.Context, namespace, query string,
	options *model.ListOptions) (*model.ListResponse[*agentmodel.Agent], error) {
	agents, err := s.persistence.SearchAgents(ctx, namespace, query, options)
	if err != nil {
		return nil, fmt.Errorf("search agents: %w", err)
	}

	return agents, nil
}

// InvalidateCache forces the next read to persistence.
func (s *AgentStore) InvalidateCache(uid uuid.UUID) {
	if s.cache != nil {
		s.cache.Delete(uid)
	}
}

// Shutdown releases the local cache.
func (s *AgentStore) Shutdown() {
	if s.cache != nil {
		s.cache.DeleteAll()
		s.cache.Stop()
	}
}
