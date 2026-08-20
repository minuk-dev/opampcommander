// Package failover makes the agent liveness fast tier optional at runtime, not just
// at configuration time.
//
// It wraps a shared primary tier (Redis) and a node-local fallback behind one port,
// with a circuit breaker between them. When the primary starts failing, calls route
// to the fallback and the server keeps serving with no restart and no config change;
// when it recovers, calls route back on their own.
//
// The fallback is kept warm rather than cold: every write goes to both tiers, so at
// the moment the breaker trips the node-local store already holds this server's
// agents. Without that, a trip would fall back onto an empty store and every agent
// would look brand new.
package failover

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ agentport.AgentLivenessPort = (*Store)(nil)

// Config holds the breaker's tuning knobs. Zero values fall back to the package
// defaults.
type Config struct {
	// FailureThreshold is how many consecutive primary failures trip the breaker.
	FailureThreshold int
	// ProbeInterval is how long the breaker stays open before testing for recovery.
	ProbeInterval time.Duration
}

// Store routes agent liveness between a primary and a fallback tier.
type Store struct {
	primary  agentport.AgentLivenessPort
	fallback agentport.AgentLivenessPort

	breaker *breaker
	metrics agentport.AgentLivenessMetricsPort
	logger  *slog.Logger

	// onTrip runs once each time the breaker opens. It is how the write-behind
	// flush is forced: the fallback holds observations the primary was supposed to
	// keep, and they must reach the durable store before reads start relying on it.
	onTrip func(context.Context)
	// flushes tracks the forced flushes still in flight, so shutdown can wait for
	// them instead of dropping observations mid-write.
	flushes sync.WaitGroup
}

// New wraps a primary tier with a node-local fallback and a circuit breaker.
func New(
	primary agentport.AgentLivenessPort,
	fallback agentport.AgentLivenessPort,
	metrics agentport.AgentLivenessMetricsPort,
	config Config,
	passiveClock clock.PassiveClock,
	logger *slog.Logger,
) *Store {
	if passiveClock == nil {
		passiveClock = clock.NewRealClock()
	}

	store := &Store{
		primary:  primary,
		fallback: fallback,
		breaker:  newBreaker(passiveClock, config.FailureThreshold, config.ProbeInterval),
		metrics:  metrics,
		logger:   logger,
		onTrip:   nil,
		flushes:  sync.WaitGroup{},
	}

	// Publish the starting state so the gauge reads "closed" from boot rather than
	// only after the first transition.
	metrics.RecordBreakerState(store.State().String())

	return store
}

// OnTrip registers a callback run once each time the breaker opens.
//
// It is set after construction because the natural callback — forcing a
// write-behind flush — depends on this store, so the two cannot be built in one pass.
func (s *Store) OnTrip(callback func(context.Context)) {
	s.onTrip = callback
}

// Wait blocks until every forced flush has finished.
//
// A trip fires its flush asynchronously so an agent message is never held up by it;
// this is how shutdown collects those goroutines rather than exiting mid-write.
func (s *Store) Wait() {
	s.flushes.Wait()
}

// State returns the breaker's current state, for health reporting and metrics.
func (s *Store) State() State {
	return s.breaker.State()
}

// Touch implements [agentport.AgentLivenessPort].
//
// The write always reaches the fallback, whatever the primary does. That is what
// keeps the node-local tier warm enough to take over the instant the breaker trips.
//
// The record the caller gets back is the primary's while the breaker is closed and
// the fallback's otherwise, so the write-through anchor it carries always comes from
// whichever tier is currently answering reads.
func (s *Store) Touch(
	ctx context.Context,
	liveness *agentmodel.AgentLiveness,
) (*agentmodel.AgentLiveness, error) {
	local, localErr := s.fallback.Touch(ctx, liveness)

	if s.breaker.allow() {
		shared, err := s.primary.Touch(ctx, liveness)
		if !s.record(ctx, "touch", err) {
			return shared, nil
		}
	}

	// The observation is safe in the fallback either way, so a primary failure is
	// not worth reporting to a caller who can do nothing about it.
	if localErr != nil {
		return nil, fmt.Errorf("failed to record agent liveness in the fallback tier: %w", localErr)
	}

	return local, nil
}

// MarkPersisted implements [agentport.AgentLivenessPort].
//
// Both tiers are anchored: the fallback so it stays a usable stand-in, and the
// primary so a later recovery does not resurrect an already-written observation.
func (s *Store) MarkPersisted(ctx context.Context, instanceUID uuid.UUID, reportedAt time.Time) error {
	_ = s.fallback.MarkPersisted(ctx, instanceUID, reportedAt)

	if !s.breaker.allow() {
		return nil
	}

	err := s.primary.MarkPersisted(ctx, instanceUID, reportedAt)
	s.record(ctx, "markPersisted", err)

	return nil
}

// Get implements [agentport.AgentLivenessPort].
func (s *Store) Get(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.AgentLiveness, error) {
	if s.breaker.allow() {
		record, err := s.primary.Get(ctx, instanceUID)
		if !s.record(ctx, "get", err) {
			return record, nil
		}
	}

	record, err := s.fallback.Get(ctx, instanceUID)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent liveness from the fallback tier: %w", err)
	}

	return record, nil
}

// GetMany implements [agentport.AgentLivenessPort].
func (s *Store) GetMany(
	ctx context.Context,
	instanceUIDs []uuid.UUID,
) (map[uuid.UUID]*agentmodel.AgentLiveness, error) {
	if s.breaker.allow() {
		records, err := s.primary.GetMany(ctx, instanceUIDs)
		if !s.record(ctx, "getMany", err) {
			return records, nil
		}
	}

	records, err := s.fallback.GetMany(ctx, instanceUIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent liveness batch from the fallback tier: %w", err)
	}

	return records, nil
}

// Delete implements [agentport.AgentLivenessPort].
//
// Both tiers are cleared: a record left in the primary would come back the moment
// the breaker closes again.
func (s *Store) Delete(ctx context.Context, instanceUID uuid.UUID) error {
	_ = s.fallback.Delete(ctx, instanceUID)

	if !s.breaker.allow() {
		return nil
	}

	err := s.primary.Delete(ctx, instanceUID)
	s.record(ctx, "delete", err)

	return nil
}

// ListPendingWriteThrough implements [agentport.AgentLivenessPort].
//
// While the breaker is open this answers from the fallback, which is exactly what
// the forced flush needs: the node-local store holds this server's agents and their
// write-through anchors.
func (s *Store) ListPendingWriteThrough(
	ctx context.Context,
	notPersistedSince time.Time,
	limit int,
) ([]*agentmodel.AgentLiveness, error) {
	if s.breaker.allow() {
		records, err := s.primary.ListPendingWriteThrough(ctx, notPersistedSince, limit)
		if !s.record(ctx, "listPendingWriteThrough", err) {
			return records, nil
		}
	}

	records, err := s.fallback.ListPendingWriteThrough(ctx, notPersistedSince, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending agent liveness from the fallback tier: %w", err)
	}

	return records, nil
}

// record folds a primary result into the breaker and reports whether the call
// failed. Transitions are logged once each, not once per call: an outage would
// otherwise produce a log line per agent message.
func (s *Store) record(ctx context.Context, operation string, err error) bool {
	if err == nil {
		if s.breaker.recordSuccess() {
			s.logger.Info("agent liveness primary tier recovered; routing back to it")
		}

		s.metrics.RecordBreakerState(s.breaker.State().String())

		return false
	}

	// Counted per operation, not per outage: this is the volume the shared tier is
	// no longer carrying, which is what an operator watches during a degradation.
	s.metrics.RecordFallback(operation)

	tripped := s.breaker.recordFailure()
	s.metrics.RecordBreakerState(s.breaker.State().String())

	if tripped {
		s.logger.Error("agent liveness primary tier is failing; falling back to node-local state "+
			"and the database until it recovers",
			slog.String("operation", operation),
			slog.String("error", err.Error()),
		)

		s.forceFlush(ctx)
	}

	return true
}

// forceFlush pushes what the fallback holds to the durable store at the moment the
// breaker trips.
//
// Without it the durable store would keep whatever staleness the primary had
// accumulated, and reads falling back to it could report live agents as
// disconnected. The callback is given a context detached from the failing call, so
// the flush is not cancelled by the request that happened to trip the breaker.
func (s *Store) forceFlush(ctx context.Context) {
	if s.onTrip == nil {
		return
	}

	s.flushes.Go(func() {
		s.onTrip(context.WithoutCancel(ctx))
	})
}
