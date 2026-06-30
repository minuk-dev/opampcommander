package container_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	containersvc "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/container"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

var errMock = errors.New("mock error")

// mockContainerUsecase is a mock implementation of agentport.ContainerUsecase.
type mockContainerUsecase struct {
	mock.Mock
}

func (m *mockContainerUsecase) GetContainer(ctx context.Context, id string) (*agentmodel.Container, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	container, ok := args.Get(0).(*agentmodel.Container)
	if !ok {
		return nil, errMock
	}

	return container, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockContainerUsecase) ListContainers(
	ctx context.Context, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Container], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*agentmodel.Container])
	if !ok {
		return nil, errMock
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockContainerUsecase) ObserveAgent(ctx context.Context, agent *agentmodel.Agent) error {
	args := m.Called(ctx, agent)

	return args.Error(0) //nolint:wrapcheck // mock error
}

// mockAgentUsecase is a mock implementation of agentport.AgentUsecase.
type mockAgentUsecase struct {
	mock.Mock
}

func (m *mockAgentUsecase) GetAgent(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.Agent, error) {
	args := m.Called(ctx, instanceUID)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	agent, ok := args.Get(0).(*agentmodel.Agent)
	if !ok {
		return nil, errMock
	}

	return agent, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentUsecase) GetOrCreateAgent(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.Agent, error) {
	args := m.Called(ctx, instanceUID)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	agent, ok := args.Get(0).(*agentmodel.Agent)
	if !ok {
		return nil, errMock
	}

	return agent, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentUsecase) ListAgentsBySelector(
	ctx context.Context, selector agentmodel.AgentSelector, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	args := m.Called(ctx, selector, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*agentmodel.Agent])
	if !ok {
		return nil, errMock
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentUsecase) SaveAgent(ctx context.Context, agent *agentmodel.Agent) error {
	args := m.Called(ctx, agent)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *mockAgentUsecase) DeleteAgent(ctx context.Context, instanceUID uuid.UUID) error {
	args := m.Called(ctx, instanceUID)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *mockAgentUsecase) ListAgents(
	ctx context.Context, namespace string, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	args := m.Called(ctx, namespace, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*agentmodel.Agent])
	if !ok {
		return nil, errMock
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *mockAgentUsecase) SearchAgents(
	ctx context.Context, namespace string, query string, options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	args := m.Called(ctx, namespace, query, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*agentmodel.Agent])
	if !ok {
		return nil, errMock
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func newSvc(t *testing.T, container *mockContainerUsecase, agent *mockAgentUsecase) *containersvc.Service {
	t.Helper()

	base := testutil.NewBase(t)

	return containersvc.New(container, agent, base.Logger)
}

func newContainer(id string) *agentmodel.Container {
	return &agentmodel.Container{
		Metadata: agentmodel.ContainerMetadata{
			ID:   id,
			Name: "container-" + id,
		},
		Status: agentmodel.ContainerStatus{},
	}
}

func TestService_GetContainer(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockContainer := new(mockContainerUsecase)
		svc := newSvc(t, mockContainer, new(mockAgentUsecase))

		mockContainer.On("GetContainer", ctx, "c-1").Return(newContainer("c-1"), nil)

		result, err := svc.GetContainer(ctx, "c-1")

		require.NoError(t, err)
		assert.Equal(t, v1.ContainerKind, result.Kind)
		assert.Equal(t, "c-1", result.Metadata.ID)
		mockContainer.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockContainer := new(mockContainerUsecase)
		svc := newSvc(t, mockContainer, new(mockAgentUsecase))

		mockContainer.On("GetContainer", ctx, "missing").Return(nil, errMock)

		result, err := svc.GetContainer(ctx, "missing")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to get container")
		mockContainer.AssertExpectations(t)
	})
}

func TestService_ListContainers(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockContainer := new(mockContainerUsecase)
		svc := newSvc(t, mockContainer, new(mockAgentUsecase))

		opts := &applicationport.ListOptions{Limit: 10}
		resp := &model.ListResponse[*agentmodel.Container]{
			Items:    []*agentmodel.Container{newContainer("c-1"), newContainer("c-2")},
			Continue: "next",
		}
		mockContainer.On("ListContainers", ctx, opts.ToDomain()).Return(resp, nil)

		result, err := svc.ListContainers(ctx, opts)

		require.NoError(t, err)
		assert.Equal(t, v1.ContainerKind, result.Kind)
		assert.Len(t, result.Items, 2)
		assert.Equal(t, "next", result.Metadata.Continue)
		mockContainer.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockContainer := new(mockContainerUsecase)
		svc := newSvc(t, mockContainer, new(mockAgentUsecase))

		opts := &applicationport.ListOptions{Limit: 10}
		mockContainer.On("ListContainers", ctx, opts.ToDomain()).Return(nil, errMock)

		result, err := svc.ListContainers(ctx, opts)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to list containers")
		mockContainer.AssertExpectations(t)
	})
}

func TestService_ListAgentsByContainer(t *testing.T) {
	t.Parallel()

	t.Run("container with no agents returns empty list", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockContainer := new(mockContainerUsecase)
		mockAgent := new(mockAgentUsecase)
		svc := newSvc(t, mockContainer, mockAgent)

		mockContainer.On("GetContainer", ctx, "c-1").Return(newContainer("c-1"), nil)

		result, err := svc.ListAgentsByContainer(ctx, "c-1", &applicationport.ListOptions{Limit: 10})

		require.NoError(t, err)
		assert.Equal(t, v1.AgentKind, result.Kind)
		assert.Empty(t, result.Items)
		mockAgent.AssertNotCalled(t, "GetAgent", mock.Anything, mock.Anything)
		mockContainer.AssertExpectations(t)
	})

	t.Run("container lookup error", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockContainer := new(mockContainerUsecase)
		svc := newSvc(t, mockContainer, new(mockAgentUsecase))

		mockContainer.On("GetContainer", ctx, "missing").Return(nil, errMock)

		result, err := svc.ListAgentsByContainer(ctx, "missing", &applicationport.ListOptions{Limit: 10})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to get container")
		mockContainer.AssertExpectations(t)
	})
}
