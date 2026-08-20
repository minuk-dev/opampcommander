package agentservice_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// flushFixture wires an agent service over mock persistence and a fake fast tier,
// plus a flusher that claims anything not written through within staleAfter.
type flushFixture struct {
	persistence *MockAgentPersistencePort
	liveness    *fakeLivenessPort
	service     *agentservice.AgentService
	flusher     *agentservice.AgentLivenessFlusher
}

func newFlushFixture(t *testing.T, staleAfter time.Duration) *flushFixture {
	t.Helper()

	persistence := new(MockAgentPersistencePort)
	liveness := newFakeLivenessPort()

	service := agentservice.NewAgentService(
		persistence, liveness, slog.Default(),
		agentservice.AgentCacheConfig{Enabled: false, TTL: 0, MaxCapacity: 0},
		// A long throttle so the per-message path never writes on its own and the
		// flusher is unambiguously the thing under test.
		agentservice.AgentLivenessConfig{PersistThrottle: time.Hour},
		"",
		nil,
	)

	flusher := agentservice.NewAgentLivenessFlusher(
		service, liveness,
		agentservice.AgentLivenessFlushConfig{
			Interval:   agentservice.DefaultLivenessFlushInterval,
			BatchSize:  agentservice.DefaultLivenessFlushBatchSize,
			StaleAfter: staleAfter,
		},
		slog.Default(),
		nil,
	)

	return &flushFixture{
		persistence: persistence,
		liveness:    liveness,
		service:     service,
		flusher:     flusher,
	}
}

func TestFlush_WritesTheObservationTheThrottleWithheld(t *testing.T) {
	t.Parallel()

	instanceUID := uuid.New()
	observed := time.Now()

	fixture := newFlushFixture(t, time.Nanosecond)
	fixture.persistence.On("UpdateAgentLiveness", mock.Anything, mock.Anything).Return(nil)

	mustTouchLiveness(t, fixture.liveness, newLivenessRecord(instanceUID, observed, 42))

	written, err := fixture.flusher.Flush(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, written)

	written2, ok := fixture.persistence.Calls[0].Arguments.Get(1).(*agentmodel.AgentLiveness)
	require.True(t, ok)
	assert.Equal(t, observed, written2.LastReportedAt)
	assert.Equal(t, uint64(42), written2.SequenceNum)
}

func TestFlush_NeverRewritesTheWholeDocument(t *testing.T) {
	t.Parallel()

	fixture := newFlushFixture(t, time.Nanosecond)
	fixture.persistence.On("UpdateAgentLiveness", mock.Anything, mock.Anything).Return(nil)

	mustTouchLiveness(t, fixture.liveness, newLivenessRecord(uuid.New(), time.Now(), 1))

	_, err := fixture.flusher.Flush(t.Context())
	require.NoError(t, err)

	// Rewriting the document on this cadence is the cost the fast tier exists to
	// remove, and it would bump the resource version on every heartbeat — so routine
	// liveness would keep invalidating concurrent API writes.
	fixture.persistence.AssertNotCalled(t, "PutAgent", mock.Anything, mock.Anything)
	fixture.persistence.AssertNotCalled(t, "GetAgent", mock.Anything, mock.Anything)
}

func TestFlush_AnchorsTheRecordSoTheNextCycleIsANoop(t *testing.T) {
	t.Parallel()

	instanceUID := uuid.New()

	fixture := newFlushFixture(t, time.Nanosecond)
	fixture.persistence.On("UpdateAgentLiveness", mock.Anything, mock.Anything).Return(nil)

	mustTouchLiveness(t, fixture.liveness, newLivenessRecord(instanceUID, time.Now(), 1))

	written, err := fixture.flusher.Flush(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, written)

	written, err = fixture.flusher.Flush(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 0, written, "a flushed observation must not be written again")
}

func TestFlush_LeavesAFreshDocumentAlone(t *testing.T) {
	t.Parallel()

	instanceUID := uuid.New()
	observed := time.Now()

	// An hour's tolerance: an observation written moments ago is nowhere near
	// overdue, so the flush has nothing to do.
	fixture := newFlushFixture(t, time.Hour)

	mustTouchLiveness(t, fixture.liveness, newLivenessRecord(instanceUID, observed, 1))
	require.NoError(t, fixture.liveness.MarkPersisted(t.Context(), instanceUID, observed))
	mustTouchLiveness(t, fixture.liveness, newLivenessRecord(instanceUID, observed.Add(time.Second), 2))

	written, err := fixture.flusher.Flush(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 0, written)
	fixture.persistence.AssertNotCalled(t, "UpdateAgentLiveness", mock.Anything, mock.Anything)
}

func TestFlush_ForgetsAnAgentThatNoLongerExists(t *testing.T) {
	t.Parallel()

	instanceUID := uuid.New()

	fixture := newFlushFixture(t, time.Nanosecond)
	fixture.persistence.On("UpdateAgentLiveness", mock.Anything, mock.Anything).
		Return(model.ErrResourceNotExist)

	mustTouchLiveness(t, fixture.liveness, newLivenessRecord(instanceUID, time.Now(), 1))

	written, err := fixture.flusher.Flush(t.Context())
	require.NoError(t, err, "a deleted agent is not a flush failure")
	assert.Equal(t, 0, written)

	// The record must be dropped, or the flush would chase the deleted agent every
	// cycle for as long as the record lives.
	got, err := fixture.liveness.Get(t.Context(), instanceUID)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestFlush_ReportsAFastTierFailure(t *testing.T) {
	t.Parallel()

	persistence := new(MockAgentPersistencePort)
	service := agentservice.NewAgentService(
		persistence, brokenLivenessPort{}, slog.Default(),
		agentservice.AgentCacheConfig{Enabled: false, TTL: 0, MaxCapacity: 0},
		agentservice.DefaultAgentLivenessConfig(), "", nil,
	)
	flusher := agentservice.NewAgentLivenessFlusher(
		service, brokenLivenessPort{}, agentservice.DefaultAgentLivenessFlushConfig(), slog.Default(), nil,
	)

	written, err := flusher.Flush(t.Context())
	require.Error(t, err, "the flusher owns the tier's health, so here the failure is worth reporting")
	assert.Equal(t, 0, written)
}

func TestFlush_RespectsTheBatchBound(t *testing.T) {
	t.Parallel()

	fixture := newFlushFixture(t, time.Nanosecond)
	fixture.persistence.On("UpdateAgentLiveness", mock.Anything, mock.Anything).Return(nil)

	bounded := agentservice.NewAgentLivenessFlusher(
		fixture.service, fixture.liveness,
		agentservice.AgentLivenessFlushConfig{Interval: time.Second, BatchSize: 2, StaleAfter: time.Nanosecond},
		slog.Default(),
		nil,
	)

	for range 5 {
		mustTouchLiveness(t, fixture.liveness, newLivenessRecord(uuid.New(), time.Now(), 1))
	}

	written, err := bounded.Flush(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 2, written, "a cycle must not exceed its batch bound")
}

func TestNewAgentLivenessFlusher_AcceptsTheDefaults(t *testing.T) {
	t.Parallel()

	// The defaults must survive construction untouched — defaults that trip their own
	// guard would warn on every startup and quietly run at a different cadence than
	// the documented one.
	fixture := newFlushFixture(t, time.Hour)
	flusher := agentservice.NewAgentLivenessFlusher(
		fixture.service, fixture.liveness, agentservice.DefaultAgentLivenessFlushConfig(), slog.Default(), nil,
	)

	assert.Equal(t, agentservice.DefaultLivenessFlushInterval, flusher.Interval())
	assert.Equal(t, agentservice.DefaultLivenessFlushStaleAfter, flusher.StaleAfter())
}

func TestNewAgentLivenessFlusher_ClampsTheCadenceAsAPair(t *testing.T) {
	t.Parallel()

	fixture := newFlushFixture(t, time.Hour)
	flusher := agentservice.NewAgentLivenessFlusher(
		fixture.service, fixture.liveness,
		agentservice.AgentLivenessFlushConfig{Interval: 5 * time.Minute, BatchSize: 0, StaleAfter: 5 * time.Minute},
		slog.Default(),
		nil,
	)

	// It is the sum that bounds how stale a stored document gets, so both halves
	// have to come down together — clamping only the interval would let the pair
	// overshoot the window while looking configured correctly.
	assert.LessOrEqual(t, flusher.Interval()+flusher.StaleAfter(),
		agentservice.MaxLivenessStalenessBudget())
	assert.Positive(t, flusher.Interval())
	assert.Positive(t, flusher.StaleAfter())
}

func TestMaxLivenessStalenessBudgetLeavesHeadroom(t *testing.T) {
	t.Parallel()

	assert.Less(t, agentservice.MaxLivenessStalenessBudget(), agentmodel.DefaultConnectionStaleness,
		"the budget must leave room between the last write-through and the staleness window")
	assert.LessOrEqual(t,
		agentservice.DefaultLivenessFlushInterval+agentservice.DefaultLivenessFlushStaleAfter,
		agentservice.MaxLivenessStalenessBudget())
}
