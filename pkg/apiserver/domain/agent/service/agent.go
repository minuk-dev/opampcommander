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

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var (
	_ agentport.AgentUsecase          = (*AgentService)(nil)
	_ agentport.AgentCacheInvalidator = (*AgentService)(nil)
)

const (
	// DefaultAgentCacheTTL is the default time-to-live for agent cache entries.
	DefaultAgentCacheTTL = 30 * time.Second
	// DefaultAgentCacheCapacity is the default maximum number of agent cache entries.
	DefaultAgentCacheCapacity int64 = 1000
	// DefaultLivenessPersistThrottle is the minimum interval between write-throughs of
	// an agent whose only change is that it is still alive. Bursts of heartbeats in the
	// same window share a single durable write; anything that changed durable state is
	// written immediately by its own call to SaveAgent.
	//
	// Kept short (10s) so the API surface still reflects per-agent SequenceNum and
	// LastReportedAt within an order of magnitude of the OpAMP heartbeat interval even
	// with no shared liveness tier configured.
	DefaultLivenessPersistThrottle = 10 * time.Second
)

// AgentLivenessConfig holds the configuration of the liveness fast tier.
type AgentLivenessConfig struct {
	// PersistThrottle is the minimum interval between write-throughs of an agent
	// whose only change is its liveness. Zero selects [DefaultLivenessPersistThrottle].
	PersistThrottle time.Duration
}

// AgentCacheConfig holds the configuration for agent caching.
type AgentCacheConfig struct {
	Enabled     bool
	TTL         time.Duration
	MaxCapacity int64
}

// DefaultAgentLivenessConfig returns the liveness configuration used when no
// explicit configuration is supplied.
func DefaultAgentLivenessConfig() AgentLivenessConfig {
	return AgentLivenessConfig{
		PersistThrottle: DefaultLivenessPersistThrottle,
	}
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
	// clock is consulted for the delete connection-guard (staleness evaluation) and
	// for the liveness write-through throttle.
	clock clock.PassiveClock

	// agentLivenessPort is the fast tier for liveness state. It is a pure
	// accelerator: every error from it is logged and swallowed, and the service
	// behaves as if the record were simply absent.
	agentLivenessPort       agentport.AgentLivenessPort
	livenessMetricsPort     agentport.AgentLivenessMetricsPort
	livenessPersistThrottle time.Duration
}

// DefaultAgentCacheConfig returns the cache configuration used when no explicit
// configuration is supplied (caching enabled with the default TTL/capacity).
func DefaultAgentCacheConfig() AgentCacheConfig {
	return AgentCacheConfig{
		Enabled:     true,
		TTL:         DefaultAgentCacheTTL,
		MaxCapacity: DefaultAgentCacheCapacity,
	}
}

// NewAgentService creates a new instance of AgentService.
// defaultNamespace is the namespace assigned to newly-seen agents without a service.namespace;
// an empty value falls back to agentmodel.DefaultNamespaceName.
func NewAgentService(
	agentPersistencePort agentport.AgentPersistencePort,
	agentLivenessPort agentport.AgentLivenessPort,
	livenessMetricsPort agentport.AgentLivenessMetricsPort,
	logger *slog.Logger,
	cacheConfig AgentCacheConfig,
	livenessConfig AgentLivenessConfig,
	defaultNamespace string,
	passiveClock clock.PassiveClock,
) *AgentService {
	if passiveClock == nil {
		passiveClock = clock.RealClock{}
	}

	if defaultNamespace == "" {
		defaultNamespace = agentmodel.DefaultNamespaceName
	}

	persistThrottle := clampPersistThrottle(livenessConfig.PersistThrottle, logger)

	if !cacheConfig.Enabled {
		logger.Info("agent cache disabled")

		return &AgentService{
			agentPersistencePort:    agentPersistencePort,
			agentLivenessPort:       agentLivenessPort,
			livenessMetricsPort:     livenessMetricsPort,
			logger:                  logger,
			agentCache:              nil,
			cacheEnabled:            false,
			defaultNamespace:        defaultNamespace,
			clock:                   passiveClock,
			livenessPersistThrottle: persistThrottle,
		}
	}

	agentCache := newAgentCache(cacheConfig, logger)

	return &AgentService{
		agentPersistencePort:    agentPersistencePort,
		agentLivenessPort:       agentLivenessPort,
		livenessMetricsPort:     livenessMetricsPort,
		logger:                  logger,
		agentCache:              agentCache,
		cacheEnabled:            true,
		defaultNamespace:        defaultNamespace,
		clock:                   passiveClock,
		livenessPersistThrottle: persistThrottle,
	}
}

// clampPersistThrottle keeps the message-path write-through throttle inside the
// staleness budget.
//
// The throttle is how long an agent may keep heartbeating without its stored
// document being refreshed. Let it exceed the budget and the datastore's own
// connected-agent filter — and the agent-group connected counts, which no read-side
// overlay can reach — start reporting live agents as disconnected.
func clampPersistThrottle(throttle time.Duration, logger *slog.Logger) time.Duration {
	if throttle <= 0 {
		throttle = DefaultLivenessPersistThrottle
	}

	budget := MaxLivenessStalenessBudget()
	if throttle > budget {
		logger.Warn("liveness persist throttle clamped to fit the connection staleness window",
			slog.Duration("requested", throttle),
			slog.Duration("applied", budget),
			slog.Duration("staleness", agentmodel.DefaultConnectionStaleness),
		)

		return budget
	}

	return throttle
}

// newAgentCache builds the read cache, falling back to the package defaults for
// unset or nonsensical bounds.
func newAgentCache(
	cacheConfig AgentCacheConfig,
	logger *slog.Logger,
) *ttlcache.Cache[uuid.UUID, *agentmodel.Agent] {
	ttl := cacheConfig.TTL
	if ttl <= 0 {
		ttl = DefaultAgentCacheTTL
	}

	capacity := cacheConfig.MaxCapacity
	if capacity <= 0 {
		capacity = DefaultAgentCacheCapacity
	}

	logger.Info("agent cache initialized",
		slog.Duration("ttl", ttl),
		slog.Int64("maxCapacity", capacity),
	)

	return ttlcache.New[uuid.UUID, *agentmodel.Agent](
		ttlcache.WithTTL[uuid.UUID, *agentmodel.Agent](ttl),
		ttlcache.WithCapacity[uuid.UUID, *agentmodel.Agent](uint64(capacity)),
	)
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

// GetAgent retrieves an agent by its instance UID, with its liveness fields
// resolved from the fast tier.
func (s *AgentService) GetAgent(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.Agent, error) {
	agent, err := s.getStoredAgent(ctx, instanceUID)
	if err != nil {
		return nil, err
	}

	s.mergeLiveness(ctx, agent)

	return agent, nil
}

// GetOrCreateAgent retrieves an agent by its instance UID.
// If the agent doesn't exist, it creates a new one with default values.
//
// It deliberately does not merge fast-tier liveness. Its only caller is the OpAMP
// message path, which overwrites every liveness field from the message it is
// handling before doing anything with the agent — so merging here would spend a
// fast-tier round trip per agent message on values that are discarded microseconds
// later.
func (s *AgentService) GetOrCreateAgent(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.Agent, error) {
	agent, err := s.getStoredAgent(ctx, instanceUID)
	if err != nil {
		if errors.Is(err, model.ErrResourceNotExist) {
			agent = agentmodel.NewAgent(instanceUID, agentmodel.WithNamespace(s.defaultNamespace))
		} else {
			return nil, fmt.Errorf("failed to get agent: %w", err)
		}
	}

	return agent, nil
}

// SaveAgent saves the agent to the persistence layer with optimistic concurrency:
// the write is rejected with [model.ErrConflict] when another writer (another HA
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
		if errors.Is(err, model.ErrConflict) {
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

	s.markLivenessPersisted(ctx, agent.Metadata.InstanceUID, agent.Status.LastReportedAt,
		agentport.LivenessWriteShapeDocument)

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

	// The durable document lags the fast tier by up to one write-through interval,
	// so consult the fast tier before declaring the agent disconnected — otherwise
	// a live agent could be deleted out from under its own WebSocket.
	s.mergeLiveness(ctx, agent)

	if agent.IsConnectedAt(s.clock.Now(), agentmodel.DefaultConnectionStaleness) {
		return fmt.Errorf("failed to delete agent: %w", agentport.ErrAgentConnected)
	}

	err = s.agentPersistencePort.DeleteAgent(ctx, instanceUID)
	if err != nil {
		return fmt.Errorf("failed to delete agent from persistence: %w", err)
	}

	// Drop the fast-tier record too, or a read path merging liveness over the
	// durable store would resurrect the deleted agent's connection state.
	_ = s.ForgetAgentLiveness(ctx, instanceUID)

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

	s.mergeLivenessInto(ctx, res.Items)

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

	s.mergeLivenessInto(ctx, resp.Items)

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

	s.mergeLivenessInto(ctx, resp.Items)

	return resp, nil
}

// getStoredAgent returns the agent as the durable store holds it, reading through
// the local cache. Liveness is deliberately not merged here: the cache holds the
// durable document, so the fast tier is consulted per read rather than frozen into
// a cache entry for the whole TTL.
func (s *AgentService) getStoredAgent(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.Agent, error) {
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
