package agentservice_test

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var errLivenessTierDown = errors.New("liveness tier down")

// brokenLivenessPort fails every call, standing in for an unavailable fast tier.
type brokenLivenessPort struct{}

func (brokenLivenessPort) Touch(
	context.Context,
	*agentmodel.AgentLiveness,
) (*agentmodel.AgentLiveness, error) {
	return nil, errLivenessTierDown
}

func (brokenLivenessPort) MarkPersisted(context.Context, uuid.UUID, time.Time) error {
	return errLivenessTierDown
}

func (brokenLivenessPort) Get(context.Context, uuid.UUID) (*agentmodel.AgentLiveness, error) {
	return nil, errLivenessTierDown
}

func (brokenLivenessPort) GetMany(
	context.Context,
	[]uuid.UUID,
) (map[uuid.UUID]*agentmodel.AgentLiveness, error) {
	return nil, errLivenessTierDown
}

func (brokenLivenessPort) Delete(context.Context, uuid.UUID) error {
	return errLivenessTierDown
}

func (brokenLivenessPort) ListPendingWriteThrough(
	context.Context,
	time.Time,
	int,
) ([]*agentmodel.AgentLiveness, error) {
	return nil, errLivenessTierDown
}

// mustTouchLiveness records an observation into a fake tier for test setup.
func mustTouchLiveness(t *testing.T, port *fakeLivenessPort, liveness *agentmodel.AgentLiveness) {
	t.Helper()

	_, err := port.Touch(t.Context(), liveness)
	require.NoError(t, err)
}

func TestTouchAgentLiveness_FirstObservationIsDue(t *testing.T) {
	t.Parallel()

	persistence := new(MockAgentPersistencePort)
	liveness := newFakeLivenessPort()
	service := agentservice.NewAgentService(
		persistence, liveness, newFakeLivenessMetrics(), slog.Default(),
		agentservice.DefaultAgentCacheConfig(), agentservice.DefaultAgentLivenessConfig(), "", nil,
	)

	agent := agentmodel.NewAgent(uuid.New())

	assert.True(t, service.TouchAgentLiveness(t.Context(), agent, time.Now()),
		"an agent with no write-through anchor is always due")

	stored, err := liveness.Get(t.Context(), agent.Metadata.InstanceUID)
	require.NoError(t, err)
	require.NotNil(t, stored, "the observation must land in the fast tier even when it is also due")
}

func TestTouchAgentLiveness_ThrottlesAfterSave(t *testing.T) {
	t.Parallel()

	persistence := new(MockAgentPersistencePort)
	persistence.On("PutAgent", mock.Anything, mock.Anything).Return(nil)

	liveness := newFakeLivenessPort()
	service := agentservice.NewAgentService(
		persistence, liveness, newFakeLivenessMetrics(), slog.Default(),
		agentservice.DefaultAgentCacheConfig(),
		agentservice.AgentLivenessConfig{PersistThrottle: time.Hour},
		"",
		nil,
	)

	agent := agentmodel.NewAgent(uuid.New())
	agent.Status.LastReportedAt = time.Now()

	// SaveAgent anchors the throttle window with the observation it wrote, so the
	// next one is not due.
	require.NoError(t, service.SaveAgent(t.Context(), agent))

	assert.False(t, service.TouchAgentLiveness(t.Context(), agent, time.Now()),
		"an agent written through within the window must not be written again")
}

func TestTouchAgentLiveness_PreservesTheWriteThroughAnchor(t *testing.T) {
	t.Parallel()

	persistence := new(MockAgentPersistencePort)
	persistence.On("PutAgent", mock.Anything, mock.Anything).Return(nil)

	liveness := newFakeLivenessPort()
	service := agentservice.NewAgentService(
		persistence, liveness, newFakeLivenessMetrics(), slog.Default(),
		agentservice.DefaultAgentCacheConfig(),
		agentservice.AgentLivenessConfig{PersistThrottle: time.Hour},
		"",
		nil,
	)

	agent := agentmodel.NewAgent(uuid.New())
	agent.Status.LastReportedAt = time.Now()
	require.NoError(t, service.SaveAgent(t.Context(), agent))

	// Repeated observations must not reset the anchor — that would restart the
	// throttle window on every heartbeat and defeat the whole mechanism.
	for range 5 {
		require.False(t, service.TouchAgentLiveness(t.Context(), agent, time.Now()))
	}
}

func TestForgetAgentLiveness_MakesTheNextObservationDue(t *testing.T) {
	t.Parallel()

	persistence := new(MockAgentPersistencePort)
	persistence.On("PutAgent", mock.Anything, mock.Anything).Return(nil)

	liveness := newFakeLivenessPort()
	service := agentservice.NewAgentService(
		persistence, liveness, newFakeLivenessMetrics(), slog.Default(),
		agentservice.DefaultAgentCacheConfig(),
		agentservice.AgentLivenessConfig{PersistThrottle: time.Hour},
		"",
		nil,
	)

	agent := agentmodel.NewAgent(uuid.New())
	agent.Status.LastReportedAt = time.Now()
	require.NoError(t, service.SaveAgent(t.Context(), agent))

	require.NoError(t, service.ForgetAgentLiveness(t.Context(), agent.Metadata.InstanceUID))

	assert.True(t, service.TouchAgentLiveness(t.Context(), agent, time.Now()),
		"a reconnect must not inherit the previous session's throttle window")
}

func TestTouchAgentLiveness_FastTierFailureDegradesToADurableWrite(t *testing.T) {
	t.Parallel()

	persistence := new(MockAgentPersistencePort)
	service := agentservice.NewAgentService(
		persistence, brokenLivenessPort{}, newFakeLivenessMetrics(), slog.Default(),
		agentservice.DefaultAgentCacheConfig(),
		agentservice.AgentLivenessConfig{PersistThrottle: time.Hour},
		"",
		nil,
	)

	agent := agentmodel.NewAgent(uuid.New())

	assert.True(t, service.TouchAgentLiveness(t.Context(), agent, time.Now()),
		"with no usable fast tier the durable store is the only place left to write")
}

func TestSaveAgent_SurvivesAFastTierFailure(t *testing.T) {
	t.Parallel()

	persistence := new(MockAgentPersistencePort)
	persistence.On("PutAgent", mock.Anything, mock.Anything).Return(nil)

	service := agentservice.NewAgentService(
		persistence, brokenLivenessPort{}, newFakeLivenessMetrics(), slog.Default(),
		agentservice.DefaultAgentCacheConfig(), agentservice.DefaultAgentLivenessConfig(), "", nil,
	)

	require.NoError(t, service.SaveAgent(t.Context(), agentmodel.NewAgent(uuid.New())))
	persistence.AssertExpectations(t)
}

func TestTouchAgentLiveness_NilAgent(t *testing.T) {
	t.Parallel()

	service := agentservice.NewAgentService(
		new(MockAgentPersistencePort), newFakeLivenessPort(), newFakeLivenessMetrics(), slog.Default(),
		agentservice.DefaultAgentCacheConfig(), agentservice.DefaultAgentLivenessConfig(), "", nil,
	)

	assert.False(t, service.TouchAgentLiveness(t.Context(), nil, time.Now()))
}

// countingLivenessPort counts calls so a test can assert on the number of fast-tier
// round trips, not just the outcome.
type countingLivenessPort struct {
	*fakeLivenessPort

	touches       atomic.Int64
	gets          atomic.Int64
	markPersisted atomic.Int64
}

func newCountingLivenessPort() *countingLivenessPort {
	//exhaustruct:ignore
	return &countingLivenessPort{fakeLivenessPort: newFakeLivenessPort()}
}

func (c *countingLivenessPort) Touch(
	ctx context.Context,
	liveness *agentmodel.AgentLiveness,
) (*agentmodel.AgentLiveness, error) {
	c.touches.Add(1)

	return c.fakeLivenessPort.Touch(ctx, liveness)
}

func (c *countingLivenessPort) Get(
	ctx context.Context,
	instanceUID uuid.UUID,
) (*agentmodel.AgentLiveness, error) {
	c.gets.Add(1)

	return c.fakeLivenessPort.Get(ctx, instanceUID)
}

func (c *countingLivenessPort) MarkPersisted(
	ctx context.Context,
	instanceUID uuid.UUID,
	at time.Time,
) error {
	c.markPersisted.Add(1)

	return c.fakeLivenessPort.MarkPersisted(ctx, instanceUID, at)
}

func TestTouchAgentLivenessCostsOneRoundTrip(t *testing.T) {
	t.Parallel()

	liveness := newCountingLivenessPort()
	service := agentservice.NewAgentService(
		new(MockAgentPersistencePort), liveness, newFakeLivenessMetrics(), slog.Default(),
		agentservice.AgentCacheConfig{Enabled: false, TTL: 0, MaxCapacity: 0},
		agentservice.DefaultAgentLivenessConfig(), "", nil,
	)

	agent := agentmodel.NewAgent(uuid.New())

	const messages = 10
	for range messages {
		service.TouchAgentLiveness(t.Context(), agent, time.Now())
	}

	// This runs on every agent message. A read-then-write would double both the
	// latency and the op count against a shared tier, so the anchor has to come
	// back from the write itself.
	assert.Equal(t, int64(messages), liveness.touches.Load())
	assert.Equal(t, int64(0), liveness.gets.Load(), "the hot path must not read before it writes")
}

func TestSaveAgentAnchorsWithoutReading(t *testing.T) {
	t.Parallel()

	persistence := new(MockAgentPersistencePort)
	persistence.On("PutAgent", mock.Anything, mock.Anything).Return(nil)

	liveness := newCountingLivenessPort()
	service := agentservice.NewAgentService(
		persistence, liveness, newFakeLivenessMetrics(), slog.Default(),
		agentservice.AgentCacheConfig{Enabled: false, TTL: 0, MaxCapacity: 0},
		agentservice.DefaultAgentLivenessConfig(), "", nil,
	)

	require.NoError(t, service.SaveAgent(t.Context(), agentmodel.NewAgent(uuid.New())))

	assert.Equal(t, int64(1), liveness.markPersisted.Load())
	assert.Equal(t, int64(0), liveness.gets.Load(),
		"anchoring writes one field; reading the record first would only invite a lost update")
}

// newLivenessRecord builds a fast-tier record that reports the agent as connected at `at`.
func newLivenessRecord(instanceUID uuid.UUID, at time.Time, sequenceNum uint64) *agentmodel.AgentLiveness {
	return &agentmodel.AgentLiveness{
		InstanceUID:       instanceUID,
		Connected:         true,
		ConnectionType:    agentmodel.ConnectionTypeWebSocket,
		SequenceNum:       sequenceNum,
		LastReportedAt:    at,
		LastReportedTo:    "server-b",
		DurableReportedAt: time.Time{},
	}
}

// staleStoredAgent is an agent whose durable document has not been written through
// since `at` — the state a heartbeat-throttled agent is in between write-throughs.
func staleStoredAgent(instanceUID uuid.UUID, at time.Time) *agentmodel.Agent {
	agent := agentmodel.NewAgent(instanceUID)
	agent.Status.Connected = true
	agent.Status.SequenceNum = 1
	agent.Status.LastReportedAt = at
	agent.Status.LastReportedTo = "server-a"

	return agent
}

func newMergeService(
	persistence *MockAgentPersistencePort,
	liveness *fakeLivenessPort,
) *agentservice.AgentService {
	return agentservice.NewAgentService(
		persistence, liveness, newFakeLivenessMetrics(), slog.Default(),
		agentservice.AgentCacheConfig{Enabled: false, TTL: 0, MaxCapacity: 0},
		agentservice.DefaultAgentLivenessConfig(),
		"",
		nil,
	)
}

func TestGetAgent_MergesLivenessOverTheStoredDocument(t *testing.T) {
	t.Parallel()

	instanceUID := uuid.New()
	stored := time.Now().Add(-5 * time.Minute)

	persistence := new(MockAgentPersistencePort)
	persistence.On("GetAgent", mock.Anything, instanceUID).
		Return(staleStoredAgent(instanceUID, stored), nil)

	liveness := newFakeLivenessPort()
	fresh := time.Now()
	mustTouchLiveness(t, liveness, newLivenessRecord(instanceUID, fresh, 99))

	agent, err := newMergeService(persistence, liveness).GetAgent(t.Context(), instanceUID)
	require.NoError(t, err)
	assert.Equal(t, fresh, agent.Status.LastReportedAt)
	assert.Equal(t, uint64(99), agent.Status.SequenceNum)
	assert.Equal(t, "server-b", agent.Status.LastReportedTo)
	assert.True(t, agent.IsConnectedAt(time.Now(), agentmodel.DefaultConnectionStaleness),
		"an agent whose write-through is overdue must still read as connected")
}

func TestGetAgent_KeepsTheStoredDocumentWhenTheFastTierIsBehind(t *testing.T) {
	t.Parallel()

	instanceUID := uuid.New()
	fresh := time.Now()

	persistence := new(MockAgentPersistencePort)
	persistence.On("GetAgent", mock.Anything, instanceUID).
		Return(staleStoredAgent(instanceUID, fresh), nil)

	liveness := newFakeLivenessPort()
	stale := fresh.Add(-time.Hour)
	mustTouchLiveness(t, liveness, newLivenessRecord(instanceUID, stale, 1))

	agent, err := newMergeService(persistence, liveness).GetAgent(t.Context(), instanceUID)
	require.NoError(t, err)
	assert.Equal(t, fresh, agent.Status.LastReportedAt, "a record left by a crashed node must not win")
	assert.Equal(t, "server-a", agent.Status.LastReportedTo)
}

func TestGetAgent_SurvivesAFastTierFailure(t *testing.T) {
	t.Parallel()

	instanceUID := uuid.New()
	stored := time.Now()

	persistence := new(MockAgentPersistencePort)
	persistence.On("GetAgent", mock.Anything, instanceUID).
		Return(staleStoredAgent(instanceUID, stored), nil)

	service := agentservice.NewAgentService(
		persistence, brokenLivenessPort{}, newFakeLivenessMetrics(), slog.Default(),
		agentservice.AgentCacheConfig{Enabled: false, TTL: 0, MaxCapacity: 0},
		agentservice.DefaultAgentLivenessConfig(), "", nil,
	)

	agent, err := service.GetAgent(t.Context(), instanceUID)
	require.NoError(t, err)
	assert.Equal(t, stored, agent.Status.LastReportedAt, "the durable document is still a valid answer")
}

func TestListAgents_MergesLivenessAcrossThePage(t *testing.T) {
	t.Parallel()

	merged := uuid.New()
	untouched := uuid.New()
	stored := time.Now().Add(-5 * time.Minute)

	page := &model.ListResponse[*agentmodel.Agent]{
		Items: []*agentmodel.Agent{
			staleStoredAgent(merged, stored),
			staleStoredAgent(untouched, stored),
		},
		RemainingItemCount: 0,
		Continue:           "",
	}

	persistence := new(MockAgentPersistencePort)
	persistence.On("ListAgents", mock.Anything, mock.Anything, mock.Anything).Return(page, nil)

	liveness := newFakeLivenessPort()
	fresh := time.Now()
	mustTouchLiveness(t, liveness, newLivenessRecord(merged, fresh, 42))

	//exhaustruct:ignore
	result, err := newMergeService(persistence, liveness).
		ListAgents(t.Context(), agentmodel.DefaultNamespaceName, &model.ListOptions{Limit: 10})
	require.NoError(t, err)
	require.Len(t, result.Items, 2)

	assert.Equal(t, fresh, result.Items[0].Status.LastReportedAt)
	assert.Equal(t, uint64(42), result.Items[0].Status.SequenceNum)
	assert.Equal(t, stored, result.Items[1].Status.LastReportedAt,
		"an agent with no fast-tier record keeps its stored state")
}

func TestListAgents_SurvivesAFastTierFailure(t *testing.T) {
	t.Parallel()

	instanceUID := uuid.New()
	stored := time.Now()

	page := &model.ListResponse[*agentmodel.Agent]{
		Items:              []*agentmodel.Agent{staleStoredAgent(instanceUID, stored)},
		RemainingItemCount: 0,
		Continue:           "",
	}

	persistence := new(MockAgentPersistencePort)
	persistence.On("ListAgents", mock.Anything, mock.Anything, mock.Anything).Return(page, nil)

	service := agentservice.NewAgentService(
		persistence, brokenLivenessPort{}, newFakeLivenessMetrics(), slog.Default(),
		agentservice.AgentCacheConfig{Enabled: false, TTL: 0, MaxCapacity: 0},
		agentservice.DefaultAgentLivenessConfig(), "", nil,
	)

	//exhaustruct:ignore
	result, err := service.ListAgents(t.Context(), agentmodel.DefaultNamespaceName, &model.ListOptions{Limit: 10})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Equal(t, stored, result.Items[0].Status.LastReportedAt)
}

func TestDeleteAgent_RefusesAnAgentThatIsOnlyLiveInTheFastTier(t *testing.T) {
	t.Parallel()

	instanceUID := uuid.New()

	// The durable document is stale enough to read as disconnected on its own.
	stale := staleStoredAgent(instanceUID, time.Now().Add(-2*agentmodel.DefaultConnectionStaleness))

	persistence := new(MockAgentPersistencePort)
	persistence.On("GetAgent", mock.Anything, instanceUID).Return(stale, nil)

	liveness := newFakeLivenessPort()
	mustTouchLiveness(t, liveness, newLivenessRecord(instanceUID, time.Now(), 7))

	err := newMergeService(persistence, liveness).DeleteAgent(t.Context(), instanceUID)
	require.Error(t, err, "a live agent must not be deleted just because its write-through is overdue")
	persistence.AssertNotCalled(t, "DeleteAgent", mock.Anything, mock.Anything)
}

func TestGetOrCreateAgentSkipsTheMerge(t *testing.T) {
	t.Parallel()

	instanceUID := uuid.New()

	persistence := new(MockAgentPersistencePort)
	persistence.On("GetAgent", mock.Anything, instanceUID).
		Return(staleStoredAgent(instanceUID, time.Now()), nil)

	liveness := newCountingLivenessPort()
	service := agentservice.NewAgentService(
		persistence, liveness, newFakeLivenessMetrics(), slog.Default(),
		agentservice.AgentCacheConfig{Enabled: false, TTL: 0, MaxCapacity: 0},
		agentservice.DefaultAgentLivenessConfig(), "", nil,
	)

	_, err := service.GetOrCreateAgent(t.Context(), instanceUID)
	require.NoError(t, err)

	// The OpAMP message path overwrites every liveness field from the message it is
	// handling, so a merge here would be a fast-tier round trip per agent message
	// spent on values that are discarded microseconds later.
	assert.Equal(t, int64(0), liveness.gets.Load(),
		"the message path must not pay for a merge it immediately overwrites")

	// The API read path still merges.
	_, err = service.GetAgent(t.Context(), instanceUID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), liveness.gets.Load())
}

func TestLivenessMetricsCountTheSavedWrites(t *testing.T) {
	t.Parallel()

	persistence := new(MockAgentPersistencePort)
	persistence.On("PutAgent", mock.Anything, mock.Anything).Return(nil)
	persistence.On("UpdateAgentLiveness", mock.Anything, mock.Anything).Return(nil)

	metrics := newFakeLivenessMetrics()
	service := agentservice.NewAgentService(
		persistence, newFakeLivenessPort(), metrics, slog.Default(),
		agentservice.AgentCacheConfig{Enabled: false, TTL: 0, MaxCapacity: 0},
		agentservice.AgentLivenessConfig{PersistThrottle: time.Hour},
		"",
		nil,
	)

	agent := agentmodel.NewAgent(uuid.New())
	agent.Status.LastReportedAt = time.Now()

	// One full document write anchors the throttle window.
	require.NoError(t, service.SaveAgent(t.Context(), agent))

	// Every heartbeat inside the window is absorbed instead of written.
	const heartbeats = 5

	for range heartbeats {
		require.False(t, service.TouchAgentLiveness(t.Context(), agent, time.Now()))
	}

	// The write-behind flush writes narrowly.
	require.NoError(t, service.PersistAgentLiveness(t.Context(),
		agentmodel.NewAgentLivenessFromAgent(agent)))

	absorbed, written := metrics.counts()
	assert.Equal(t, heartbeats, absorbed, "absorbed observations are the database writes the tier saved")
	assert.Equal(t, 1, written[agentport.LivenessWriteShapeDocument])
	assert.Equal(t, 1, written[agentport.LivenessWriteShapeLiveness],
		"the flush must be visible as a liveness-shaped write, not lumped in with document rewrites")
}
