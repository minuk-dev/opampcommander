package redis_test

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	livenessredis "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/liveness/redis"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

const (
	testTTL            = 2 * time.Minute
	testCommandTimeout = 2 * time.Second
	serverA            = "server-a"
)

// One Redis container serves the whole package; see newStore.
//
//nolint:gochecknoglobals // package-scoped test fixture, started once
var (
	sharedRedisOnce     sync.Once
	sharedRedisEndpoint string
)

// newStore returns a store over a Redis container shared by the whole package.
//
// One container serves every test — starting one per test overwhelms the Docker
// host — and isolation comes from a per-test key prefix instead, which also
// exercises the prefix two deployments sharing a Redis would rely on.
func newStore(t *testing.T) *livenessredis.Store {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping redis container test in short mode")
	}

	sharedRedisOnce.Do(func() {
		sharedRedisEndpoint = testutil.NewBase(t).StartRedis().Endpoint
	})

	require.NotEmpty(t, sharedRedisEndpoint, "shared redis container failed to start")

	//exhaustruct:ignore
	store, err := livenessredis.New(livenessredis.Config{
		Endpoints:      []string{sharedRedisEndpoint},
		DialTimeout:    testCommandTimeout,
		CommandTimeout: testCommandTimeout,
		TTL:            testTTL,
		KeyPrefix:      "test:" + t.Name() + ":",
	})
	require.NoError(t, err)

	t.Cleanup(func() { _ = store.Close() })

	return store
}

func observationAt(instanceUID uuid.UUID, reportedAt time.Time) *agentmodel.AgentLiveness {
	return &agentmodel.AgentLiveness{
		InstanceUID:       instanceUID,
		Connected:         true,
		ConnectionType:    agentmodel.ConnectionTypeWebSocket,
		SequenceNum:       7,
		LastReportedAt:    reportedAt,
		LastReportedTo:    serverA,
		DurableReportedAt: time.Time{},
	}
}

// mustTouch records an observation and fails the test if the store rejects it.
func mustTouch(t *testing.T, store *livenessredis.Store, liveness *agentmodel.AgentLiveness) {
	t.Helper()

	_, err := store.Touch(t.Context(), liveness)
	require.NoError(t, err)
}

// persistedAt records an observation and anchors it, which is the only way an anchor
// is set — Touch never takes one from the caller.
func persistedAt(
	t *testing.T,
	store *livenessredis.Store,
	instanceUID uuid.UUID,
	observed, persisted time.Time,
) {
	t.Helper()

	mustTouch(t, store, observationAt(instanceUID, observed))
	require.NoError(t, store.MarkPersisted(t.Context(), instanceUID, persisted))
	mustTouch(t, store, observationAt(instanceUID, observed))
}

func TestStoreRoundTrip(t *testing.T) {
	t.Parallel()

	store := newStore(t)
	instanceUID := uuid.New()
	reportedAt := time.Now().Truncate(time.Nanosecond)

	mustTouch(t, store, observationAt(instanceUID, reportedAt))

	got, err := store.Get(t.Context(), instanceUID)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, instanceUID, got.InstanceUID)
	assert.True(t, got.Connected)
	assert.Equal(t, agentmodel.ConnectionTypeWebSocket, got.ConnectionType)
	assert.Equal(t, uint64(7), got.SequenceNum)
	assert.Equal(t, reportedAt.UnixNano(), got.LastReportedAt.UnixNano())
	assert.Equal(t, serverA, got.LastReportedTo)
	assert.True(t, got.DurableReportedAt.IsZero(), "a never-written-through record must round-trip as zero")
}

func TestStoreGetIsNotAnErrorWhenAbsent(t *testing.T) {
	t.Parallel()

	store := newStore(t)

	got, err := store.Get(t.Context(), uuid.New())
	require.NoError(t, err, "an absent record is a normal outcome, not an error")
	assert.Nil(t, got)
}

func TestStoreGetMany(t *testing.T) {
	t.Parallel()

	store := newStore(t)
	known := uuid.New()
	unknown := uuid.New()
	now := time.Now()

	mustTouch(t, store, observationAt(known, now))

	got, err := store.GetMany(t.Context(), []uuid.UUID{known, unknown})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Contains(t, got, known)

	empty, err := store.GetMany(t.Context(), nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestStoreDelete(t *testing.T) {
	t.Parallel()

	store := newStore(t)
	instanceUID := uuid.New()
	now := time.Now()

	mustTouch(t, store, observationAt(instanceUID, now))
	require.NoError(t, store.Delete(t.Context(), instanceUID))

	got, err := store.Get(t.Context(), instanceUID)
	require.NoError(t, err)
	assert.Nil(t, got)

	// The pending index must be cleared too, or the flush would keep chasing a
	// record that no longer exists.
	pending, err := store.ListPendingWriteThrough(t.Context(), time.Now(), 0)
	require.NoError(t, err)
	assert.Empty(t, pending)

	require.NoError(t, store.Delete(t.Context(), uuid.New()), "deleting an absent record succeeds")
}

func TestStorePendingIndexTracksTheWriteThroughState(t *testing.T) {
	t.Parallel()

	store := newStore(t)
	instanceUID := uuid.New()
	reportedAt := time.Now()

	// An observation with no write-through behind it is pending.
	mustTouch(t, store, observationAt(instanceUID, reportedAt))

	pending, err := store.ListPendingWriteThrough(t.Context(), time.Now(), 0)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	assert.Equal(t, instanceUID, pending[0].InstanceUID)

	// Writing it through clears it from the index.
	persistedAt(t, store, instanceUID, reportedAt, time.Now())

	pending, err = store.ListPendingWriteThrough(t.Context(), time.Now(), 0)
	require.NoError(t, err)
	assert.Empty(t, pending, "a written-through record must leave the pending index")
}

func TestStorePendingRespectsTheCutoff(t *testing.T) {
	t.Parallel()

	store := newStore(t)
	now := time.Now()

	overdue := uuid.New()
	justWritten := uuid.New()

	persistedAt(t, store, overdue, now, now.Add(-10*time.Minute))
	persistedAt(t, store, justWritten, now, now.Add(-time.Second))

	pending, err := store.ListPendingWriteThrough(t.Context(), now.Add(-time.Minute), 0)
	require.NoError(t, err)
	require.Len(t, pending, 1, "the cutoff is what keeps the flush from duplicating throttled writes")
	assert.Equal(t, overdue, pending[0].InstanceUID)
}

func TestStorePendingIsOrderedOldestFirstAndBounded(t *testing.T) {
	t.Parallel()

	store := newStore(t)
	now := time.Now()

	oldest := uuid.New()
	middle := uuid.New()
	newest := uuid.New()

	persistedAt(t, store, middle, now, now.Add(-20*time.Minute))
	persistedAt(t, store, newest, now, now.Add(-10*time.Minute))
	persistedAt(t, store, oldest, now, now.Add(-30*time.Minute))

	pending, err := store.ListPendingWriteThrough(t.Context(), now.Add(-time.Minute), 2)
	require.NoError(t, err)
	require.Len(t, pending, 2, "the limit must bound the batch")
	assert.Equal(t, oldest, pending[0].InstanceUID,
		"a saturated batch must drain the agents closest to falling out of the staleness window first")
	assert.Equal(t, middle, pending[1].InstanceUID)
}

func TestStoreTouchNil(t *testing.T) {
	t.Parallel()

	store := newStore(t)

	_, err := store.Touch(t.Context(), nil)
	require.NoError(t, err)
}

func TestNewRejectsMissingEndpoints(t *testing.T) {
	t.Parallel()

	//exhaustruct:ignore
	_, err := livenessredis.New(livenessredis.Config{})
	require.ErrorIs(t, err, livenessredis.ErrNoEndpoints)
}

func TestStoreTouchReturnsThePreservedAnchor(t *testing.T) {
	t.Parallel()

	store := newStore(t)
	instanceUID := uuid.New()
	observed := time.Now().Truncate(time.Nanosecond)
	anchored := observed.Add(-time.Minute)

	mustTouch(t, store, observationAt(instanceUID, observed))
	require.NoError(t, store.MarkPersisted(t.Context(), instanceUID, anchored))

	// The anchor has to come back from the write itself: this runs on every agent
	// message, and a read-then-write would double both the latency and the op count.
	stored, err := store.Touch(t.Context(), observationAt(instanceUID, observed.Add(time.Second)))
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, anchored.UnixNano(), stored.DurableReportedAt.UnixNano())
}

func TestStoreTouchIgnoresACallerSuppliedAnchor(t *testing.T) {
	t.Parallel()

	store := newStore(t)
	instanceUID := uuid.New()
	now := time.Now()

	mustTouch(t, store, observationAt(instanceUID, now.Add(-time.Minute)))
	require.NoError(t, store.MarkPersisted(t.Context(), instanceUID, now.Add(-time.Minute)))

	// Only MarkPersisted moves the anchor. An observation claiming otherwise must not
	// be able to reset the throttle window.
	observation := observationAt(instanceUID, now)
	observation.DurableReportedAt = now

	stored, err := store.Touch(t.Context(), observation)
	require.NoError(t, err)
	assert.Equal(t, now.Add(-time.Minute).UnixNano(), stored.DurableReportedAt.UnixNano())
}

func TestStoreQuietAgentLeavesThePendingSet(t *testing.T) {
	t.Parallel()

	store := newStore(t)
	instanceUID := uuid.New()
	now := time.Now()

	// Anchored at its own observation: nothing left to write through, even though the
	// index still carries the member.
	mustTouch(t, store, observationAt(instanceUID, now))
	require.NoError(t, store.MarkPersisted(t.Context(), instanceUID, now))

	pending, err := store.ListPendingWriteThrough(t.Context(), time.Now(), 0)
	require.NoError(t, err)
	assert.Empty(t, pending, "an agent with nothing new to write must not be flushed again")
}
