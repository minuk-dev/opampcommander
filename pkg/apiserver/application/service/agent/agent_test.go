package agent_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/agent"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var errMockError = errors.New("mock error")

type MockAgentUsecase struct {
	mock.Mock
}

func (m *MockAgentUsecase) GetAgent(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.Agent, error) {
	args := m.Called(ctx, instanceUID)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	agent, ok := args.Get(0).(*agentmodel.Agent)
	if !ok {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	return agent, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentUsecase) GetOrCreateAgent(ctx context.Context, instanceUID uuid.UUID) (*agentmodel.Agent, error) {
	args := m.Called(ctx, instanceUID)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	agent, ok := args.Get(0).(*agentmodel.Agent)
	if !ok {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	return agent, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentUsecase) ListAgentsBySelector(
	ctx context.Context,
	selector agentmodel.AgentSelector,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	args := m.Called(ctx, selector, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*agentmodel.Agent])
	if !ok {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentUsecase) SaveAgent(ctx context.Context, agnt *agentmodel.Agent) error {
	args := m.Called(ctx, agnt)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *MockAgentUsecase) DeleteAgent(ctx context.Context, instanceUID uuid.UUID) error {
	args := m.Called(ctx, instanceUID)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *MockAgentUsecase) ListAgents(
	ctx context.Context,
	namespace string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	args := m.Called(ctx, namespace, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	//nolint:wrapcheck, forcetypeassert // mock error
	return args.Get(0).(*model.ListResponse[*agentmodel.Agent]), args.Error(1)
}

func (m *MockAgentUsecase) SearchAgents(
	ctx context.Context,
	namespace string,
	query string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	args := m.Called(ctx, namespace, query, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	//nolint:wrapcheck,forcetypeassert // mock error
	return args.Get(0).(*model.ListResponse[*agentmodel.Agent]), args.Error(1)
}

type MockAgentNotificationUsecase struct {
	mock.Mock
}

func (m *MockAgentNotificationUsecase) NotifyAgentUpdated(ctx context.Context, agnt *agentmodel.Agent) error {
	args := m.Called(ctx, agnt)

	return args.Error(0) //nolint:wrapcheck // mock error
}

// stubEndpointDetectionUsecase is a no-op agentport.EndpointDetectionUsecase for the
// agent-service tests, which do not exercise endpoint detection.
type stubEndpointDetectionUsecase struct{}

func (stubEndpointDetectionUsecase) ReconcileEndpointsFromRemoteConfig(
	context.Context, *agentmodel.AgentRemoteConfig,
) error {
	return nil
}

func (stubEndpointDetectionUsecase) ExtractEndpointsFromAgent(
	*agentmodel.Agent,
) ([]*agentmodel.Endpoint, error) {
	return nil, nil
}

func TestService_SearchAgents(t *testing.T) {
	t.Parallel()

	t.Run("SearchAgents returns matching agents", func(t *testing.T) {
		t.Parallel()

		// given
		ctx := t.Context()
		mockAgentUsecase := new(MockAgentUsecase)
		mockNotificationUsecase := new(MockAgentNotificationUsecase)
		service := agent.New(
			mockAgentUsecase, mockNotificationUsecase, stubEndpointDetectionUsecase{},
			noopCacheInvalidationPublisher{}, slog.Default())

		instanceUID := uuid.New()
		domainAgents := []*agentmodel.Agent{
			agentmodel.NewAgent(instanceUID),
		}
		domainResponse := &model.ListResponse[*agentmodel.Agent]{
			Items:              domainAgents,
			Continue:           "",
			RemainingItemCount: 0,
		}

		mockAgentUsecase.On("SearchAgents", ctx, "default", "1234", mock.Anything).Return(domainResponse, nil)

		// when
		response, err := service.SearchAgents(ctx, "default", "1234", &applicationport.ListOptions{})

		// then
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Items, 1)
		assert.Equal(t, instanceUID, response.Items[0].Metadata.InstanceUID)
		mockAgentUsecase.AssertExpectations(t)
	})

	t.Run("SearchAgents returns error on usecase failure", func(t *testing.T) {
		t.Parallel()

		// given
		ctx := t.Context()
		mockAgentUsecase := new(MockAgentUsecase)
		mockNotificationUsecase := new(MockAgentNotificationUsecase)
		service := agent.New(
			mockAgentUsecase, mockNotificationUsecase, stubEndpointDetectionUsecase{},
			noopCacheInvalidationPublisher{}, slog.Default())

		mockAgentUsecase.On("SearchAgents", ctx, "default", "test", mock.Anything).Return(nil, errMockError)

		// when
		response, err := service.SearchAgents(ctx, "default", "test", &applicationport.ListOptions{})

		// then
		require.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "failed to search agents")
		mockAgentUsecase.AssertExpectations(t)
	})

	t.Run("SearchAgents with pagination", func(t *testing.T) {
		t.Parallel()

		// given
		ctx := t.Context()
		mockAgentUsecase := new(MockAgentUsecase)
		mockNotificationUsecase := new(MockAgentNotificationUsecase)
		service := agent.New(
			mockAgentUsecase, mockNotificationUsecase, stubEndpointDetectionUsecase{},
			noopCacheInvalidationPublisher{}, slog.Default())

		domainAgents := []*agentmodel.Agent{
			agentmodel.NewAgent(uuid.New()),
			agentmodel.NewAgent(uuid.New()),
		}
		domainResponse := &model.ListResponse[*agentmodel.Agent]{
			Items:              domainAgents,
			Continue:           "next-token",
			RemainingItemCount: 10,
		}

		options := &applicationport.ListOptions{
			Limit:    2,
			Continue: "",
		}

		mockAgentUsecase.On("SearchAgents", ctx, "default", "abcd", options.ToDomain()).Return(domainResponse, nil)

		// when
		response, err := service.SearchAgents(ctx, "default", "abcd", options)

		// then
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Items, 2)
		assert.Equal(t, "next-token", response.Metadata.Continue)
		assert.Equal(t, int64(10), response.Metadata.RemainingItemCount)
		mockAgentUsecase.AssertExpectations(t)
	})
}

func TestService_DeleteAgent(t *testing.T) {
	t.Parallel()

	t.Run("deletes a disconnected agent", func(t *testing.T) {
		t.Parallel()

		// given
		ctx := t.Context()
		mockAgentUsecase := new(MockAgentUsecase)
		mockNotificationUsecase := new(MockAgentNotificationUsecase)
		service := agent.New(
			mockAgentUsecase, mockNotificationUsecase, stubEndpointDetectionUsecase{},
			noopCacheInvalidationPublisher{}, slog.Default())

		instanceUID := uuid.New()
		domainAgent := agentmodel.NewAgent(instanceUID) // Status.Connected defaults to false

		mockAgentUsecase.On("GetAgent", ctx, instanceUID).Return(domainAgent, nil)
		mockAgentUsecase.On("DeleteAgent", ctx, instanceUID).Return(nil)

		// when
		err := service.DeleteAgent(ctx, "default", instanceUID)

		// then
		require.NoError(t, err)
		mockAgentUsecase.AssertExpectations(t)
	})

	t.Run("propagates the domain connection guard (ErrAgentConnected)", func(t *testing.T) {
		t.Parallel()

		// given: the connection guard now lives in the domain DeleteAgent, so the
		// application service just surfaces ErrAgentConnected to the caller.
		ctx := t.Context()
		mockAgentUsecase := new(MockAgentUsecase)
		mockNotificationUsecase := new(MockAgentNotificationUsecase)
		service := agent.New(
			mockAgentUsecase, mockNotificationUsecase, stubEndpointDetectionUsecase{},
			noopCacheInvalidationPublisher{}, slog.Default())

		instanceUID := uuid.New()
		domainAgent := agentmodel.NewAgent(instanceUID) // namespace "default"

		mockAgentUsecase.On("GetAgent", ctx, instanceUID).Return(domainAgent, nil)
		mockAgentUsecase.On("DeleteAgent", ctx, instanceUID).Return(applicationport.ErrAgentConnected)

		// when
		err := service.DeleteAgent(ctx, "default", instanceUID)

		// then
		require.ErrorIs(t, err, applicationport.ErrAgentConnected)
		mockAgentUsecase.AssertExpectations(t)
	})

	t.Run("rejects deletion when namespace mismatches", func(t *testing.T) {
		t.Parallel()

		// given
		ctx := t.Context()
		mockAgentUsecase := new(MockAgentUsecase)
		mockNotificationUsecase := new(MockAgentNotificationUsecase)
		service := agent.New(
			mockAgentUsecase, mockNotificationUsecase, stubEndpointDetectionUsecase{},
			noopCacheInvalidationPublisher{}, slog.Default())

		instanceUID := uuid.New()
		domainAgent := agentmodel.NewAgent(instanceUID) // namespace defaults to "default"

		mockAgentUsecase.On("GetAgent", ctx, instanceUID).Return(domainAgent, nil)

		// when
		err := service.DeleteAgent(ctx, "other", instanceUID)

		// then
		require.ErrorIs(t, err, agent.ErrAgentNamespaceMismatch)
		mockAgentUsecase.AssertNotCalled(t, "DeleteAgent", mock.Anything, mock.Anything)
		mockAgentUsecase.AssertExpectations(t)
	})

	t.Run("returns error on usecase failure", func(t *testing.T) {
		t.Parallel()

		// given
		ctx := t.Context()
		mockAgentUsecase := new(MockAgentUsecase)
		mockNotificationUsecase := new(MockAgentNotificationUsecase)
		service := agent.New(
			mockAgentUsecase, mockNotificationUsecase, stubEndpointDetectionUsecase{},
			noopCacheInvalidationPublisher{}, slog.Default())

		instanceUID := uuid.New()
		domainAgent := agentmodel.NewAgent(instanceUID)

		mockAgentUsecase.On("GetAgent", ctx, instanceUID).Return(domainAgent, nil)
		mockAgentUsecase.On("DeleteAgent", ctx, instanceUID).Return(errMockError)

		// when
		err := service.DeleteAgent(ctx, "default", instanceUID)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete agent")
		mockAgentUsecase.AssertExpectations(t)
	})
}

// noopCacheInvalidationPublisher satisfies agentport.AgentCacheInvalidationPublisher in
// tests that do not assert on broadcasts.
type noopCacheInvalidationPublisher struct{}

func (noopCacheInvalidationPublisher) BroadcastAgentCacheInvalidation(
	context.Context, ...uuid.UUID,
) error {
	return nil
}

// spyCacheInvalidationPublisher records the UIDs it was asked to broadcast.
type spyCacheInvalidationPublisher struct {
	broadcasted []uuid.UUID
}

func (s *spyCacheInvalidationPublisher) BroadcastAgentCacheInvalidation(
	_ context.Context, instanceUIDs ...uuid.UUID,
) error {
	s.broadcasted = append(s.broadcasted, instanceUIDs...)

	return nil
}

func TestService_DeleteAgent_BroadcastsCacheInvalidation(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockAgentUsecase := new(MockAgentUsecase)
	mockNotificationUsecase := new(MockAgentNotificationUsecase)
	spy := new(spyCacheInvalidationPublisher)
	service := agent.New(mockAgentUsecase, mockNotificationUsecase, stubEndpointDetectionUsecase{}, spy, slog.Default())

	instanceUID := uuid.New()
	domainAgent := agentmodel.NewAgent(instanceUID)
	mockAgentUsecase.On("GetAgent", ctx, instanceUID).Return(domainAgent, nil)
	mockAgentUsecase.On("DeleteAgent", ctx, instanceUID).Return(nil)

	err := service.DeleteAgent(ctx, "default", instanceUID)
	require.NoError(t, err)

	require.Len(t, spy.broadcasted, 1)
	assert.Equal(t, instanceUID, spy.broadcasted[0])
}
