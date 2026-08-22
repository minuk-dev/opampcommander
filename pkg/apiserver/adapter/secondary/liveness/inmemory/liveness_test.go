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
		InstanceUID:     instanceUID,
		Connected:       true,
		ConnectionType:  agentmodel.ConnectionTypeWebSocket,
		SequenceNum:     1,
		LastReportedAt:  at,
		LastReportedTo:  "server-a",
		LastPersistedAt: time.Time{},
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
	assert.Equal(t, persistedAt, stored.LastPersistedAt)
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
	observation.LastPersistedAt = now

	stored, err := store.Touch(t.Context(), observation)
	require.NoError(t, err)
	assert.Equal(t, now.Add(-time.Minute), stored.LastPersistedAt)
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
	assert.Equal(t, now, got.LastPersistedAt)
	assert.True(t, got.LastReportedAt.IsZero(),
		"MarkPersisted records only the anchor; it must not invent an observation")
}
