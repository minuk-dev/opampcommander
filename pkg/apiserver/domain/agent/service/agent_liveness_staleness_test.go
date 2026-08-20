package agentservice_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
)

// storedDocument stands in for the durable store, recording only what matters here:
// the observation timestamp the document holds, and how it got there.
type storedDocument struct {
	reportedAt   time.Time
	fullWrites   int
	narrowWrites int
}

// stalenessPersistence is a persistence port that tracks the stored document.
type stalenessPersistence struct {
	*MockAgentPersistencePort

	doc *storedDocument
}

func (p *stalenessPersistence) PutAgent(_ context.Context, agent *agentmodel.Agent) error {
	p.doc.reportedAt = agent.Status.LastReportedAt
	p.doc.fullWrites++

	return nil
}

func (p *stalenessPersistence) UpdateAgentLiveness(
	_ context.Context,
	liveness *agentmodel.AgentLiveness,
) error {
	p.doc.reportedAt = liveness.LastReportedAt
	p.doc.narrowWrites++

	return nil
}

// TestStoredDocumentStaysInsideTheStalenessWindow is the invariant the whole
// write-behind design has to hold, checked across the range of heartbeat intervals a
// fleet actually uses.
//
// It matters because the datastore evaluates "connected" against the stored
// timestamp, in two places the read-side liveness overlay cannot reach: the
// connected-agent list filter and the agent-group connected counts. Let the stored
// document drift past DefaultConnectionStaleness and both start reporting live
// agents as disconnected while their WebSockets are fine.
//
// Bounding the flush interval alone is not enough — the document goes stale by up to
// Interval + StaleAfter, and the message-path throttle is a third way to overshoot.
func TestStoredDocumentStaysInsideTheStalenessWindow(t *testing.T) {
	t.Parallel()

	for heartbeatSeconds := 5; heartbeatSeconds <= 90; heartbeatSeconds += 5 {
		heartbeat := time.Duration(heartbeatSeconds) * time.Second

		t.Run(heartbeat.String(), func(t *testing.T) {
			t.Parallel()

			worst, doc := simulateFleet(t, heartbeat)

			assert.Less(t, worst, agentmodel.DefaultConnectionStaleness,
				"a live agent's stored document must never age past the staleness window "+
					"(full writes %d, narrow writes %d)", doc.fullWrites, doc.narrowWrites)
		})
	}
}

// TestWriteBehindPrefersNarrowWrites pins the payoff: at a typical heartbeat the
// durable store sees liveness-only updates, not full document rewrites.
func TestWriteBehindPrefersNarrowWrites(t *testing.T) {
	t.Parallel()

	_, doc := simulateFleet(t, 30*time.Second)

	assert.Equal(t, 1, doc.fullWrites, "only the agent's first message should rewrite the document")
	assert.Greater(t, doc.narrowWrites, 10, "steady-state liveness should ride the narrow write path")
}

// simulateFleet drives one agent through 30 minutes of heartbeats against the real
// throttle and flush logic, and reports how stale the stored document ever got.
//
// Time is advanced by hand rather than slept, so a half-hour of fleet behaviour costs
// milliseconds.
func simulateFleet(t *testing.T, heartbeat time.Duration) (time.Duration, *storedDocument) {
	t.Helper()

	doc := &storedDocument{}
	mockPersistence := new(MockAgentPersistencePort)
	mockPersistence.On("GetAgent", mock.Anything, mock.Anything).Return(nil, nil).Maybe()

	persistence := &stalenessPersistence{MockAgentPersistencePort: mockPersistence, doc: doc}
	liveness := newFakeLivenessPort()
	testClock := &fixedClock{}

	service := agentservice.NewAgentService(
		persistence, liveness, newFakeLivenessMetrics(), slog.New(slog.DiscardHandler),
		agentservice.AgentCacheConfig{Enabled: false, TTL: 0, MaxCapacity: 0},
		// Deliberately oversized: the service must clamp it into the budget rather
		// than honour a throttle that would push the document past the window.
		agentservice.AgentLivenessConfig{PersistThrottle: 10 * time.Minute},
		"",
		testClock,
	)

	flusher := agentservice.NewAgentLivenessFlusher(
		service, liveness, agentservice.DefaultAgentLivenessFlushConfig(),
		slog.New(slog.DiscardHandler), testClock)

	agent := agentmodel.NewAgent(uuid.New())
	worst := time.Duration(0)
	start := time.Unix(0, 0)
	// Offset the first flush so ticks and heartbeats do not stay in phase.
	nextFlush := time.Second

	for elapsed := time.Second; elapsed <= 30*time.Minute; elapsed += time.Second {
		testClock.now = start.Add(elapsed)

		if elapsed%heartbeat == 0 {
			agent.Status.LastReportedAt = testClock.now
			if service.TouchAgentLiveness(t.Context(), agent, testClock.now) {
				require.NoError(t, service.SaveAgent(t.Context(), agent))
			}
		}

		if elapsed >= nextFlush {
			nextFlush += flusher.Interval()
			_, err := flusher.Flush(t.Context())
			require.NoError(t, err)
		}

		// Measure only once the agent is established and genuinely live.
		if !doc.reportedAt.IsZero() && elapsed > 3*time.Minute {
			if age := testClock.now.Sub(doc.reportedAt); age > worst {
				worst = age
			}
		}
	}

	return worst, doc
}
