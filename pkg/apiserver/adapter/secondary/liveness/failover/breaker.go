package failover

import (
	"sync"
	"time"

	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

// State is the circuit breaker's current disposition toward the primary tier.
type State int

// Breaker states.
const (
	// StateClosed routes to the primary tier: it is answering.
	StateClosed State = iota
	// StateOpen routes to the fallback: the primary failed often enough that
	// continuing to try it would only add latency to every call.
	StateOpen
	// StateHalfOpen lets a single call through to see whether the primary
	// recovered, while everything else still routes to the fallback.
	StateHalfOpen
)

// String returns the state's name, for logs and metric labels.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

const (
	// DefaultFailureThreshold is how many consecutive failures trip the breaker.
	// Kept low: the primary is an accelerator, so giving up on it early costs only
	// database writes, while retrying a dead one costs latency on every agent message.
	DefaultFailureThreshold = 3
	// DefaultProbeInterval is how long the breaker stays open before letting one
	// call through to test for recovery.
	DefaultProbeInterval = 30 * time.Second
)

// breaker is a consecutive-failure circuit breaker.
//
// Recovery needs no restart and no config change: once the probe interval has
// passed, the next call goes to the primary, and a success closes the breaker.
type breaker struct {
	mu sync.Mutex

	state            State
	consecutiveFails int
	openedAt         time.Time

	clock            clock.PassiveClock
	failureThreshold int
	probeInterval    time.Duration
}

func newBreaker(clk clock.PassiveClock, failureThreshold int, probeInterval time.Duration) *breaker {
	if failureThreshold <= 0 {
		failureThreshold = DefaultFailureThreshold
	}

	if probeInterval <= 0 {
		probeInterval = DefaultProbeInterval
	}

	return &breaker{
		mu:               sync.Mutex{},
		state:            StateClosed,
		consecutiveFails: 0,
		openedAt:         time.Time{},
		clock:            clk,
		failureThreshold: failureThreshold,
		probeInterval:    probeInterval,
	}
}

// State returns the breaker's current state.
func (b *breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.state
}

// allow reports whether the caller may use the primary tier, transitioning an open
// breaker to half-open once the probe interval has elapsed.
func (b *breaker) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state == StateOpen && b.clock.Now().Sub(b.openedAt) >= b.probeInterval {
		b.state = StateHalfOpen
	}

	return b.state != StateOpen
}

// recordSuccess closes the breaker. It returns true when this call is what closed
// it, so the caller can log the recovery exactly once.
func (b *breaker) recordSuccess() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.consecutiveFails = 0

	if b.state == StateClosed {
		return false
	}

	b.state = StateClosed

	return true
}

// recordFailure counts a failure and opens the breaker once the threshold is
// reached. It returns true when this call is what opened it, so the caller can log
// and force a flush exactly once per outage rather than once per failed call.
func (b *breaker) recordFailure() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state == StateOpen {
		return false
	}

	b.consecutiveFails++

	// A failed probe sends the breaker straight back to open: the primary was given
	// its chance and did not take it.
	if b.state == StateHalfOpen || b.consecutiveFails >= b.failureThreshold {
		b.state = StateOpen
		b.openedAt = b.clock.Now()

		return true
	}

	return false
}
