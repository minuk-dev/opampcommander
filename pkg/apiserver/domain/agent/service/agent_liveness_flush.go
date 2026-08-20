package agentservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"k8s.io/utils/clock"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

const (
	// DefaultLivenessFlushInterval is how often liveness observations absorbed by the
	// fast tier are written through to the durable store.
	DefaultLivenessFlushInterval = 30 * time.Second
	// DefaultLivenessFlushStaleAfter is how far behind a stored document must fall
	// before the flush claims it.
	DefaultLivenessFlushStaleAfter = 30 * time.Second
	// DefaultLivenessFlushBatchSize bounds how many agents one flush cycle writes.
	// A cycle issues its writes sequentially, so the batch also spreads them across
	// the interval rather than firing them as one burst at the tick.
	DefaultLivenessFlushBatchSize = 2000
	// livenessStalenessHeadroomDivisor sets how much of the connection-staleness
	// window the flush must leave unused: one part in this many.
	livenessStalenessHeadroomDivisor = 3
	// clampedCadenceHalves splits the budget evenly when a requested ratio cannot be
	// preserved (a degenerate zero on one side).
	clampedCadenceHalves = 2
)

// MaxLivenessStalenessBudget is the largest staleness the flush may allow the stored
// document to reach.
//
// This is the number that matters, and it is not the flush interval alone. A stored
// document goes stale by up to Interval + StaleAfter: the flush only claims agents
// already StaleAfter behind, and it only looks every Interval. Bounding just the
// interval — as an earlier version of this did — lets the sum drift past the window,
// at which point the datastore's own connected-agent filter and the agent-group
// connected counts start reporting live agents as disconnected. Neither is reachable
// from the read-side overlay, because both are evaluated inside the datastore.
func MaxLivenessStalenessBudget() time.Duration {
	staleness := agentmodel.DefaultConnectionStaleness

	return staleness - staleness/livenessStalenessHeadroomDivisor
}

// AgentLivenessFlusher writes liveness observations absorbed by the fast tier
// through to the durable store on a slow cadence.
//
// It is the write-behind half of the fast tier, and the reason losing that tier is
// survivable rather than fleet-wide: heartbeats stop costing a durable write each,
// but the durable store never falls more than one flush interval behind.
type AgentLivenessFlusher struct {
	agentUsecase      agentport.AgentUsecase
	agentLivenessPort agentport.AgentLivenessPort
	logger            *slog.Logger
	clock             clock.PassiveClock

	interval   time.Duration
	batchSize  int
	staleAfter time.Duration
}

// AgentLivenessFlushConfig holds the flusher's tuning knobs. Zero values fall back
// to the package defaults.
type AgentLivenessFlushConfig struct {
	// Interval is how often the flush cycle runs.
	Interval time.Duration
	// StaleAfter is how far behind a stored document must fall before the flush
	// claims it.
	StaleAfter time.Duration
	// BatchSize bounds how many agents one cycle writes.
	BatchSize int
}

// DefaultAgentLivenessFlushConfig returns the flush configuration used when no
// explicit configuration is supplied.
func DefaultAgentLivenessFlushConfig() AgentLivenessFlushConfig {
	return AgentLivenessFlushConfig{
		Interval:   DefaultLivenessFlushInterval,
		StaleAfter: DefaultLivenessFlushStaleAfter,
		BatchSize:  DefaultLivenessFlushBatchSize,
	}
}

// NewAgentLivenessFlusher creates the write-behind flusher.
func NewAgentLivenessFlusher(
	agentUsecase agentport.AgentUsecase,
	agentLivenessPort agentport.AgentLivenessPort,
	config AgentLivenessFlushConfig,
	logger *slog.Logger,
	passiveClock clock.PassiveClock,
) *AgentLivenessFlusher {
	if passiveClock == nil {
		passiveClock = clock.RealClock{}
	}

	interval, staleAfter := clampToStalenessBudget(config.Interval, config.StaleAfter, logger)

	return &AgentLivenessFlusher{
		agentUsecase:      agentUsecase,
		agentLivenessPort: agentLivenessPort,
		logger:            logger,
		clock:             passiveClock,
		interval:          interval,
		staleAfter:        staleAfter,
		batchSize:         defaultedBatchSize(config.BatchSize),
	}
}

// clampToStalenessBudget scales the interval and cutoff down together until their
// sum fits the staleness budget.
//
// They are clamped as a pair because it is their sum, not either one, that bounds how
// stale a stored document gets. Scaling preserves whatever ratio the operator asked
// for instead of silently substituting defaults.
func clampToStalenessBudget(interval, staleAfter time.Duration, logger *slog.Logger) (time.Duration, time.Duration) {
	if interval <= 0 {
		interval = DefaultLivenessFlushInterval
	}

	if staleAfter <= 0 {
		staleAfter = DefaultLivenessFlushStaleAfter
	}

	budget := MaxLivenessStalenessBudget()

	total := interval + staleAfter
	if total <= budget {
		return interval, staleAfter
	}

	// Scale in floating point: durations multiply into nanoseconds, and a few
	// minutes times a minute overflows int64 long before the ratio is computed.
	scaledInterval := time.Duration(float64(budget) * (float64(interval) / float64(total)))
	if scaledInterval <= 0 {
		scaledInterval = budget / clampedCadenceHalves
	}

	scaledStaleAfter := budget - scaledInterval
	if scaledStaleAfter <= 0 {
		scaledStaleAfter = budget / clampedCadenceHalves
		scaledInterval = budget - scaledStaleAfter
	}

	logger.Warn("liveness flush cadence clamped to fit the connection staleness window",
		slog.Duration("requestedInterval", interval),
		slog.Duration("requestedStaleAfter", staleAfter),
		slog.Duration("appliedInterval", scaledInterval),
		slog.Duration("appliedStaleAfter", scaledStaleAfter),
		slog.Duration("budget", budget),
		slog.Duration("staleness", agentmodel.DefaultConnectionStaleness),
	)

	return scaledInterval, scaledStaleAfter
}

func defaultedBatchSize(batchSize int) int {
	if batchSize <= 0 {
		return DefaultLivenessFlushBatchSize
	}

	return batchSize
}

// Name implements the background-runner contract used by the apiserver's scheduler
// executor.
func (f *AgentLivenessFlusher) Name() string {
	return "agent-liveness-flusher"
}

// Interval returns the flush cadence actually in effect, after clamping.
func (f *AgentLivenessFlusher) Interval() time.Duration {
	return f.interval
}

// StaleAfter returns the write-through cutoff actually in effect, after clamping.
func (f *AgentLivenessFlusher) StaleAfter() time.Duration {
	return f.staleAfter
}

// Run flushes on a ticker until the context is cancelled.
func (f *AgentLivenessFlusher) Run(ctx context.Context) error {
	ticker := time.NewTicker(f.interval)
	defer ticker.Stop()

	f.logger.Info("agent liveness write-behind flusher started",
		slog.Duration("interval", f.interval),
		slog.Int("batchSize", f.batchSize),
	)

	for {
		select {
		case <-ctx.Done():
			// Drain what is held before going away, so a graceful shutdown does not
			// leave the durable store an interval behind.
			f.flushQuietly(context.WithoutCancel(ctx))

			return fmt.Errorf("agent liveness flush loop exited: %w", ctx.Err())
		case <-ticker.C:
			f.flushQuietly(ctx)
		}
	}
}

// Flush writes every pending observation through to the durable store and returns
// how many agents were written.
//
// It is also the forced-flush entry point: a fast tier that is about to become
// unavailable calls this so its in-process state reaches the durable store before
// reads start falling back to it.
func (f *AgentLivenessFlusher) Flush(ctx context.Context) (int, error) {
	cutoff := f.clock.Now().Add(-f.staleAfter)

	pending, err := f.agentLivenessPort.ListPendingWriteThrough(ctx, cutoff, f.batchSize)
	if err != nil {
		return 0, fmt.Errorf("failed to list agents pending write-through: %w", err)
	}

	if len(pending) == 0 {
		return 0, nil
	}

	written := 0

	for _, record := range pending {
		if ctx.Err() != nil {
			break
		}

		if f.flushOne(ctx, record) {
			written++
		}
	}

	if len(pending) == f.batchSize {
		f.logger.Warn("liveness flush batch saturated; remaining agents wait for the next cycle",
			slog.Int("batchSize", f.batchSize),
		)
	}

	return written, nil
}

// flushOne writes a single agent's liveness through, reporting whether the write
// happened.
//
// It uses the record the index already handed us rather than re-reading the agent:
// the record carries everything the durable store needs, so a flush cycle costs one
// narrow datastore write per agent and nothing else.
func (f *AgentLivenessFlusher) flushOne(ctx context.Context, record *agentmodel.AgentLiveness) bool {
	err := f.agentUsecase.PersistAgentLiveness(ctx, record)
	if err == nil {
		return true
	}

	// The agent was deleted while its record lingered. Drop the record so the flush
	// stops chasing it every cycle.
	if errors.Is(err, model.ErrResourceNotExist) {
		forgetErr := f.agentUsecase.ForgetAgentLiveness(ctx, record.InstanceUID)
		if forgetErr != nil {
			f.logger.Warn("failed to forget liveness for a deleted agent",
				slog.String("instanceUID", record.InstanceUID.String()),
				slog.String("error", forgetErr.Error()),
			)
		}

		return false
	}

	f.logger.Warn("failed to flush agent liveness",
		slog.String("instanceUID", record.InstanceUID.String()),
		slog.String("error", err.Error()),
	)

	return false
}

func (f *AgentLivenessFlusher) flushQuietly(ctx context.Context) {
	started := f.clock.Now()

	written, err := f.Flush(ctx)
	if err != nil {
		f.logger.Warn("agent liveness flush cycle failed", slog.String("error", err.Error()))

		return
	}

	if written > 0 {
		f.logger.Debug("flushed agent liveness to the durable store",
			slog.Int("agents", written),
			slog.Duration("took", f.clock.Since(started)),
		)
	}
}
