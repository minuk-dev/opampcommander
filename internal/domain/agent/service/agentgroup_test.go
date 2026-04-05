package agentservice_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	"github.com/minuk-dev/opampcommander/internal/domain/agent/model/agent"
	agentservice "github.com/minuk-dev/opampcommander/internal/domain/agent/service"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/module/helper"
)

var _ helper.Runner = (*agentservice.AgentGroupService)(nil)

var errAgentGroupNotFound = errors.New("agent group not found")

// MockAgentGroupPersistencePort is a mock implementation of AgentGroupPersistencePort.
type MockAgentGroupPersistencePort struct {
	mock.Mock
}

func (m *MockAgentGroupPersistencePort) GetAgentGroup(
	ctx context.Context,
	namespace string,
	name string,
	options *model.GetOptions,
) (*agentmodel.AgentGroup, error) {
	args := m.Called(ctx, namespace, name, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	agentGroup, ok := args.Get(0).(*agentmodel.AgentGroup)
	if !ok {
		return nil, errUnexpectedType
	}

	return agentGroup, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentGroupPersistencePort) PutAgentGroup(
	ctx context.Context,
	namespace string,
	name string,
	agentGroup *agentmodel.AgentGroup,
) (*agentmodel.AgentGroup, error) {
	args := m.Called(ctx, namespace, name, agentGroup)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	result, ok := args.Get(0).(*agentmodel.AgentGroup)
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentGroupPersistencePort) ListAgentGroups(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentGroup], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	result, ok := args.Get(0).(*model.ListResponse[*agentmodel.AgentGroup])
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
) (*agentmodel.Agent, error) {
	args := m.Called(ctx, instanceUID)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	agnt, ok := args.Get(0).(*agentmodel.Agent)
	if !ok {
		return nil, errUnexpectedType
	}

	return agnt, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentUsecaseForGroup) GetOrCreateAgent(
	ctx context.Context,
	instanceUID uuid.UUID,
) (*agentmodel.Agent, error) {
	args := m.Called(ctx, instanceUID)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	agnt, ok := args.Get(0).(*agentmodel.Agent)
	if !ok {
		return nil, errUnexpectedType
	}

	return agnt, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentUsecaseForGroup) ListAgentsBySelector(
	ctx context.Context,
	selector agentmodel.AgentSelector,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	args := m.Called(ctx, selector, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	result, ok := args.Get(0).(*model.ListResponse[*agentmodel.Agent])
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentUsecaseForGroup) SaveAgent(ctx context.Context, agnt *agentmodel.Agent) error {
	args := m.Called(ctx, agnt)

	return args.Error(0) //nolint:wrapcheck // mock error
}

func (m *MockAgentUsecaseForGroup) ListAgents(
	ctx context.Context,
	namespace string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	args := m.Called(ctx, namespace, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	result, ok := args.Get(0).(*model.ListResponse[*agentmodel.Agent])
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentUsecaseForGroup) SearchAgents(
	ctx context.Context,
	namespace string,
	query string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	args := m.Called(ctx, namespace, query, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	result, ok := args.Get(0).(*model.ListResponse[*agentmodel.Agent])
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
	namespace string,
	name string,
) (*agentmodel.AgentRemoteConfig, error) {
	args := m.Called(ctx, namespace, name)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	result, ok := args.Get(0).(*agentmodel.AgentRemoteConfig)
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentRemoteConfigPersistencePort) PutAgentRemoteConfig(
	ctx context.Context,
	config *agentmodel.AgentRemoteConfig,
) (*agentmodel.AgentRemoteConfig, error) {
	args := m.Called(ctx, config)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	result, ok := args.Get(0).(*agentmodel.AgentRemoteConfig)
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockAgentRemoteConfigPersistencePort) ListAgentRemoteConfigs(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentRemoteConfig], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	result, ok := args.Get(0).(*model.ListResponse[*agentmodel.AgentRemoteConfig])
	if !ok {
		return nil, errUnexpectedType
	}

	return result, args.Error(1) //nolint:wrapcheck // mock error
}

// MockCertificatePersistencePortForGroup is a mock implementation of CertificatePersistencePort for agentgroup tests.
type MockCertificatePersistencePortForGroup struct {
	mock.Mock
}

func (m *MockCertificatePersistencePortForGroup) GetCertificate(
	ctx context.Context,
	namespace string,
	name string,
) (*agentmodel.Certificate, error) {
	args := m.Called(ctx, namespace, name)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	cert, ok := args.Get(0).(*agentmodel.Certificate)
	if !ok {
		return nil, errUnexpectedType
	}

	return cert, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockCertificatePersistencePortForGroup) PutCertificate(
	ctx context.Context,
	certificate *agentmodel.Certificate,
) (*agentmodel.Certificate, error) {
	args := m.Called(ctx, certificate)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	cert, ok := args.Get(0).(*agentmodel.Certificate)
	if !ok {
		return nil, errUnexpectedType
	}

	return cert, args.Error(1) //nolint:wrapcheck // mock error
}

func (m *MockCertificatePersistencePortForGroup) ListCertificate(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Certificate], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck // mock error
	}

	resp, ok := args.Get(0).(*model.ListResponse[*agentmodel.Certificate])
	if !ok {
		return nil, errUnexpectedType
	}

	return resp, args.Error(1) //nolint:wrapcheck // mock error
}

func TestAgentGroupService_GetAgentGroup(t *testing.T) {
	t.Parallel()

	t.Run("Successfully get agent group", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPersistence := new(MockAgentGroupPersistencePort)
		mockAgentUsecase := new(MockAgentUsecaseForGroup)
		mockRemoteConfigPort := new(MockAgentRemoteConfigPersistencePort)
		logger := slog.Default()

		mockCertPersistence := new(MockCertificatePersistencePortForGroup)
		svc := agentservice.NewAgentGroupService(
			mockPersistence, mockRemoteConfigPort, mockCertPersistence, mockAgentUsecase, logger)

		expectedGroup := &agentmodel.AgentGroup{
			Metadata: agentmodel.AgentGroupMetadata{
				Name: "test-group",
			},
		}

		mockPersistence.On("GetAgentGroup", ctx, "default", "test-group", (*model.GetOptions)(nil)).Return(expectedGroup, nil)

		result, err := svc.GetAgentGroup(ctx, "default", "test-group", nil)

		require.NoError(t, err)
		assert.Equal(t, "test-group", result.Metadata.Name)
		mockPersistence.AssertExpectations(t)
	})

	t.Run("Error when agent group not found", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockPersistence := new(MockAgentGroupPersistencePort)
		mockAgentUsecase := new(MockAgentUsecaseForGroup)
		mockRemoteConfigPort := new(MockAgentRemoteConfigPersistencePort)
		logger := slog.Default()

		mockCertPersistence := new(MockCertificatePersistencePortForGroup)
		svc := agentservice.NewAgentGroupService(
			mockPersistence, mockRemoteConfigPort, mockCertPersistence, mockAgentUsecase, logger)

		mockPersistence.On(
			"GetAgentGroup", ctx, "default", "non-existent", (*model.GetOptions)(nil),
		).Return(nil, errAgentGroupNotFound)

		result, err := svc.GetAgentGroup(ctx, "default", "non-existent", nil)

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

		ctx := t.Context()
		mockPersistence := new(MockAgentGroupPersistencePort)
		mockAgentUsecase := new(MockAgentUsecaseForGroup)
		mockRemoteConfigPort := new(MockAgentRemoteConfigPersistencePort)
		logger := slog.Default()

		mockCertPersistence := new(MockCertificatePersistencePortForGroup)
		svc := agentservice.NewAgentGroupService(
			mockPersistence, mockRemoteConfigPort, mockCertPersistence, mockAgentUsecase, logger)

		expectedResponse := &model.ListResponse[*agentmodel.AgentGroup]{
			Items: []*agentmodel.AgentGroup{
				{Metadata: agentmodel.AgentGroupMetadata{Name: "group-1"}},
				{Metadata: agentmodel.AgentGroupMetadata{Name: "group-2"}},
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

		ctx := t.Context()
		mockPersistence := new(MockAgentGroupPersistencePort)
		mockAgentUsecase := new(MockAgentUsecaseForGroup)
		mockRemoteConfigPort := new(MockAgentRemoteConfigPersistencePort)
		logger := slog.Default()

		mockCertPersistence := new(MockCertificatePersistencePortForGroup)
		svc := agentservice.NewAgentGroupService(
			mockPersistence, mockRemoteConfigPort, mockCertPersistence, mockAgentUsecase, logger)

		agentGroup := &agentmodel.AgentGroup{
			Metadata: agentmodel.AgentGroupMetadata{
				Name: "test-group",
			},
			Spec: agentmodel.AgentGroupSpec{
				Selector: agentmodel.AgentSelector{
					IdentifyingAttributes: map[string]string{
						"service.name": "test-service",
					},
				},
			},
		}

		agent1 := agentmodel.NewAgent(uuid.New(), agentmodel.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{
				"service.name": "test-service",
			},
		}))

		expectedResponse := &model.ListResponse[*agentmodel.Agent]{
			Items:              []*agentmodel.Agent{agent1},
			Continue:           "",
			RemainingItemCount: 0,
		}

		options := &model.ListOptions{Limit: 10}
		mockAgentUsecase.On("ListAgentsBySelector", ctx, agentGroup.Spec.Selector, options).
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

		ctx := t.Context()
		mockPersistence := new(MockAgentGroupPersistencePort)
		mockAgentUsecase := new(MockAgentUsecaseForGroup)
		mockRemoteConfigPort := new(MockAgentRemoteConfigPersistencePort)
		logger := slog.Default()

		mockCertPersistence := new(MockCertificatePersistencePortForGroup)
		svc := agentservice.NewAgentGroupService(
			mockPersistence, mockRemoteConfigPort, mockCertPersistence, mockAgentUsecase, logger)

		testAgent := agentmodel.NewAgent(uuid.New(), agentmodel.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{
				"service.name": "my-service",
			},
			NonIdentifyingAttributes: map[string]string{
				"os.type": "linux",
			},
		}))

		matchingGroup := &agentmodel.AgentGroup{
			Metadata: agentmodel.AgentGroupMetadata{
				Name: "matching-group",
			},
			Spec: agentmodel.AgentGroupSpec{
				Selector: agentmodel.AgentSelector{
					IdentifyingAttributes: map[string]string{
						"service.name": "my-service",
					},
				},
			},
		}

		nonMatchingGroup := &agentmodel.AgentGroup{
			Metadata: agentmodel.AgentGroupMetadata{
				Name: "non-matching-group",
			},
			Spec: agentmodel.AgentGroupSpec{
				Selector: agentmodel.AgentSelector{
					IdentifyingAttributes: map[string]string{
						"service.name": "other-service",
					},
				},
			},
		}

		allGroups := &model.ListResponse[*agentmodel.AgentGroup]{
			Items:              []*agentmodel.AgentGroup{matchingGroup, nonMatchingGroup},
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

		ctx := t.Context()
		mockPersistence := new(MockAgentGroupPersistencePort)
		mockAgentUsecase := new(MockAgentUsecaseForGroup)
		mockRemoteConfigPort := new(MockAgentRemoteConfigPersistencePort)
		logger := slog.Default()

		mockCertPersistence := new(MockCertificatePersistencePortForGroup)
		svc := agentservice.NewAgentGroupService(
			mockPersistence, mockRemoteConfigPort, mockCertPersistence, mockAgentUsecase, logger)

		testAgent := agentmodel.NewAgent(uuid.New(), agentmodel.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{
				"service.name": "unique-service",
			},
		}))

		nonMatchingGroup := &agentmodel.AgentGroup{
			Metadata: agentmodel.AgentGroupMetadata{
				Name: "non-matching-group",
			},
			Spec: agentmodel.AgentGroupSpec{
				Selector: agentmodel.AgentSelector{
					IdentifyingAttributes: map[string]string{
						"service.name": "other-service",
					},
				},
			},
		}

		allGroups := &model.ListResponse[*agentmodel.AgentGroup]{
			Items:              []*agentmodel.AgentGroup{nonMatchingGroup},
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

	mockCertPersistence := new(MockCertificatePersistencePortForGroup)
	svc := agentservice.NewAgentGroupService(
		mockPersistence, mockRemoteConfigPort, mockCertPersistence, mockAgentUsecase, logger)

	assert.Equal(t, "AgentGroupService", svc.Name())
}
