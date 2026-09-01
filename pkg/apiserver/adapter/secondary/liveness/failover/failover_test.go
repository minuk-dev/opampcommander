package failover_test

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/liveness/failover"
	livenessinmemory "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/liveness/inmemory"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

var errPrimaryDown = errors.New("primary tier unavailable")

// movableClock is a clock.PassiveClock the test advances by hand.
type movableClock struct{ now time.Time }

func (c *movableClock) Now() time.Time                  { return c.now }
func (c *movableClock) Since(t time.Time) time.Duration { return c.now.Sub(t) }

// flakyPrimary is a liveness tier the test switches between working and failing.
type flakyPrimary struct {
	*livenessinmemory.Store

	down  atomic.Bool
	calls atomic.Int64
}

func newFlakyPrimary() *flakyPrimary {
	//exhaustruct:ignore
	return &flakyPrimary{
		Store: livenessinmemory.New(livenessinmemory.Config{TTL: time.Hour, GCInterval: time.Hour}, nil),
	}
}

func (p *flakyPrimary) Touch(
	ctx context.Context,
	liveness *agentmodel.AgentLiveness,
) (*agentmodel.AgentLiveness, error) {
	p.calls.Add(1)

	if p.down.Load() {
		return nil, errPrimaryDown
	}

	return p.Store.Touch(ctx, liveness) //nolint:wrapcheck // delegating fake
}

func (p *flakyPrimary) MarkPersisted(ctx context.Context, instanceUID uuid.UUID, at time.Time) error {
	p.calls.Add(1)

	if p.down.Load() {
		return errPrimaryDown
	}

	return p.Store.MarkPersisted(ctx, instanceUID, at) //nolint:wrapcheck // delegating fake
}

func (p *flakyPrimary) Get(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.AgentLiveness, error) {
	p.calls.Add(1)

	if p.down.Load() {
		return nil, errPrimaryDown
	}

	return p.Store.Get(ctx, instanceUID) //nolint:wrapcheck // delegating fake
}

func (p *flakyPrimary) GetMany(
	ctx context.Context,
	instanceUIDs []uuid.UUID,
) (map[uuid.UUID]*agentmodel.AgentLiveness, error) {
	p.calls.Add(1)

	if p.down.Load() {
		return nil, errPrimaryDown
	}

	return p.Store.GetMany(ctx, instanceUIDs) //nolint:wrapcheck // delegating fake
}

func (p *flakyPrimary) Delete(ctx context.Context, instanceUID uuid.UUID) error {
	p.calls.Add(1)

	if p.down.Load() {
		return errPrimaryDown
	}

	return p.Store.Delete(ctx, instanceUID) //nolint:wrapcheck // delegating fake
}

func (p *flakyPrimary) ListPendingWriteThrough(
	ctx context.Context,
	notPersistedSince time.Time,
	limit int,
) ([]*agentmodel.AgentLiveness, error) {
	p.calls.Add(1)

	if p.down.Load() {
		return nil, errPrimaryDown
	}

	return p.Store.ListPendingWriteThrough(ctx, notPersistedSince, limit) //nolint:wrapcheck // delegating fake
}

type fixture struct {
	primary  *flakyPrimary
	fallback *livenessinmemory.Store
	store    *failover.Store
	clock    *movableClock
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	testClock := &movableClock{now: time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC)}
	primary := newFlakyPrimary()
	fallback := livenessinmemory.New(
		livenessinmemory.Config{TTL: time.Hour, GCInterval: time.Hour}, testClock)

	store := failover.New(primary, fallback,
		failover.Config{FailureThreshold: 2, ProbeInterval: time.Minute},
		testClock, slog.Default())

	return &fixture{primary: primary, fallback: fallback, store: store, clock: testClock}
}

// mustTouch records an observation and fails the test if the store reports an error.
func mustTouch(t *testing.T, store *failover.Store, observation *agentmodel.AgentLiveness) {
	t.Helper()

	_, err := store.Touch(t.Context(), observation)
	require.NoError(t, err)
}

func liveness(instanceUID uuid.UUID, at time.Time) *agentmodel.AgentLiveness {
	return &agentmodel.AgentLiveness{
		InstanceUID:       instanceUID,
		Connected:         true,
		ConnectionType:    agentmodel.ConnectionTypeWebSocket,
		SequenceNum:       1,
		LastReportedAt:    at,
		LastReportedTo:    "server-a",
		DurableReportedAt: time.Time{},
	}
}

func TestTouchKeepsTheFallbackWarm(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	instanceUID := uuid.New()

	mustTouch(t, fixture.store, liveness(instanceUID, fixture.clock.now))

	// The node-local tier must already hold the agent while the primary is healthy,
	// or a trip would fall back onto an empty store and every agent would look new.
	got, err := fixture.fallback.Get(t.Context(), instanceUID)
	require.NoError(t, err)
	assert.NotNil(t, got)
}

func TestTripsAfterTheFailureThresholdAndServesFromTheFallback(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	instanceUID := uuid.New()

	mustTouch(t, fixture.store, liveness(instanceUID, fixture.clock.now))
	fixture.primary.down.Store(true)

	// Below the threshold the breaker stays closed and keeps trying the primary.
	_, err := fixture.store.Get(t.Context(), instanceUID)
	require.NoError(t, err)
	assert.Equal(t, failover.StateClosed, fixture.store.State())

	_, err = fixture.store.Get(t.Context(), instanceUID)
	require.NoError(t, err)
	assert.Equal(t, failover.StateOpen, fixture.store.State())

	// Open, the answer comes from the fallback without touching the primary at all.
	before := fixture.primary.calls.Load()

	got, err := fixture.store.Get(t.Context(), instanceUID)
	require.NoError(t, err)
	require.NotNil(t, got, "the warm fallback still knows this agent")
	assert.Equal(t, before, fixture.primary.calls.Load(),
		"an open breaker must not keep paying for calls to a dead primary")
}

func TestNoErrorEverReachesTheCaller(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	fixture.primary.down.Store(true)

	instanceUID := uuid.New()

	mustTouch(t, fixture.store, liveness(instanceUID, fixture.clock.now))

	_, err := fixture.store.Get(t.Context(), instanceUID)
	require.NoError(t, err)

	_, err = fixture.store.GetMany(t.Context(), []uuid.UUID{instanceUID})
	require.NoError(t, err)

	_, err = fixture.store.ListPendingWriteThrough(t.Context(), fixture.clock.now, 0)
	require.NoError(t, err)

	require.NoError(t, fixture.store.Delete(t.Context(), instanceUID))
}

func TestTouchSurvivesAPrimaryOutage(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	fixture.primary.down.Store(true)

	instanceUID := uuid.New()

	mustTouch(t, fixture.store, liveness(instanceUID, fixture.clock.now))

	// The observation must survive in the fallback: losing it would lose the
	// heartbeat entirely, which is worse than an extra database write.
	got, err := fixture.store.Get(t.Context(), instanceUID)
	require.NoError(t, err)
	assert.NotNil(t, got)
}

func TestRecoversWithoutARestart(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	instanceUID := uuid.New()

	fixture.primary.down.Store(true)

	for range 2 {
		_, err := fixture.store.Get(t.Context(), instanceUID)
		require.NoError(t, err)
	}

	require.Equal(t, failover.StateOpen, fixture.store.State())

	// Before the probe interval the breaker stays shut against the primary.
	fixture.clock.now = fixture.clock.now.Add(30 * time.Second)
	_, err := fixture.store.Get(t.Context(), instanceUID)
	require.NoError(t, err)
	assert.Equal(t, failover.StateOpen, fixture.store.State())

	// After it, one call is let through — and a healthy primary closes the breaker.
	fixture.primary.down.Store(false)
	fixture.clock.now = fixture.clock.now.Add(time.Minute)

	_, err = fixture.store.Get(t.Context(), instanceUID)
	require.NoError(t, err)
	assert.Equal(t, failover.StateClosed, fixture.store.State())
}

func TestAFailedProbeReopensImmediately(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	instanceUID := uuid.New()

	fixture.primary.down.Store(true)

	for range 2 {
		_, err := fixture.store.Get(t.Context(), instanceUID)
		require.NoError(t, err)
	}

	require.Equal(t, failover.StateOpen, fixture.store.State())

	// The probe finds the primary still down: back to open on that one failure,
	// without waiting out the threshold again.
	fixture.clock.now = fixture.clock.now.Add(time.Minute)

	_, err := fixture.store.Get(t.Context(), instanceUID)
	require.NoError(t, err)
	assert.Equal(t, failover.StateOpen, fixture.store.State())
}

func TestForcesAFlushExactlyOncePerOutage(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)

	var flushes atomic.Int64

	done := make(chan struct{}, 1)

	fixture.store.OnTrip(func(context.Context) {
		flushes.Add(1)

		select {
		case done <- struct{}{}:
		default:
		}
	})

	instanceUID := uuid.New()

	fixture.primary.down.Store(true)

	for range 5 {
		_, err := fixture.store.Get(t.Context(), instanceUID)
		require.NoError(t, err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the breaker tripped without forcing a flush")
	}

	// Five failed calls, one outage: the flush must not fire per failure.
	assert.Equal(t, int64(1), flushes.Load())
}

func TestDeleteClearsBothTiers(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	instanceUID := uuid.New()

	mustTouch(t, fixture.store, liveness(instanceUID, fixture.clock.now))
	require.NoError(t, fixture.store.Delete(t.Context(), instanceUID))

	// A record left in the primary would come back the moment the breaker closes.
	fromPrimary, err := fixture.primary.Store.Get(t.Context(), instanceUID)
	require.NoError(t, err)
	assert.Nil(t, fromPrimary)

	fromFallback, err := fixture.fallback.Get(t.Context(), instanceUID)
	require.NoError(t, err)
	assert.Nil(t, fromFallback)
}

func TestStateString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "closed", failover.StateClosed.String())
	assert.Equal(t, "open", failover.StateOpen.String())
	assert.Equal(t, "half-open", failover.StateHalfOpen.String())
}

func TestTouchReturnsTheAnswringTiersAnchor(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	instanceUID := uuid.New()
	anchored := fixture.clock.now.Add(-time.Minute)

	mustTouch(t, fixture.store, liveness(instanceUID, fixture.clock.now))
	require.NoError(t, fixture.store.MarkPersisted(t.Context(), instanceUID, anchored))

	// While the primary answers, the anchor comes from it.
	stored, err := fixture.store.Touch(t.Context(), liveness(instanceUID, fixture.clock.now))
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, anchored, stored.DurableReportedAt)

	// With the primary gone the fallback answers, and it was anchored too — so the
	// throttle window survives the outage instead of restarting.
	fixture.primary.down.Store(true)

	for range 2 {
		_, err := fixture.store.Get(t.Context(), instanceUID)
		require.NoError(t, err)
	}

	require.Equal(t, failover.StateOpen, fixture.store.State())

	stored, err = fixture.store.Touch(t.Context(), liveness(instanceUID, fixture.clock.now))
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, anchored, stored.DurableReportedAt)
}

func TestWaitCollectsForcedFlushes(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)

	started := make(chan struct{})
	release := make(chan struct{})

	var finished atomic.Bool

	fixture.store.OnTrip(func(context.Context) {
		close(started)
		<-release

		finished.Store(true)
	})

	fixture.primary.down.Store(true)

	for range 2 {
		_, err := fixture.store.Get(t.Context(), uuid.New())
		require.NoError(t, err)
	}

	<-started
	close(release)

	// A forced flush outlives the call that tripped the breaker, so shutdown has to
	// collect it rather than exit mid-write.
	fixture.store.Wait()
	assert.True(t, finished.Load())
}
