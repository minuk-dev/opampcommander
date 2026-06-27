package agentservice_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/agent"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// MockContainerPersistencePort is a mock implementation of ContainerPersistencePort.
type MockContainerPersistencePort struct {
	mock.Mock
}

func (m *MockContainerPersistencePort) GetContainer(ctx context.Context, id string) (*agentmodel.Container, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	container, ok := args.Get(0).(*agentmodel.Container)
	if !ok {
		return nil, errUnexpectedType
	}

	return container, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockContainerPersistencePort) PutContainer(
	ctx context.Context,
	container *agentmodel.Container,
) (*agentmodel.Container, error) {
	args := m.Called(ctx, container)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	saved, ok := args.Get(0).(*agentmodel.Container)
	if !ok {
		return nil, errUnexpectedType
	}

	return saved, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockContainerPersistencePort) ListContainers(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Container], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*agentmodel.Container])
	if !ok {
		return nil, errUnexpectedType
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func containerAgent(t *testing.T) *agentmodel.Agent {
	t.Helper()

	a := agentmodel.NewAgent(uuid.New())
	require.NoError(t, a.ReportDescription(&agent.Description{
		IdentifyingAttributes: map[string]string{},
		NonIdentifyingAttributes: map[string]string{
			"k8s.pod.uid":   "pod-uid",
			"k8s.pod.name":  "otelcol-abc",
			"k8s.node.name": "node-1",
		},
	}))

	return a
}

func TestContainerServiceObserveAgent(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	t.Run("creates a new container keyed by pod uid", func(t *testing.T) {
		t.Parallel()

		persistence := &MockContainerPersistencePort{}
		svc := agentservice.NewContainerService(persistence, fixedClock{now: now})
		a := containerAgent(t)

		persistence.On("GetContainer", mock.Anything, "pod-uid").Return(nil, model.ErrResourceNotExist)
		persistence.On("PutContainer", mock.Anything, mock.MatchedBy(func(c *agentmodel.Container) bool {
			return c.Metadata.ID == "pod-uid" &&
				c.Spec.Platform == agent.PlatformKubernetes &&
				c.Spec.HostID == "node-1" &&
				len(c.Status.AgentInstanceUIDs) == 1
		})).Return(&agentmodel.Container{}, nil)

		require.NoError(t, svc.ObserveAgent(context.Background(), a))
		persistence.AssertExpectations(t)
	})

	t.Run("is a no-op when the agent reports no container attributes", func(t *testing.T) {
		t.Parallel()

		persistence := &MockContainerPersistencePort{}
		svc := agentservice.NewContainerService(persistence, fixedClock{now: now})
		a := agentmodel.NewAgent(uuid.New())

		require.NoError(t, svc.ObserveAgent(context.Background(), a))
		persistence.AssertNotCalled(t, "GetContainer")
		persistence.AssertNotCalled(t, "PutContainer")
	})
}
