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

	"github.com/minuk-dev/opampcommander/internal/application/service/agent"
	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
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

func (m *MockAgentUsecase) ListAgents(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	//nolint:wrapcheck, forcetypeassert // mock error
	return args.Get(0).(*model.ListResponse[*agentmodel.Agent]), args.Error(1)
}

func (m *MockAgentUsecase) SearchAgents(
	ctx context.Context,
	query string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	args := m.Called(ctx, query, options)
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

func (m *MockAgentNotificationUsecase) RestartAgent(ctx context.Context, instanceUID uuid.UUID) error {
	args := m.Called(ctx, instanceUID)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func TestService_SearchAgents(t *testing.T) {
	t.Parallel()

	t.Run("SearchAgents returns matching agents", func(t *testing.T) {
		t.Parallel()

		// given
		ctx := t.Context()
		mockAgentUsecase := new(MockAgentUsecase)
		mockNotificationUsecase := new(MockAgentNotificationUsecase)
		service := agent.New(mockAgentUsecase, mockNotificationUsecase, slog.Default())

		instanceUID := uuid.New()
		domainAgents := []*agentmodel.Agent{
			agentmodel.NewAgent(instanceUID),
		}
		domainResponse := &model.ListResponse[*agentmodel.Agent]{
			Items:              domainAgents,
			Continue:           "",
			RemainingItemCount: 0,
		}

		mockAgentUsecase.On("SearchAgents", ctx, "1234", mock.Anything).Return(domainResponse, nil)

		// when
		response, err := service.SearchAgents(ctx, "1234", &model.ListOptions{})

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
		service := agent.New(mockAgentUsecase, mockNotificationUsecase, slog.Default())

		mockAgentUsecase.On("SearchAgents", ctx, "test", mock.Anything).Return(nil, errMockError)

		// when
		response, err := service.SearchAgents(ctx, "test", &model.ListOptions{})

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
		service := agent.New(mockAgentUsecase, mockNotificationUsecase, slog.Default())

		domainAgents := []*agentmodel.Agent{
			agentmodel.NewAgent(uuid.New()),
			agentmodel.NewAgent(uuid.New()),
		}
		domainResponse := &model.ListResponse[*agentmodel.Agent]{
			Items:              domainAgents,
			Continue:           "next-token",
			RemainingItemCount: 10,
		}

		options := &model.ListOptions{
			Limit:    2,
			Continue: "",
		}

		mockAgentUsecase.On("SearchAgents", ctx, "abcd", options).Return(domainResponse, nil)

		// when
		response, err := service.SearchAgents(ctx, "abcd", options)

		// then
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Items, 2)
		assert.Equal(t, "next-token", response.Metadata.Continue)
		assert.Equal(t, int64(10), response.Metadata.RemainingItemCount)
		mockAgentUsecase.AssertExpectations(t)
	})
}
