package agentservice // the throttle decision is clock-driven, and only the service's own package can stop its clock

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

// benchClock is a clock the benchmark advances by hand, so a fleet-cadence
// measurement does not take fleet-cadence time.
type benchClock struct{ now time.Time }

func (c *benchClock) Now() time.Time                  { return c.now }
func (c *benchClock) Since(t time.Time) time.Duration { return c.now.Sub(t) }

// countingPersistence counts durable writes by shape and does nothing else. Only the
// two write methods are reached here; the rest would panic, which keeps the fake
// honest about what the benchmark exercises.
type countingPersistence struct {
	agentport.AgentPersistencePort

	documentWrites int
	livenessWrites int
}

func (p *countingPersistence) PutAgent(context.Context, *agentmodel.Agent) error {
	p.documentWrites++

	return nil
}

func (p *countingPersistence) UpdateAgentLiveness(context.Context, *agentmodel.AgentLiveness) error {
	p.livenessWrites++

	return nil
}

// benchLiveness is a minimal in-process fast tier.
type benchLiveness struct {
	mu      sync.Mutex
	records map[uuid.UUID]*agentmodel.AgentLiveness
}

func (s *benchLiveness) Touch(
	_ context.Context,
	liveness *agentmodel.AgentLiveness,
) (*agentmodel.AgentLiveness, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored := liveness.Clone()
	if previous, found := s.records[liveness.InstanceUID]; found {
		stored.DurableReportedAt = previous.DurableReportedAt
	}

	s.records[liveness.InstanceUID] = stored

	return stored.Clone(), nil
}

func (s *benchLiveness) MarkPersisted(_ context.Context, instanceUID uuid.UUID, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, found := s.records[instanceUID]
	if !found {
		//exhaustruct:ignore
		record = &agentmodel.AgentLiveness{InstanceUID: instanceUID}
	}

	record.DurableReportedAt = at
	s.records[instanceUID] = record

	return nil
}

func (s *benchLiveness) Get(_ context.Context, instanceUID uuid.UUID) (*agentmodel.AgentLiveness, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.records[instanceUID].Clone(), nil
}

func (s *benchLiveness) GetMany(
	context.Context,
	[]uuid.UUID,
) (map[uuid.UUID]*agentmodel.AgentLiveness, error) {
	return map[uuid.UUID]*agentmodel.AgentLiveness{}, nil
}

func (s *benchLiveness) Delete(context.Context, uuid.UUID) error { return nil }

func (s *benchLiveness) ListPendingWriteThrough(
	_ context.Context,
	cutoff time.Time,
	_ int,
) ([]*agentmodel.AgentLiveness, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pending := make([]*agentmodel.AgentLiveness, 0, len(s.records))

	for _, record := range s.records {
		if record.IsPendingWriteThroughSince(cutoff) {
			pending = append(pending, record.Clone())
		}
	}

	return pending, nil
}

// benchMetrics discards measurements.
type benchMetrics struct{}

func (benchMetrics) RecordHeartbeatAbsorbed()                        {}
func (benchMetrics) RecordWriteThrough(agentport.LivenessWriteShape) {}
func (benchMetrics) RecordFallback(string)                           {}
func (benchMetrics) RecordBreakerState(string)                       {}

// BenchmarkHeartbeatWriteCost measures the thing this whole mechanism is about: what
// a fleet's heartbeats cost the durable store.
//
// It is a counting benchmark, not a latency one, and it reports two numbers because
// the two write shapes are not interchangeable. A full document rewrite carries the
// whole agent and bumps its resource version; a liveness write $sets four fields. The
// point of the write-behind path is to move steady-state traffic from the first to
// the second, so a single "writes per heartbeat" number would hide the result.
func BenchmarkHeartbeatWriteCost(b *testing.B) {
	cases := []struct {
		name string
		// heartbeat is how often the agent reports.
		heartbeat time.Duration
		// throttle is the message-path write-through window. A nanosecond reproduces
		// the behaviour before any of this existed: every heartbeat rewrote the
		// agent document.
		throttle time.Duration
		// withFlush runs the write-behind flush alongside the message path.
		withFlush bool
	}{
		{name: "5s heartbeat/every heartbeat (baseline)", heartbeat: 5 * time.Second, throttle: time.Nanosecond},
		{name: "5s heartbeat/throttle only", heartbeat: 5 * time.Second},
		{name: "5s heartbeat/write-behind", heartbeat: 5 * time.Second, withFlush: true},
		{name: "30s heartbeat/every heartbeat (baseline)", heartbeat: 30 * time.Second, throttle: time.Nanosecond},
		{name: "30s heartbeat/throttle only", heartbeat: 30 * time.Second},
		{name: "30s heartbeat/write-behind", heartbeat: 30 * time.Second, withFlush: true},
	}

	for _, testCase := range cases {
		b.Run(testCase.name, func(b *testing.B) {
			persistence := runHeartbeats(b, testCase.heartbeat, testCase.throttle, testCase.withFlush, b.N)

			// Multiply by fleet size for the datastore write rate at this cadence.
			b.ReportMetric(float64(persistence.documentWrites)/float64(b.N), "docWrites/hb")
			b.ReportMetric(float64(persistence.livenessWrites)/float64(b.N), "liveWrites/hb")
		})
	}
}

// runHeartbeats drives heartbeats for one agent through the real throttle and flush
// logic, and returns what reached the durable store.
func runHeartbeats(
	b *testing.B,
	heartbeat, throttle time.Duration,
	withFlush bool,
	heartbeats int,
) *countingPersistence {
	b.Helper()

	if throttle <= 0 {
		throttle = MaxLivenessStalenessBudget()
	}

	persistence := &countingPersistence{}
	liveness := &benchLiveness{records: make(map[uuid.UUID]*agentmodel.AgentLiveness)}
	testClock := &benchClock{now: time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC)}

	service := NewAgentService(
		persistence, liveness, benchMetrics{}, slog.New(slog.DiscardHandler),
		AgentCacheConfig{Enabled: false, TTL: 0, MaxCapacity: 0},
		AgentLivenessConfig{PersistThrottle: throttle},
		"",
		testClock,
	)

	flusher := NewAgentLivenessFlusher(
		service, liveness, DefaultAgentLivenessFlushConfig(), slog.New(slog.DiscardHandler), testClock)

	agent := agentmodel.NewAgent(uuid.New())
	nextFlush := flusher.Interval()

	b.ResetTimer()

	for range heartbeats {
		testClock.now = testClock.now.Add(heartbeat)
		agent.Status.LastReportedAt = testClock.now

		if service.TouchAgentLiveness(b.Context(), agent, testClock.now) {
			err := service.SaveAgent(b.Context(), agent)
			if err != nil {
				b.Fatal(err)
			}
		}

		// Run whatever flush cycles fall inside the interval just advanced through.
		for withFlush && nextFlush <= heartbeat {
			nextFlush += flusher.Interval()

			_, err := flusher.Flush(b.Context())
			if err != nil {
				b.Fatal(err)
			}
		}

		nextFlush -= heartbeat
		if nextFlush <= 0 {
			nextFlush = flusher.Interval()
		}
	}

	b.StopTimer()

	return persistence
}
