package service

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
)

var errRemoteConfigNotFound = errors.New("remote config not found")

// mockAgentGroupPersistence is a mock for AgentGroupPersistencePort.
type mockAgentGroupPersistence struct {
	mock.Mock
}

func (m *mockAgentGroupPersistence) GetAgentGroup(ctx context.Context, name string) (*model.AgentGroup, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	return args.Get(0).(*model.AgentGroup), args.Error(1) //nolint:wrapcheck
}

func (m *mockAgentGroupPersistence) PutAgentGroup(
	ctx context.Context,
	name string,
	ag *model.AgentGroup,
) (*model.AgentGroup, error) {
	args := m.Called(ctx, name, ag)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	return args.Get(0).(*model.AgentGroup), args.Error(1) //nolint:wrapcheck
}

func (m *mockAgentGroupPersistence) ListAgentGroups(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.AgentGroup], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	return args.Get(0).(*model.ListResponse[*model.AgentGroup]), args.Error(1) //nolint:wrapcheck
}

// mockAgentUsecase is a mock for AgentUsecase.
type mockAgentUsecase struct {
	mock.Mock
}

func (m *mockAgentUsecase) GetAgent(ctx context.Context, uid uuid.UUID) (*model.Agent, error) {
	args := m.Called(ctx, uid)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	return args.Get(0).(*model.Agent), args.Error(1) //nolint:wrapcheck
}

func (m *mockAgentUsecase) GetOrCreateAgent(ctx context.Context, uid uuid.UUID) (*model.Agent, error) {
	args := m.Called(ctx, uid)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	return args.Get(0).(*model.Agent), args.Error(1) //nolint:wrapcheck
}

func (m *mockAgentUsecase) ListAgentsBySelector(
	ctx context.Context,
	selector model.AgentSelector,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	args := m.Called(ctx, selector, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	return args.Get(0).(*model.ListResponse[*model.Agent]), args.Error(1) //nolint:wrapcheck
}

func (m *mockAgentUsecase) SaveAgent(ctx context.Context, a *model.Agent) error {
	args := m.Called(ctx, a)

	return args.Error(0) //nolint:wrapcheck
}

func (m *mockAgentUsecase) ListAgents(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	return args.Get(0).(*model.ListResponse[*model.Agent]), args.Error(1) //nolint:wrapcheck
}

func (m *mockAgentUsecase) SearchAgents(
	ctx context.Context,
	query string,
	options *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	args := m.Called(ctx, query, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	return args.Get(0).(*model.ListResponse[*model.Agent]), args.Error(1) //nolint:wrapcheck
}

// mockRemoteConfigPersistence is a mock for AgentRemoteConfigPersistencePort.
type mockRemoteConfigPersistence struct {
	mock.Mock
}

func (m *mockRemoteConfigPersistence) GetAgentRemoteConfig(
	ctx context.Context,
	name string,
) (*model.AgentRemoteConfig, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	return args.Get(0).(*model.AgentRemoteConfig), args.Error(1) //nolint:wrapcheck
}

func (m *mockRemoteConfigPersistence) PutAgentRemoteConfig(
	ctx context.Context,
	config *model.AgentRemoteConfig,
) (*model.AgentRemoteConfig, error) {
	args := m.Called(ctx, config)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	return args.Get(0).(*model.AgentRemoteConfig), args.Error(1) //nolint:wrapcheck
}

func (m *mockRemoteConfigPersistence) ListAgentRemoteConfigs(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.AgentRemoteConfig], error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}

	return args.Get(0).(*model.ListResponse[*model.AgentRemoteConfig]), args.Error(1) //nolint:wrapcheck
}

func TestResolveRemoteConfig_RefMode(t *testing.T) {
	t.Parallel()

	t.Run("Successfully resolves referenced AgentRemoteConfig", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockPersistence := new(mockAgentGroupPersistence)
		mockAgentUC := new(mockAgentUsecase)
		mockRemoteConfigPort := new(mockRemoteConfigPersistence)
		logger := slog.Default()

		svc := NewAgentGroupService(mockPersistence, mockAgentUC, logger)
		svc.AgentRemoteConfigPersistencePort = mockRemoteConfigPort

		refName := "shared-otel-config"
		referencedConfig := &model.AgentRemoteConfig{
			Metadata: model.AgentRemoteConfigMetadata{
				Name: refName,
			},
			Spec: model.AgentRemoteConfigSpec{
				Value:       []byte("receivers:\n  otlp:\n    protocols:\n      grpc:"),
				ContentType: "application/yaml",
			},
		}

		mockRemoteConfigPort.On("GetAgentRemoteConfig", ctx, refName).Return(referencedConfig, nil)

		remoteConfig := model.AgentGroupAgentRemoteConfig{
			AgentRemoteConfigRef: &refName,
		}

		configFile, configName, err := svc.resolveRemoteConfig(ctx, "test-group", remoteConfig)

		require.NoError(t, err)
		assert.Equal(t, refName, configName) // No prefix for refs
		assert.Equal(t, referencedConfig.Spec.Value, configFile.Body)
		assert.Equal(t, referencedConfig.Spec.ContentType, configFile.ContentType)
		mockRemoteConfigPort.AssertExpectations(t)
	})

	t.Run("Returns error when referenced config not found", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockPersistence := new(mockAgentGroupPersistence)
		mockAgentUC := new(mockAgentUsecase)
		mockRemoteConfigPort := new(mockRemoteConfigPersistence)
		logger := slog.Default()

		svc := NewAgentGroupService(mockPersistence, mockAgentUC, logger)
		svc.AgentRemoteConfigPersistencePort = mockRemoteConfigPort

		refName := "non-existent-config"
		mockRemoteConfigPort.On("GetAgentRemoteConfig", ctx, refName).Return(nil, errRemoteConfigNotFound)

		remoteConfig := model.AgentGroupAgentRemoteConfig{
			AgentRemoteConfigRef: &refName,
		}

		_, _, err := svc.resolveRemoteConfig(ctx, "test-group", remoteConfig)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "get agent remote config")
		mockRemoteConfigPort.AssertExpectations(t)
	})
}

func TestResolveRemoteConfig_DirectMode(t *testing.T) {
	t.Parallel()

	t.Run("Inline config gets AgentGroupName prefix", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockPersistence := new(mockAgentGroupPersistence)
		mockAgentUC := new(mockAgentUsecase)
		logger := slog.Default()

		svc := NewAgentGroupService(mockPersistence, mockAgentUC, logger)

		configName := "collector-config"
		configValue := []byte("exporters:\n  debug:\n    verbosity: detailed")
		contentType := "application/yaml"

		remoteConfig := model.AgentGroupAgentRemoteConfig{
			AgentRemoteConfigName: &configName,
			AgentRemoteConfigSpec: &model.AgentRemoteConfigSpec{
				Value:       configValue,
				ContentType: contentType,
			},
		}

		configFile, resolvedName, err := svc.resolveRemoteConfig(ctx, "staging-group", remoteConfig)

		require.NoError(t, err)
		// Config name should be prefixed with AgentGroupName
		assert.Equal(t, "staging-group/collector-config", resolvedName)
		assert.Equal(t, configValue, configFile.Body)
		assert.Equal(t, contentType, configFile.ContentType)
	})

	t.Run("Returns error when spec is nil", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockPersistence := new(mockAgentGroupPersistence)
		mockAgentUC := new(mockAgentUsecase)
		logger := slog.Default()

		svc := NewAgentGroupService(mockPersistence, mockAgentUC, logger)

		configName := "missing-spec-config"
		remoteConfig := model.AgentGroupAgentRemoteConfig{
			AgentRemoteConfigName: &configName,
			AgentRemoteConfigSpec: nil, // Missing spec
		}

		_, _, err := svc.resolveRemoteConfig(ctx, "test-group", remoteConfig)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid remote config")
	})

	t.Run("Returns error when name is nil", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockPersistence := new(mockAgentGroupPersistence)
		mockAgentUC := new(mockAgentUsecase)
		logger := slog.Default()

		svc := NewAgentGroupService(mockPersistence, mockAgentUC, logger)

		remoteConfig := model.AgentGroupAgentRemoteConfig{
			AgentRemoteConfigName: nil, // Missing name
			AgentRemoteConfigSpec: &model.AgentRemoteConfigSpec{
				Value:       []byte("some config"),
				ContentType: "text/plain",
			},
		}

		_, _, err := svc.resolveRemoteConfig(ctx, "test-group", remoteConfig)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid remote config")
	})
}

func TestApplyRemoteConfigs(t *testing.T) {
	t.Parallel()

	t.Run("Applies ref config to agent without prefix", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockPersistence := new(mockAgentGroupPersistence)
		mockAgentUC := new(mockAgentUsecase)
		mockRemoteConfigPort := new(mockRemoteConfigPersistence)
		logger := slog.Default()

		svc := NewAgentGroupService(mockPersistence, mockAgentUC, logger)
		svc.AgentRemoteConfigPersistencePort = mockRemoteConfigPort

		testAgent := model.NewAgent(uuid.New(), model.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{"service.name": "test"},
		}))

		refName := "global-config"
		referencedConfig := &model.AgentRemoteConfig{
			Metadata: model.AgentRemoteConfigMetadata{Name: refName},
			Spec: model.AgentRemoteConfigSpec{
				Value:       []byte("global config content"),
				ContentType: "text/plain",
			},
		}

		agentGroup := &model.AgentGroup{
			Metadata: model.AgentGroupMetadata{Name: "production"},
			Spec: model.AgentGroupSpec{
				AgentRemoteConfigs: []model.AgentGroupAgentRemoteConfig{
					{AgentRemoteConfigRef: &refName},
				},
			},
		}

		mockRemoteConfigPort.On("GetAgentRemoteConfig", ctx, refName).Return(referencedConfig, nil)

		err := svc.applyRemoteConfigs(ctx, agentGroup, testAgent)

		require.NoError(t, err)
		// Verify config was applied with original name (no prefix)
		configFile, exists := testAgent.Spec.RemoteConfig.ConfigMap.ConfigMap[refName]
		assert.True(t, exists, "Config should be applied with original name")
		assert.Equal(t, referencedConfig.Spec.Value, configFile.Body)
		mockRemoteConfigPort.AssertExpectations(t)
	})

	t.Run("Applies inline config to agent with AgentGroupName prefix", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockPersistence := new(mockAgentGroupPersistence)
		mockAgentUC := new(mockAgentUsecase)
		logger := slog.Default()

		svc := NewAgentGroupService(mockPersistence, mockAgentUC, logger)

		testAgent := model.NewAgent(uuid.New(), model.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{"service.name": "test"},
		}))

		inlineName := "local-config"
		inlineValue := []byte("local config content")
		agentGroup := &model.AgentGroup{
			Metadata: model.AgentGroupMetadata{Name: "staging"},
			Spec: model.AgentGroupSpec{
				AgentRemoteConfigs: []model.AgentGroupAgentRemoteConfig{
					{
						AgentRemoteConfigName: &inlineName,
						AgentRemoteConfigSpec: &model.AgentRemoteConfigSpec{
							Value:       inlineValue,
							ContentType: "text/plain",
						},
					},
				},
			},
		}

		err := svc.applyRemoteConfigs(ctx, agentGroup, testAgent)

		require.NoError(t, err)
		// Verify config was applied with prefixed name
		expectedName := "staging/local-config"
		configFile, exists := testAgent.Spec.RemoteConfig.ConfigMap.ConfigMap[expectedName]
		assert.True(t, exists, "Config should be applied with prefixed name: %s", expectedName)
		assert.Equal(t, inlineValue, configFile.Body)
	})
}

func TestNameCollisionPrevention(t *testing.T) {
	t.Parallel()

	t.Run("Same config name in different groups produces different keys", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockPersistence := new(mockAgentGroupPersistence)
		mockAgentUC := new(mockAgentUsecase)
		logger := slog.Default()

		svc := NewAgentGroupService(mockPersistence, mockAgentUC, logger)

		// Create agent
		testAgent := model.NewAgent(uuid.New(), model.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{"service.name": "test"},
		}))

		configName := "config" // Same name used in both groups

		// Group Alpha
		groupAlpha := &model.AgentGroup{
			Metadata: model.AgentGroupMetadata{Name: "group-alpha"},
			Spec: model.AgentGroupSpec{
				AgentRemoteConfigs: []model.AgentGroupAgentRemoteConfig{
					{
						AgentRemoteConfigName: &configName,
						AgentRemoteConfigSpec: &model.AgentRemoteConfigSpec{
							Value:       []byte("content from alpha"),
							ContentType: "text/plain",
						},
					},
				},
			},
		}

		// Group Beta
		groupBeta := &model.AgentGroup{
			Metadata: model.AgentGroupMetadata{Name: "group-beta"},
			Spec: model.AgentGroupSpec{
				AgentRemoteConfigs: []model.AgentGroupAgentRemoteConfig{
					{
						AgentRemoteConfigName: &configName,
						AgentRemoteConfigSpec: &model.AgentRemoteConfigSpec{
							Value:       []byte("content from beta"),
							ContentType: "text/plain",
						},
					},
				},
			},
		}

		// Apply configs from both groups
		err := svc.applyRemoteConfigs(ctx, groupAlpha, testAgent)
		require.NoError(t, err)

		err = svc.applyRemoteConfigs(ctx, groupBeta, testAgent)
		require.NoError(t, err)

		// Verify both configs exist with different prefixed names
		alphaConfig, alphaExists := testAgent.Spec.RemoteConfig.ConfigMap.ConfigMap["group-alpha/config"]
		betaConfig, betaExists := testAgent.Spec.RemoteConfig.ConfigMap.ConfigMap["group-beta/config"]

		assert.True(t, alphaExists, "Alpha config should exist")
		assert.True(t, betaExists, "Beta config should exist")
		assert.Equal(t, []byte("content from alpha"), alphaConfig.Body)
		assert.Equal(t, []byte("content from beta"), betaConfig.Body)

		// Verify we have exactly 2 configs (no collision)
		assert.Len(t, testAgent.Spec.RemoteConfig.ConfigMap.ConfigMap, 2)
	})
}

func TestUpdateAgentsByAgentGroup(t *testing.T) {
	t.Parallel()

	t.Run("Full propagation flow with ref config", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockPersistence := new(mockAgentGroupPersistence)
		mockAgentUC := new(mockAgentUsecase)
		mockRemoteConfigPort := new(mockRemoteConfigPersistence)
		logger := slog.Default()

		svc := NewAgentGroupService(mockPersistence, mockAgentUC, logger)
		svc.AgentRemoteConfigPersistencePort = mockRemoteConfigPort

		testAgent := model.NewAgent(uuid.New(), model.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{"service.name": "my-service"},
		}))

		refName := "shared-config"
		referencedConfig := &model.AgentRemoteConfig{
			Metadata: model.AgentRemoteConfigMetadata{Name: refName},
			Spec: model.AgentRemoteConfigSpec{
				Value:       []byte("shared config content"),
				ContentType: "application/yaml",
			},
		}

		agentGroup := &model.AgentGroup{
			Metadata: model.AgentGroupMetadata{
				Name: "production",
				Selector: model.AgentSelector{
					IdentifyingAttributes: map[string]string{"service.name": "my-service"},
				},
			},
			Spec: model.AgentGroupSpec{
				AgentRemoteConfigs: []model.AgentGroupAgentRemoteConfig{
					{AgentRemoteConfigRef: &refName},
				},
			},
		}

		agentsResponse := &model.ListResponse[*model.Agent]{
			Items:              []*model.Agent{testAgent},
			Continue:           "",
			RemainingItemCount: 0,
		}

		mockAgentUC.On("ListAgentsBySelector", ctx, agentGroup.Metadata.Selector, mock.Anything).
			Return(agentsResponse, nil)
		mockRemoteConfigPort.On("GetAgentRemoteConfig", ctx, refName).Return(referencedConfig, nil)
		mockAgentUC.On("SaveAgent", ctx, mock.MatchedBy(func(a *model.Agent) bool {
			_, exists := a.Spec.RemoteConfig.ConfigMap.ConfigMap[refName]

			return exists
		})).Return(nil)

		err := svc.updateAgentsByAgentGroup(ctx, agentGroup)

		require.NoError(t, err)
		mockAgentUC.AssertExpectations(t)
		mockRemoteConfigPort.AssertExpectations(t)
	})

	t.Run("Full propagation flow with inline config", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		mockPersistence := new(mockAgentGroupPersistence)
		mockAgentUC := new(mockAgentUsecase)
		logger := slog.Default()

		svc := NewAgentGroupService(mockPersistence, mockAgentUC, logger)

		testAgent := model.NewAgent(uuid.New(), model.WithDescription(&agent.Description{
			IdentifyingAttributes: map[string]string{"service.name": "my-service"},
		}))

		inlineName := "inline-config"
		agentGroup := &model.AgentGroup{
			Metadata: model.AgentGroupMetadata{
				Name: "staging",
				Selector: model.AgentSelector{
					IdentifyingAttributes: map[string]string{"service.name": "my-service"},
				},
			},
			Spec: model.AgentGroupSpec{
				AgentRemoteConfigs: []model.AgentGroupAgentRemoteConfig{
					{
						AgentRemoteConfigName: &inlineName,
						AgentRemoteConfigSpec: &model.AgentRemoteConfigSpec{
							Value:       []byte("inline config content"),
							ContentType: "text/plain",
						},
					},
				},
			},
		}

		agentsResponse := &model.ListResponse[*model.Agent]{
			Items:              []*model.Agent{testAgent},
			Continue:           "",
			RemainingItemCount: 0,
		}

		mockAgentUC.On("ListAgentsBySelector", ctx, agentGroup.Metadata.Selector, mock.Anything).
			Return(agentsResponse, nil)
		mockAgentUC.On("SaveAgent", ctx, mock.MatchedBy(func(a *model.Agent) bool {
			// Verify the config has the prefixed name
			_, exists := a.Spec.RemoteConfig.ConfigMap.ConfigMap["staging/inline-config"]

			return exists
		})).Return(nil)

		err := svc.updateAgentsByAgentGroup(ctx, agentGroup)

		require.NoError(t, err)
		mockAgentUC.AssertExpectations(t)
	})
}
