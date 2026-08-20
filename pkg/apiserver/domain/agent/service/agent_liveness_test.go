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
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
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

func TestTouchAgentLiveness_FirstObservationIsDue(t *testing.T) {
	t.Parallel()

	persistence := new(MockAgentPersistencePort)
	liveness := newFakeLivenessPort()
	service := agentservice.NewAgentService(
		persistence, liveness, slog.Default(),
		agentservice.DefaultAgentCacheConfig(), agentservice.DefaultAgentLivenessConfig(), "",
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
		persistence, liveness, slog.Default(),
		agentservice.DefaultAgentCacheConfig(),
		agentservice.AgentLivenessConfig{PersistThrottle: time.Hour},
		"",
	)

	agent := agentmodel.NewAgent(uuid.New())

	// SaveAgent anchors the throttle window, so the next observation is not due.
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
		persistence, liveness, slog.Default(),
		agentservice.DefaultAgentCacheConfig(),
		agentservice.AgentLivenessConfig{PersistThrottle: time.Hour},
		"",
	)

	agent := agentmodel.NewAgent(uuid.New())
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
		persistence, liveness, slog.Default(),
		agentservice.DefaultAgentCacheConfig(),
		agentservice.AgentLivenessConfig{PersistThrottle: time.Hour},
		"",
	)

	agent := agentmodel.NewAgent(uuid.New())
	require.NoError(t, service.SaveAgent(t.Context(), agent))

	require.NoError(t, service.ForgetAgentLiveness(t.Context(), agent.Metadata.InstanceUID))

	assert.True(t, service.TouchAgentLiveness(t.Context(), agent, time.Now()),
		"a reconnect must not inherit the previous session's throttle window")
}

func TestTouchAgentLiveness_FastTierFailureDegradesToADurableWrite(t *testing.T) {
	t.Parallel()

	persistence := new(MockAgentPersistencePort)
	service := agentservice.NewAgentService(
		persistence, brokenLivenessPort{}, slog.Default(),
		agentservice.DefaultAgentCacheConfig(),
		agentservice.AgentLivenessConfig{PersistThrottle: time.Hour},
		"",
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
		persistence, brokenLivenessPort{}, slog.Default(),
		agentservice.DefaultAgentCacheConfig(), agentservice.DefaultAgentLivenessConfig(), "",
	)

	require.NoError(t, service.SaveAgent(t.Context(), agentmodel.NewAgent(uuid.New())))
	persistence.AssertExpectations(t)
}

func TestTouchAgentLiveness_NilAgent(t *testing.T) {
	t.Parallel()

	service := agentservice.NewAgentService(
		new(MockAgentPersistencePort), newFakeLivenessPort(), slog.Default(),
		agentservice.DefaultAgentCacheConfig(), agentservice.DefaultAgentLivenessConfig(), "",
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
		new(MockAgentPersistencePort), liveness, slog.Default(),
		agentservice.AgentCacheConfig{Enabled: false, TTL: 0, MaxCapacity: 0},
		agentservice.DefaultAgentLivenessConfig(), "",
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
		persistence, liveness, slog.Default(),
		agentservice.AgentCacheConfig{Enabled: false, TTL: 0, MaxCapacity: 0},
		agentservice.DefaultAgentLivenessConfig(), "",
	)

	require.NoError(t, service.SaveAgent(t.Context(), agentmodel.NewAgent(uuid.New())))

	assert.Equal(t, int64(1), liveness.markPersisted.Load())
	assert.Equal(t, int64(0), liveness.gets.Load(),
		"anchoring writes one field; reading the record first would only invite a lost update")
}
