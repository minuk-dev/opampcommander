// Package inmemory provides the node-local implementation of the agent liveness
// port.
//
// It is the default and the fallback: with no shared fast tier configured — or
// when the configured one is unavailable — each server keeps its own view of the
// agents connected to it, which is exactly the behaviour the OpAMP service had
// when it tracked heartbeat timestamps in a process-local map.
//
// Records are held only for agents this node has seen, and are evicted once they
// go untouched for longer than the TTL. HTTP-polling agents never signal a
// disconnect, so without that sweep their records would accumulate for the
// lifetime of the process.
package inmemory

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ agentport.AgentLivenessPort = (*Store)(nil)

// Store is the node-local implementation of [agentport.AgentLivenessPort].
type Store struct {
	mu      sync.RWMutex
	records map[uuid.UUID]*agentmodel.AgentLiveness

	clock      clock.PassiveClock
	ttl        time.Duration
	gcInterval time.Duration
}

// New creates an empty node-local liveness store.
func New(config Config, passiveClock clock.PassiveClock) *Store {
	if passiveClock == nil {
		passiveClock = clock.NewRealClock()
	}

	return &Store{
		mu:         sync.RWMutex{},
		records:    make(map[uuid.UUID]*agentmodel.AgentLiveness),
		clock:      passiveClock,
		ttl:        config.effectiveTTL(),
		gcInterval: config.effectiveGCInterval(),
	}
}

// GCInterval returns how often [Store.GC] should be called.
func (s *Store) GCInterval() time.Duration {
	return s.gcInterval
}

// Touch implements [agentport.AgentLivenessPort].
//
// The write-through anchor is carried over from the record already held rather
// than taken from the caller, so recording an observation can never reset the
// throttle window.
func (s *Store) Touch(
	_ context.Context,
	liveness *agentmodel.AgentLiveness,
) (*agentmodel.AgentLiveness, error) {
	if liveness == nil {
		return nil, nil //nolint:nilnil // nothing to record and nothing to report
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	stored := liveness.Clone()
	if previous, found := s.records[liveness.InstanceUID]; found {
		stored.LastPersistedAt = previous.LastPersistedAt
	}

	s.records[liveness.InstanceUID] = stored

	return stored.Clone(), nil
}

// MarkPersisted implements [agentport.AgentLivenessPort].
//
// A record that is not held is created from the anchor alone: the agent reached
// the durable store, which is worth remembering even if its observation has since
// expired here.
func (s *Store) MarkPersisted(_ context.Context, instanceUID uuid.UUID, persistedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, found := s.records[instanceUID]
	if !found {
		//exhaustruct:ignore // only the anchor is known at this point
		record = &agentmodel.AgentLiveness{InstanceUID: instanceUID}
	}

	record.LastPersistedAt = persistedAt
	s.records[instanceUID] = record

	return nil
}

// Get implements [agentport.AgentLivenessPort].
func (s *Store) Get(_ context.Context, instanceUID uuid.UUID) (*agentmodel.AgentLiveness, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.lookup(instanceUID), nil
}

// GetMany implements [agentport.AgentLivenessPort].
func (s *Store) GetMany(
	_ context.Context,
	instanceUIDs []uuid.UUID,
) (map[uuid.UUID]*agentmodel.AgentLiveness, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return lo.FilterSliceToMap(instanceUIDs,
		func(instanceUID uuid.UUID) (uuid.UUID, *agentmodel.AgentLiveness, bool) {
			record := s.lookup(instanceUID)

			return instanceUID, record, record != nil
		}), nil
}

// Delete implements [agentport.AgentLivenessPort].
func (s *Store) Delete(_ context.Context, instanceUID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.records, instanceUID)

	return nil
}

// Name implements the background-runner contract used by the apiserver's
// scheduler executor.
func (s *Store) Name() string {
	return "agent-liveness-inmemory-gc"
}

// Run sweeps expired records on a ticker until the context is cancelled.
//
// The store has no backing TTL of its own, so without this sweep records for
// HTTP-polling agents — which never signal a disconnect — would be held for the
// lifetime of the process. Reads already treat an expired record as absent, so
// the sweep only reclaims memory and never changes an answer.
func (s *Store) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.gcInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("agent liveness gc loop exited: %w", ctx.Err())
		case <-ticker.C:
			s.GC()
		}
	}
}

// GC evicts every record that has gone untouched for longer than the TTL and
// returns how many were dropped.
func (s *Store) GC() int {
	now := s.clock.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	before := len(s.records)

	maps.DeleteFunc(s.records, func(_ uuid.UUID, record *agentmodel.AgentLiveness) bool {
		return record.IsExpiredAt(now, s.ttl)
	})

	return before - len(s.records)
}

// Len returns the number of records currently held.
func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.records)
}

// lookup returns a clone of the record for instanceUID, treating an expired
// record as absent so a read never resurrects state the next GC would drop.
// Callers must hold at least the read lock.
func (s *Store) lookup(instanceUID uuid.UUID) *agentmodel.AgentLiveness {
	record, found := s.records[instanceUID]
	if !found || record.IsExpiredAt(s.clock.Now(), s.ttl) {
		return nil
	}

	return record.Clone()
}
