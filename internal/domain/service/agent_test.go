package service_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agent"
	"github.com/minuk-dev/opampcommander/internal/domain/model/serverevent"
	"github.com/minuk-dev/opampcommander/internal/domain/service"
)

var (
	errDatabaseConnection = errors.New("database connection error")
	errUnexpectedType     = errors.New("unexpected type")
)

type MockAgentPersistencePort struct {
	mock.Mock
}

func (m *MockAgentPersistencePort) GetAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error) {
	args := m.Called(ctx, instanceUID)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	agnt, ok := args.Get(0).(*model.Agent)
	if !ok {
		return nil, errUnexpectedType
	}

	return agnt, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentPersistencePort) PutAgent(ctx context.Context, agnt *model.Agent) error {
	args := m.Called(ctx, agnt)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *MockAgentPersistencePort) ListAgents(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*model.Agent])
	if !ok {
		return nil, errUnexpectedType
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentPersistencePort) ListAgentsBySelector(
	ctx context.Context,
	selector model.AgentSelector,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	args := m.Called(ctx, selector, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*model.Agent])
	if !ok {
		return nil, errUnexpectedType
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentPersistencePort) SearchAgents(
	ctx context.Context,
	query string,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	args := m.Called(ctx, query, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*model.Agent])
	if !ok {
		return nil, errUnexpectedType
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

// MockServerMessageUsecase is a mock implementation of ServerMessageUsecase.
type MockServerMessageUsecase struct {
	mock.Mock
}

func (m *MockServerMessageUsecase) SendMessageToServerByServerID(
	ctx context.Context,
	serverID string,
	message serverevent.Message,
) error {
	args := m.Called(ctx, serverID, message)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *MockServerMessageUsecase) SendMessageToServer(
	ctx context.Context,
	server *model.Server,
	message serverevent.Message,
) error {
	args := m.Called(ctx, server, message)

	return args.Error(0) //nolint:wrapcheck // mock error
}

// MockServerIdentityProvider is a mock implementation of ServerIdentityProvider.
type MockServerIdentityProvider struct {
	mock.Mock
}

func (m *MockServerIdentityProvider) CurrentServer(ctx context.Context) (*model.Server, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	server, ok := args.Get(0).(*model.Server)
	if !ok {
		return nil, errUnexpectedType
	}

	return server, args.Error(1) //nolint:wrapcheck // mock error
}

func TestAgentService_ListAgentsBySelector(t *testing.T) {
	t.Parallel()

	t.Run("Successfully list agents by selector", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		mockAgentPersistence := new(MockAgentPersistencePort)
		logger := slog.Default()

		agentService := service.NewAgentService(
			mockAgentPersistence,
			logger,
		)

		agent1 := model.NewAgent(uuid.New(), model.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{
				"service.name": "test-service",
			},
			NonIdentifyingAttributes: map[string]string{
				"os.type": "linux",
			},
		}))

		agent2 := model.NewAgent(uuid.New(), model.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{
				"service.name": "test-service",
			},
			NonIdentifyingAttributes: map[string]string{
				"os.type": "darwin",
			},
		}))

		expectedResponse := &model.ListResponse[*model.Agent]{
			Items: []*model.Agent{
				agent1,
				agent2,
			},
			Continue:           "",
			RemainingItemCount: 0,
		}

		selector := model.AgentSelector{
			IdentifyingAttributes: map[string]string{
				"service.name": "test-service",
			},
			NonIdentifyingAttributes: map[string]string{},
		}

		options := &model.ListOptions{
			Limit:    10,
			Continue: "",
		}

		mockAgentPersistence.On("ListAgentsBySelector", ctx, selector, options).
			Return(expectedResponse, nil)

		listResponse, err := agentService.ListAgentsBySelector(ctx, selector, options)

		require.NoError(t, err)
		assert.NotNil(t, listResponse)
		assert.Len(t, listResponse.Items, 2)
		assert.Equal(t, agent1.Metadata.InstanceUID, listResponse.Items[0].Metadata.InstanceUID)
		assert.Equal(t, agent2.Metadata.InstanceUID, listResponse.Items[1].Metadata.InstanceUID)

		mockAgentPersistence.AssertExpectations(t)
	})

	t.Run("Empty result when no agents match selector", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		mockAgentPersistence := new(MockAgentPersistencePort)
		logger := slog.Default()

		agentService := service.NewAgentService(
			mockAgentPersistence,
			logger,
		)

		expectedResponse := &model.ListResponse[*model.Agent]{
			Items:              []*model.Agent{},
			Continue:           "",
			RemainingItemCount: 0,
		}

		selector := model.AgentSelector{
			IdentifyingAttributes: map[string]string{
				"service.name": "non-existent-service",
			},
			NonIdentifyingAttributes: map[string]string{},
		}

		options := &model.ListOptions{
			Limit:    10,
			Continue: "",
		}

		mockAgentPersistence.On("ListAgentsBySelector", ctx, selector, options).
			Return(expectedResponse, nil)

		listResponse, err := agentService.ListAgentsBySelector(ctx, selector, options)

		require.NoError(t, err)
		assert.NotNil(t, listResponse)
		assert.Empty(t, listResponse.Items)

		mockAgentPersistence.AssertExpectations(t)
	})

	t.Run("Error from persistence layer is propagated", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		mockAgentPersistence := new(MockAgentPersistencePort)
		logger := slog.Default()

		agentService := service.NewAgentService(
			mockAgentPersistence,
			logger,
		)

		selector := model.AgentSelector{
			IdentifyingAttributes: map[string]string{
				"service.name": "test-service",
			},
			NonIdentifyingAttributes: map[string]string{},
		}

		options := &model.ListOptions{
			Limit:    10,
			Continue: "",
		}

		mockAgentPersistence.On("ListAgentsBySelector", ctx, selector, options).
			Return(nil, errDatabaseConnection)

		listResponse, err := agentService.ListAgentsBySelector(ctx, selector, options)

		require.Error(t, err)
		assert.Nil(t, listResponse)
		assert.Contains(t, err.Error(), "failed to list agents by selector")

		mockAgentPersistence.AssertExpectations(t)
	})

	t.Run("List with pagination options", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		mockAgentPersistence := new(MockAgentPersistencePort)
		logger := slog.Default()

		agentService := service.NewAgentService(
			mockAgentPersistence,
			logger,
		)

		agents := make([]*model.Agent, 3)
		for idx := range 3 {
			agents[idx] = model.NewAgent(uuid.New(), model.WithDescription(&agent.Description{
				IdentifyingAttributes: map[string]string{
					"service.name": "test-service",
				},
				NonIdentifyingAttributes: map[string]string{
					"os.type": "linux",
				},
			}))
		}

		expectedResponse := &model.ListResponse[*model.Agent]{
			Items:              agents,
			Continue:           "some-continue-token",
			RemainingItemCount: 2,
		}

		selector := model.AgentSelector{
			IdentifyingAttributes: map[string]string{
				"service.name": "test-service",
			},
			NonIdentifyingAttributes: map[string]string{
				"os.type": "linux",
			},
		}

		options := &model.ListOptions{
			Limit:    3,
			Continue: "",
		}

		mockAgentPersistence.On("ListAgentsBySelector", ctx, selector, options).
			Return(expectedResponse, nil)

		listResponse, err := agentService.ListAgentsBySelector(ctx, selector, options)

		require.NoError(t, err)
		assert.NotNil(t, listResponse)
		assert.Len(t, listResponse.Items, 3)
		assert.Equal(t, "some-continue-token", listResponse.Continue)
		assert.Equal(t, int64(2), listResponse.RemainingItemCount)

		mockAgentPersistence.AssertExpectations(t)
	})

	t.Run("Match by non-identifying attributes only", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		mockAgentPersistence := new(MockAgentPersistencePort)
		logger := slog.Default()

		agentService := service.NewAgentService(
			mockAgentPersistence,
			logger,
		)

		agent1 := model.NewAgent(uuid.New(), model.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{
				"service.name": "service-a",
			},
			NonIdentifyingAttributes: map[string]string{
				"os.type": "linux",
			},
		}))

		agent2 := model.NewAgent(uuid.New(), model.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{
				"service.name": "service-b",
			},
			NonIdentifyingAttributes: map[string]string{
				"os.type": "linux",
			},
		}))

		expectedResponse := &model.ListResponse[*model.Agent]{
			Items: []*model.Agent{
				agent1,
				agent2,
			},
			Continue:           "",
			RemainingItemCount: 0,
		}

		selector := model.AgentSelector{
			IdentifyingAttributes: map[string]string{},
			NonIdentifyingAttributes: map[string]string{
				"os.type": "linux",
			},
		}

		options := &model.ListOptions{
			Limit:    10,
			Continue: "",
		}

		mockAgentPersistence.On("ListAgentsBySelector", ctx, selector, options).
			Return(expectedResponse, nil)

		listResponse, err := agentService.ListAgentsBySelector(ctx, selector, options)

		require.NoError(t, err)
		assert.NotNil(t, listResponse)
		assert.Len(t, listResponse.Items, 2)

		mockAgentPersistence.AssertExpectations(t)
	})
}

// Test for the ApplyRemoteConfig priority logic.
func TestAgent_ApplyRemoteConfig(t *testing.T) {
	t.Parallel()

	t.Run("Remote config name should be added to the list", func(t *testing.T) {
		t.Parallel()

		//exhaustruct:ignore
		agent := &model.Agent{}

		err := agent.ApplyRemoteConfig("config-1")
		require.NoError(t, err)
		assert.Contains(t, agent.Spec.RemoteConfig.RemoteConfigNames, "config-1")
	})

	t.Run("Multiple remote configs should be added and sorted", func(t *testing.T) {
		t.Parallel()

		//exhaustruct:ignore
		agent := &model.Agent{}

		err := agent.ApplyRemoteConfig("config-b")
		require.NoError(t, err)
		err = agent.ApplyRemoteConfig("config-a")
		require.NoError(t, err)

		assert.Equal(t, []string{"config-a", "config-b"}, agent.Spec.RemoteConfig.RemoteConfigNames)
	})

	t.Run("Duplicate remote config names should be deduplicated", func(t *testing.T) {
		t.Parallel()

		//exhaustruct:ignore
		agent := &model.Agent{}

		err := agent.ApplyRemoteConfig("config-1")
		require.NoError(t, err)
		err = agent.ApplyRemoteConfig("config-1")
		require.NoError(t, err)

		assert.Equal(t, []string{"config-1"}, agent.Spec.RemoteConfig.RemoteConfigNames)
	})
}

func TestAgentService_SearchAgents(t *testing.T) {
	t.Parallel()

	t.Run("SearchAgents returns matching agents", func(t *testing.T) {
		t.Parallel()

		// given
		ctx := context.Background()
		mockPort := new(MockAgentPersistencePort)
		agentService := service.NewAgentService(mockPort, slog.Default())

		instanceUID := uuid.New()
		expectedAgents := []*model.Agent{
			model.NewAgent(instanceUID),
		}
		expectedResponse := &model.ListResponse[*model.Agent]{
			Items:              expectedAgents,
			Continue:           "",
			RemainingItemCount: 0,
		}

		mockPort.On("SearchAgents", ctx, "1234", mock.Anything).Return(expectedResponse, nil)

		// when
		response, err := agentService.SearchAgents(ctx, "1234", &model.ListOptions{})

		// then
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Items, 1)
		assert.Equal(t, instanceUID, response.Items[0].Metadata.InstanceUID)
		mockPort.AssertExpectations(t)
	})

	t.Run("SearchAgents returns error on persistence failure", func(t *testing.T) {
		t.Parallel()

		// given
		ctx := context.Background()
		mockPort := new(MockAgentPersistencePort)
		agentService := service.NewAgentService(mockPort, slog.Default())

		mockPort.On("SearchAgents", ctx, "test", mock.Anything).Return(nil, errDatabaseConnection)

		// when
		response, err := agentService.SearchAgents(ctx, "test", &model.ListOptions{})

		// then
		require.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "failed to search agents")
		mockPort.AssertExpectations(t)
	})

	t.Run("SearchAgents with pagination", func(t *testing.T) {
		t.Parallel()

		// given
		ctx := context.Background()
		mockPort := new(MockAgentPersistencePort)
		agentService := service.NewAgentService(mockPort, slog.Default())

		expectedAgents := []*model.Agent{
			model.NewAgent(uuid.New()),
			model.NewAgent(uuid.New()),
		}
		expectedResponse := &model.ListResponse[*model.Agent]{
			Items:              expectedAgents,
			Continue:           "next-token",
			RemainingItemCount: 10,
		}

		options := &model.ListOptions{
			Limit:    2,
			Continue: "",
		}

		mockPort.On("SearchAgents", ctx, "abcd", options).Return(expectedResponse, nil)

		// when
		response, err := agentService.SearchAgents(ctx, "abcd", options)

		// then
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Items, 2)
		assert.Equal(t, "next-token", response.Continue)
		assert.Equal(t, int64(10), response.RemainingItemCount)
		mockPort.AssertExpectations(t)
	})
}
