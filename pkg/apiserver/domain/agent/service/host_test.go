package agentservice_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/agent"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// fixedClock is a clock.PassiveClock that always reports a fixed time.
type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time                  { return c.now }
func (c fixedClock) Since(t time.Time) time.Duration { return c.now.Sub(t) }

// MockHostPersistencePort is a mock implementation of HostPersistencePort.
type MockHostPersistencePort struct {
	mock.Mock
}

func (m *MockHostPersistencePort) GetHost(ctx context.Context, id string) (*agentmodel.Host, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	host, ok := args.Get(0).(*agentmodel.Host)
	if !ok {
		return nil, errUnexpectedType
	}

	return host, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockHostPersistencePort) PutHost(ctx context.Context, host *agentmodel.Host) (*agentmodel.Host, error) {
	args := m.Called(ctx, host)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	saved, ok := args.Get(0).(*agentmodel.Host)
	if !ok {
		return nil, errUnexpectedType
	}

	return saved, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockHostPersistencePort) ListHosts(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Host], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*agentmodel.Host])
	if !ok {
		return nil, errUnexpectedType
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func hostAgent(t *testing.T) *agentmodel.Agent {
	t.Helper()

	a := agentmodel.NewAgent(uuid.New())
	require.NoError(t, a.ReportDescription(&agent.Description{
		IdentifyingAttributes: map[string]string{},
		NonIdentifyingAttributes: map[string]string{
			"host.id":   "h-1",
			"host.name": "node-1",
		},
	}))

	return a
}

func TestHostServiceObserveAgent(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	t.Run("creates a new host when none exists", func(t *testing.T) {
		t.Parallel()

		persistence := &MockHostPersistencePort{}
		svc := agentservice.NewHostService(persistence, fixedClock{now: now})
		a := hostAgent(t)

		persistence.On("GetHost", mock.Anything, "h-1").Return(nil, model.ErrResourceNotExist)
		persistence.On("PutHost", mock.Anything, mock.MatchedBy(func(h *agentmodel.Host) bool {
			return h.Metadata.ID == "h-1" &&
				len(h.Status.AgentInstanceUIDs) == 1 &&
				h.Status.AgentInstanceUIDs[0] == a.Metadata.InstanceUID
		})).Return(&agentmodel.Host{}, nil)

		require.NoError(t, svc.ObserveAgent(context.Background(), a))
		persistence.AssertExpectations(t)
	})

	t.Run("is a no-op when the agent reports no host attributes", func(t *testing.T) {
		t.Parallel()

		persistence := &MockHostPersistencePort{}
		svc := agentservice.NewHostService(persistence, fixedClock{now: now})
		a := agentmodel.NewAgent(uuid.New()) // no description reported

		require.NoError(t, svc.ObserveAgent(context.Background(), a))
		persistence.AssertNotCalled(t, "GetHost")
		persistence.AssertNotCalled(t, "PutHost")
	})

	t.Run("updates an existing host", func(t *testing.T) {
		t.Parallel()

		persistence := &MockHostPersistencePort{}
		svc := agentservice.NewHostService(persistence, fixedClock{now: now})
		a := hostAgent(t)
		existing := agentmodel.NewHost("h-1", now.Add(-time.Hour))

		persistence.On("GetHost", mock.Anything, "h-1").Return(existing, nil)
		persistence.On("PutHost", mock.Anything, mock.Anything).Return(&agentmodel.Host{}, nil)

		require.NoError(t, svc.ObserveAgent(context.Background(), a))
		assert.Equal(t, now, existing.Metadata.LastSeenAt)
		persistence.AssertExpectations(t)
	})

	t.Run("retries the read-modify-write when a concurrent writer wins", func(t *testing.T) {
		t.Parallel()

		persistence := &MockHostPersistencePort{}
		svc := agentservice.NewHostService(persistence, fixedClock{now: now})
		a := hostAgent(t)
		existing := agentmodel.NewHost("h-1", now.Add(-time.Hour))

		persistence.On("GetHost", mock.Anything, "h-1").Return(existing, nil)
		// The first write loses the optimistic-concurrency race; the retry re-reads
		// and succeeds instead of dropping this agent's observation.
		persistence.On("PutHost", mock.Anything, mock.Anything).Return(nil, model.ErrConflict).Once()
		persistence.On("PutHost", mock.Anything, mock.Anything).Return(&agentmodel.Host{}, nil).Once()

		require.NoError(t, svc.ObserveAgent(context.Background(), a))
		persistence.AssertNumberOfCalls(t, "PutHost", 2)
		persistence.AssertExpectations(t)
	})

	t.Run("gives up and surfaces the conflict after exhausting retries", func(t *testing.T) {
		t.Parallel()

		persistence := &MockHostPersistencePort{}
		svc := agentservice.NewHostService(persistence, fixedClock{now: now})
		a := hostAgent(t)
		existing := agentmodel.NewHost("h-1", now.Add(-time.Hour))

		persistence.On("GetHost", mock.Anything, "h-1").Return(existing, nil)
		persistence.On("PutHost", mock.Anything, mock.Anything).Return(nil, model.ErrConflict)

		require.ErrorIs(t, svc.ObserveAgent(context.Background(), a), model.ErrConflict)
	})
}
