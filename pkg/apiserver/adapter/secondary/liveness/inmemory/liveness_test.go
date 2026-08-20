package inmemory_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/liveness/inmemory"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

// movableClock is a clock.PassiveClock whose current time the test advances.
type movableClock struct{ now time.Time }

func (c *movableClock) Now() time.Time                  { return c.now }
func (c *movableClock) Since(t time.Time) time.Duration { return c.now.Sub(t) }

func newLiveness(instanceUID uuid.UUID, at time.Time) *agentmodel.AgentLiveness {
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

// mustTouch records an observation and fails the test if the store rejects it.
func mustTouch(t *testing.T, store *inmemory.Store, liveness *agentmodel.AgentLiveness) {
	t.Helper()

	_, err := store.Touch(t.Context(), liveness)
	require.NoError(t, err)
}

func TestStoreTouchAndGet(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC)
	testClock := &movableClock{now: now}
	store := inmemory.New(inmemory.Config{TTL: time.Minute, GCInterval: time.Minute}, testClock)

	instanceUID := uuid.New()

	got, err := store.Get(t.Context(), instanceUID)
	require.NoError(t, err)
	assert.Nil(t, got, "an agent that was never touched has no record")

	mustTouch(t, store, newLiveness(instanceUID, now))

	got, err = store.Get(t.Context(), instanceUID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, uint64(1), got.SequenceNum)
	assert.Equal(t, now, got.LastReportedAt)
}

func TestStoreTouchStoresACopy(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC)
	store := inmemory.New(inmemory.Config{}, &movableClock{now: now})

	instanceUID := uuid.New()
	liveness := newLiveness(instanceUID, now)
	mustTouch(t, store, liveness)

	// Mutating the caller's copy after the write must not reach the store, and the
	// value handed back must not be the stored one either.
	liveness.SequenceNum = 100

	got, err := store.Get(t.Context(), instanceUID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, uint64(1), got.SequenceNum)

	got.SequenceNum = 200

	again, err := store.Get(t.Context(), instanceUID)
	require.NoError(t, err)
	require.NotNil(t, again)
	assert.Equal(t, uint64(1), again.SequenceNum)
}

func TestStoreTouchNil(t *testing.T) {
	t.Parallel()

	store := inmemory.New(inmemory.Config{}, nil)

	mustTouch(t, store, nil)
	assert.Equal(t, 0, store.Len())
}

func TestStoreGetMany(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC)
	store := inmemory.New(inmemory.Config{}, &movableClock{now: now})

	known := uuid.New()
	unknown := uuid.New()

	mustTouch(t, store, newLiveness(known, now))

	got, err := store.GetMany(t.Context(), []uuid.UUID{known, unknown})
	require.NoError(t, err)
	require.Len(t, got, 1, "UIDs without a record are omitted, not reported")
	assert.Contains(t, got, known)
	assert.NotContains(t, got, unknown)

	empty, err := store.GetMany(t.Context(), nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestStoreDelete(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC)
	store := inmemory.New(inmemory.Config{}, &movableClock{now: now})

	instanceUID := uuid.New()
	mustTouch(t, store, newLiveness(instanceUID, now))
	require.NoError(t, store.Delete(t.Context(), instanceUID))

	got, err := store.Get(t.Context(), instanceUID)
	require.NoError(t, err)
	assert.Nil(t, got)

	// Deleting an absent record succeeds.
	require.NoError(t, store.Delete(t.Context(), uuid.New()))
}

func TestStoreExpiredRecordReadsAsAbsent(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC)
	testClock := &movableClock{now: now}
	store := inmemory.New(inmemory.Config{TTL: time.Minute, GCInterval: time.Minute}, testClock)

	instanceUID := uuid.New()
	mustTouch(t, store, newLiveness(instanceUID, now))

	testClock.now = now.Add(time.Minute)

	got, err := store.Get(t.Context(), instanceUID)
	require.NoError(t, err)
	assert.Nil(t, got, "an expired record must read as absent even before GC sweeps it")

	many, err := store.GetMany(t.Context(), []uuid.UUID{instanceUID})
	require.NoError(t, err)
	assert.Empty(t, many)
}

func TestStoreGC(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC)
	testClock := &movableClock{now: now}
	store := inmemory.New(inmemory.Config{TTL: time.Minute, GCInterval: time.Minute}, testClock)

	stale := uuid.New()
	live := uuid.New()

	mustTouch(t, store, newLiveness(stale, now))

	testClock.now = now.Add(time.Minute)
	mustTouch(t, store, newLiveness(live, testClock.now))

	assert.Equal(t, 2, store.Len())
	assert.Equal(t, 1, store.GC())
	assert.Equal(t, 1, store.Len())

	got, err := store.Get(t.Context(), live)
	require.NoError(t, err)
	assert.NotNil(t, got)
}

func TestStoreDefaults(t *testing.T) {
	t.Parallel()

	store := inmemory.New(inmemory.Config{TTL: -1, GCInterval: 0}, nil)

	assert.Equal(t, inmemory.DefaultGCInterval, store.GCInterval())

	// The default TTL must outlive the staleness window, or GC could drop a record
	// for an agent still reported as connected.
	assert.Greater(t, inmemory.DefaultTTL, agentmodel.DefaultConnectionStaleness)
}

func TestStoreTouchReturnsThePreservedAnchor(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC)
	store := inmemory.New(inmemory.Config{TTL: time.Hour, GCInterval: time.Hour}, &movableClock{now: now})

	instanceUID := uuid.New()
	persistedAt := now.Add(-30 * time.Second)

	mustTouch(t, store, newLiveness(instanceUID, now.Add(-time.Minute)))
	require.NoError(t, store.MarkPersisted(t.Context(), instanceUID, persistedAt))

	// A later observation must come back carrying the anchor MarkPersisted set —
	// that is what lets the caller decide on a durable write without a second read.
	stored, err := store.Touch(t.Context(), newLiveness(instanceUID, now))
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, now, stored.LastReportedAt)
	assert.Equal(t, persistedAt, stored.DurableReportedAt)
}

func TestStoreTouchIgnoresACallerSuppliedAnchor(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC)
	store := inmemory.New(inmemory.Config{TTL: time.Hour, GCInterval: time.Hour}, &movableClock{now: now})

	instanceUID := uuid.New()
	mustTouch(t, store, newLiveness(instanceUID, now.Add(-time.Minute)))
	require.NoError(t, store.MarkPersisted(t.Context(), instanceUID, now.Add(-time.Minute)))

	// Only MarkPersisted moves the anchor. An observation claiming otherwise must
	// not be able to reset the throttle window.
	observation := newLiveness(instanceUID, now)
	observation.DurableReportedAt = now

	stored, err := store.Touch(t.Context(), observation)
	require.NoError(t, err)
	assert.Equal(t, now.Add(-time.Minute), stored.DurableReportedAt)
}

func TestStoreMarkPersistedCreatesAnAbsentRecord(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC)
	store := inmemory.New(inmemory.Config{TTL: time.Hour, GCInterval: time.Hour}, &movableClock{now: now})

	instanceUID := uuid.New()
	require.NoError(t, store.MarkPersisted(t.Context(), instanceUID, now))

	got, err := store.Get(t.Context(), instanceUID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, now, got.DurableReportedAt)
	assert.True(t, got.LastReportedAt.IsZero(),
		"MarkPersisted records only the anchor; it must not invent an observation")
}

// persistedAt records an observation and then anchors its write-through, which is
// the only way an anchor is set — Touch never takes one from the caller.
func persistedAt(t *testing.T, store *inmemory.Store, instanceUID uuid.UUID, observed, persisted time.Time) {
	t.Helper()

	mustTouch(t, store, newLiveness(instanceUID, observed))
	require.NoError(t, store.MarkPersisted(t.Context(), instanceUID, persisted))
	mustTouch(t, store, newLiveness(instanceUID, observed))
}

func TestStoreListPendingWriteThrough(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC)
	testClock := &movableClock{now: now}
	store := inmemory.New(inmemory.Config{TTL: time.Hour, GCInterval: time.Hour}, testClock)

	neverPersisted := uuid.New()
	longOverdue := uuid.New()
	justWritten := uuid.New()
	upToDate := uuid.New()

	mustTouch(t, store, newLiveness(neverPersisted, now))
	persistedAt(t, store, longOverdue, now, now.Add(-10*time.Minute))
	persistedAt(t, store, justWritten, now, now.Add(-time.Second))

	// Anchored after its observation: nothing left to write through.
	mustTouch(t, store, newLiveness(upToDate, now))
	require.NoError(t, store.MarkPersisted(t.Context(), upToDate, now))

	// Cutoff at 1 minute ago: the never-persisted and long-overdue records qualify,
	// the one written a second ago does not, and the up-to-date one is not pending
	// at all.
	pending, err := store.ListPendingWriteThrough(t.Context(), now.Add(-time.Minute), 0)
	require.NoError(t, err)
	require.Len(t, pending, 2)
	assert.Equal(t, neverPersisted, pending[0].InstanceUID, "never-written-through records come first")
	assert.Equal(t, longOverdue, pending[1].InstanceUID)
}

func TestStoreListPendingWriteThroughRespectsTheLimit(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC)
	store := inmemory.New(inmemory.Config{TTL: time.Hour, GCInterval: time.Hour}, &movableClock{now: now})

	for range 5 {
		mustTouch(t, store, newLiveness(uuid.New(), now))
	}

	pending, err := store.ListPendingWriteThrough(t.Context(), now, 3)
	require.NoError(t, err)
	assert.Len(t, pending, 3)
}

func TestStoreListPendingWriteThroughSkipsExpiredRecords(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC)
	testClock := &movableClock{now: now}
	store := inmemory.New(inmemory.Config{TTL: time.Minute, GCInterval: time.Hour}, testClock)

	mustTouch(t, store, newLiveness(uuid.New(), now))

	testClock.now = now.Add(time.Minute)

	pending, err := store.ListPendingWriteThrough(t.Context(), testClock.now, 0)
	require.NoError(t, err)
	assert.Empty(t, pending, "an expired record is gone, not pending")
}
