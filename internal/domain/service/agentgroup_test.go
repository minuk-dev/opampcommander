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
	"github.com/minuk-dev/opampcommander/internal/domain/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/module/helper"
)

var _ helper.Runner = (*service.AgentGroupService)(nil)

var errAgentGroupNotFound = errors.New("agent group not found")

// MockAgentGroupPersistencePort is a mock implementation of AgentGroupPersistencePort.
type MockAgentGroupPersistencePort struct {
	mock.Mock
}

func (m *MockAgentGroupPersistencePort) GetAgentGroup(
	ctx context.Context,
	name string,
) (*model.AgentGroup, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	agentGroup, ok := args.Get(0).(*model.AgentGroup)
	if !ok {
		return nil, errUnexpectedType
	}

	return agentGroup, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentGroupPersistencePort) PutAgentGroup(
	ctx context.Context,
	name string,
	agentGroup *model.AgentGroup,
) (*model.AgentGroup, error) {
	args := m.Called(ctx, name, agentGroup)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	result, ok := args.Get(0).(*model.AgentGroup)
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentGroupPersistencePort) ListAgentGroups(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.AgentGroup], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	result, ok := args.Get(0).(*model.ListResponse[*model.AgentGroup])
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck // mock error
}

// MockAgentUsecaseForGroup is a mock implementation of AgentUsecase for agent group tests.
type MockAgentUsecaseForGroup struct {
	mock.Mock
}

func (m *MockAgentUsecaseForGroup) GetAgent(
	ctx context.Context,
	instanceUID uuid.UUID,
) (*model.Agent, error) {
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

func (m *MockAgentUsecaseForGroup) GetOrCreateAgent(
	ctx context.Context,
	instanceUID uuid.UUID,
) (*model.Agent, error) {
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

func (m *MockAgentUsecaseForGroup) ListAgentsBySelector(
	ctx context.Context,
	selector model.AgentSelector,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	args := m.Called(ctx, selector, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	result, ok := args.Get(0).(*model.ListResponse[*model.Agent])
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentUsecaseForGroup) SaveAgent(ctx context.Context, agnt *model.Agent) error {
	args := m.Called(ctx, agnt)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *MockAgentUsecaseForGroup) ListAgents(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	result, ok := args.Get(0).(*model.ListResponse[*model.Agent])
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentUsecaseForGroup) SearchAgents(
	ctx context.Context,
	query string,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	args := m.Called(ctx, query, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	result, ok := args.Get(0).(*model.ListResponse[*model.Agent])
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck // mock error
}

// MockAgentRemoteConfigPersistencePort is a mock implementation of AgentRemoteConfigPersistencePort.
type MockAgentRemoteConfigPersistencePort struct {
	mock.Mock
}

func (m *MockAgentRemoteConfigPersistencePort) GetAgentRemoteConfig(
	ctx context.Context,
	name string,
) (*model.AgentRemoteConfig, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	result, ok := args.Get(0).(*model.AgentRemoteConfig)
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentRemoteConfigPersistencePort) PutAgentRemoteConfig(
	ctx context.Context,
	config *model.AgentRemoteConfig,
) (*model.AgentRemoteConfig, error) {
	args := m.Called(ctx, config)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	result, ok := args.Get(0).(*model.AgentRemoteConfig)
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentRemoteConfigPersistencePort) ListAgentRemoteConfigs(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.AgentRemoteConfig], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	result, ok := args.Get(0).(*model.ListResponse[*model.AgentRemoteConfig])
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck // mock error
}

func TestAgentGroupService_GetAgentGroup(t *testing.T) {
	t.Parallel()

	t.Run("Successfully get agent group", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockPersistence := new(MockAgentGroupPersistencePort)
		mockAgentUsecase := new(MockAgentUsecaseForGroup)
		mockRemoteConfigPort := new(MockAgentRemoteConfigPersistencePort)
		logger := slog.Default()

		svc := service.NewAgentGroupService(mockPersistence, mockRemoteConfigPort, mockAgentUsecase, logger)

		expectedGroup := &model.AgentGroup{
			Metadata: model.AgentGroupMetadata{
				Name: "test-group",
			},
		}

		mockPersistence.On("GetAgentGroup", ctx, "test-group").Return(expectedGroup, nil)

		result, err := svc.GetAgentGroup(ctx, "test-group")

		require.NoError(t, err)
		assert.Equal(t, "test-group", result.Metadata.Name)
		mockPersistence.AssertExpectations(t)
	})

	t.Run("Error when agent group not found", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockPersistence := new(MockAgentGroupPersistencePort)
		mockAgentUsecase := new(MockAgentUsecaseForGroup)
		mockRemoteConfigPort := new(MockAgentRemoteConfigPersistencePort)
		logger := slog.Default()

		svc := service.NewAgentGroupService(mockPersistence, mockRemoteConfigPort, mockAgentUsecase, logger)

		mockPersistence.On("GetAgentGroup", ctx, "non-existent").Return(nil, errAgentGroupNotFound)

		result, err := svc.GetAgentGroup(ctx, "non-existent")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "get agent group")
		mockPersistence.AssertExpectations(t)
	})
}

func TestAgentGroupService_ListAgentGroups(t *testing.T) {
	t.Parallel()

	t.Run("Successfully list agent groups", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockPersistence := new(MockAgentGroupPersistencePort)
		mockAgentUsecase := new(MockAgentUsecaseForGroup)
		mockRemoteConfigPort := new(MockAgentRemoteConfigPersistencePort)
		logger := slog.Default()

		svc := service.NewAgentGroupService(mockPersistence, mockRemoteConfigPort, mockAgentUsecase, logger)

		expectedResponse := &model.ListResponse[*model.AgentGroup]{
			Items: []*model.AgentGroup{
				{Metadata: model.AgentGroupMetadata{Name: "group-1"}},
				{Metadata: model.AgentGroupMetadata{Name: "group-2"}},
			},
			Continue:           "",
			RemainingItemCount: 0,
		}

		options := &model.ListOptions{Limit: 10}
		mockPersistence.On("ListAgentGroups", ctx, options).Return(expectedResponse, nil)

		result, err := svc.ListAgentGroups(ctx, options)

		require.NoError(t, err)
		assert.Len(t, result.Items, 2)
		mockPersistence.AssertExpectations(t)
	})
}

func TestAgentGroupService_ListAgentsByAgentGroup(t *testing.T) {
	t.Parallel()

	t.Run("Successfully list agents by agent group", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockPersistence := new(MockAgentGroupPersistencePort)
		mockAgentUsecase := new(MockAgentUsecaseForGroup)
		mockRemoteConfigPort := new(MockAgentRemoteConfigPersistencePort)
		logger := slog.Default()

		svc := service.NewAgentGroupService(mockPersistence, mockRemoteConfigPort, mockAgentUsecase, logger)

		agentGroup := &model.AgentGroup{
			Metadata: model.AgentGroupMetadata{
				Name: "test-group",
				Selector: model.AgentSelector{
					IdentifyingAttributes: map[string]string{
						"service.name": "test-service",
					},
				},
			},
		}

		agent1 := model.NewAgent(uuid.New(), model.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{
				"service.name": "test-service",
			},
		}))

		expectedResponse := &model.ListResponse[*model.Agent]{
			Items:              []*model.Agent{agent1},
			Continue:           "",
			RemainingItemCount: 0,
		}

		options := &model.ListOptions{Limit: 10}
		mockAgentUsecase.On("ListAgentsBySelector", ctx, agentGroup.Metadata.Selector, options).
			Return(expectedResponse, nil)

		result, err := svc.ListAgentsByAgentGroup(ctx, agentGroup, options)

		require.NoError(t, err)
		assert.Len(t, result.Items, 1)
		mockAgentUsecase.AssertExpectations(t)
	})
}

func TestAgentGroupService_GetAgentGroupsForAgent(t *testing.T) {
	t.Parallel()

	t.Run("Returns matching agent groups", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockPersistence := new(MockAgentGroupPersistencePort)
		mockAgentUsecase := new(MockAgentUsecaseForGroup)
		mockRemoteConfigPort := new(MockAgentRemoteConfigPersistencePort)
		logger := slog.Default()

		svc := service.NewAgentGroupService(mockPersistence, mockRemoteConfigPort, mockAgentUsecase, logger)

		testAgent := model.NewAgent(uuid.New(), model.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{
				"service.name": "my-service",
			},
			NonIdentifyingAttributes: map[string]string{
				"os.type": "linux",
			},
		}))

		matchingGroup := &model.AgentGroup{
			Metadata: model.AgentGroupMetadata{
				Name: "matching-group",
				Selector: model.AgentSelector{
					IdentifyingAttributes: map[string]string{
						"service.name": "my-service",
					},
				},
			},
		}

		nonMatchingGroup := &model.AgentGroup{
			Metadata: model.AgentGroupMetadata{
				Name: "non-matching-group",
				Selector: model.AgentSelector{
					IdentifyingAttributes: map[string]string{
						"service.name": "other-service",
					},
				},
			},
		}

		allGroups := &model.ListResponse[*model.AgentGroup]{
			Items:              []*model.AgentGroup{matchingGroup, nonMatchingGroup},
			Continue:           "",
			RemainingItemCount: 0,
		}

		mockPersistence.On("ListAgentGroups", ctx, (*model.ListOptions)(nil)).Return(allGroups, nil)

		result, err := svc.GetAgentGroupsForAgent(ctx, testAgent)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "matching-group", result[0].Metadata.Name)
		mockPersistence.AssertExpectations(t)
	})

	t.Run("Returns empty when no groups match", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockPersistence := new(MockAgentGroupPersistencePort)
		mockAgentUsecase := new(MockAgentUsecaseForGroup)
		mockRemoteConfigPort := new(MockAgentRemoteConfigPersistencePort)
		logger := slog.Default()

		svc := service.NewAgentGroupService(mockPersistence, mockRemoteConfigPort, mockAgentUsecase, logger)

		testAgent := model.NewAgent(uuid.New(), model.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{
				"service.name": "unique-service",
			},
		}))

		nonMatchingGroup := &model.AgentGroup{
			Metadata: model.AgentGroupMetadata{
				Name: "non-matching-group",
				Selector: model.AgentSelector{
					IdentifyingAttributes: map[string]string{
						"service.name": "other-service",
					},
				},
			},
		}

		allGroups := &model.ListResponse[*model.AgentGroup]{
			Items:              []*model.AgentGroup{nonMatchingGroup},
			Continue:           "",
			RemainingItemCount: 0,
		}

		mockPersistence.On("ListAgentGroups", ctx, (*model.ListOptions)(nil)).Return(allGroups, nil)

		result, err := svc.GetAgentGroupsForAgent(ctx, testAgent)

		require.NoError(t, err)
		assert.Empty(t, result)
		mockPersistence.AssertExpectations(t)
	})
}

func TestAgentGroupService_Name(t *testing.T) {
	t.Parallel()

	mockPersistence := new(MockAgentGroupPersistencePort)
	mockAgentUsecase := new(MockAgentUsecaseForGroup)
	mockRemoteConfigPort := new(MockAgentRemoteConfigPersistencePort)
	logger := slog.Default()

	svc := service.NewAgentGroupService(mockPersistence, mockRemoteConfigPort, mockAgentUsecase, logger)

	assert.Equal(t, "AgentGroupService", svc.Name())
}
